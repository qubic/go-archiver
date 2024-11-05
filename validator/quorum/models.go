package quorum

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-node-connector/types"
	"time"
)

func qubicToProto(votes types.QuorumVotes) *protobuff.QuorumTickData {
	firstQuorumTickData := votes[0]
	protoQuorumTickData := protobuff.QuorumTickData{
		QuorumTickStructure:   qubicTickStructureToProto(firstQuorumTickData),
		QuorumDiffPerComputor: make(map[uint32]*protobuff.QuorumDiff),
	}

	for _, quorumTickData := range votes {
		protoQuorumTickData.QuorumDiffPerComputor[uint32(quorumTickData.ComputorIndex)] = qubicDiffToProto(quorumTickData)
	}

	return &protoQuorumTickData
}

func qubicTickStructureToProto(tickVote types.QuorumTickVote) *protobuff.QuorumTickStructure {
	date := time.Date(2000+int(tickVote.Year), time.Month(tickVote.Month), int(tickVote.Day), int(tickVote.Hour), int(tickVote.Minute), int(tickVote.Second), 0, time.UTC)
	timestamp := date.UnixMilli() + int64(tickVote.Millisecond)
	protoQuorumTickStructure := protobuff.QuorumTickStructure{
		Epoch:                        uint32(tickVote.Epoch),
		TickNumber:                   tickVote.Tick,
		Timestamp:                    uint64(timestamp),
		PrevResourceTestingDigestHex: convertUint64ToHex(tickVote.PreviousResourceTestingDigest),
		PrevSpectrumDigestHex:        hex.EncodeToString(tickVote.PreviousSpectrumDigest[:]),
		PrevUniverseDigestHex:        hex.EncodeToString(tickVote.PreviousUniverseDigest[:]),
		PrevComputerDigestHex:        hex.EncodeToString(tickVote.PreviousComputerDigest[:]),
		TxDigestHex:                  hex.EncodeToString(tickVote.TxDigest[:]),
	}

	return &protoQuorumTickStructure
}

func qubicDiffToProto(tickVote types.QuorumTickVote) *protobuff.QuorumDiff {
	protoQuorumDiff := protobuff.QuorumDiff{
		SaltedResourceTestingDigestHex: convertUint64ToHex(tickVote.SaltedResourceTestingDigest),
		SaltedSpectrumDigestHex:        hex.EncodeToString(tickVote.SaltedSpectrumDigest[:]),
		SaltedUniverseDigestHex:        hex.EncodeToString(tickVote.SaltedUniverseDigest[:]),
		SaltedComputerDigestHex:        hex.EncodeToString(tickVote.SaltedComputerDigest[:]),
		ExpectedNextTickTxDigestHex:    hex.EncodeToString(tickVote.ExpectedNextTickTxDigest[:]),
		SignatureHex:                   hex.EncodeToString(tickVote.Signature[:]),
	}
	return &protoQuorumDiff
}

func convertUint64ToHex(value uint64) string {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, value)
	return hex.EncodeToString(b)
}

func qubicToProtoStored(votes types.QuorumVotes) *protobuff.QuorumTickDataStored {
	firstQuorumTickData := votes[0]
	protoQuorumTickData := protobuff.QuorumTickDataStored{
		QuorumTickStructure:   qubicTickStructureToProto(firstQuorumTickData),
		QuorumDiffPerComputor: make(map[uint32]*protobuff.QuorumDiffStored),
	}

	for _, quorumTickData := range votes {
		protoQuorumTickData.QuorumDiffPerComputor[uint32(quorumTickData.ComputorIndex)] = qubicDiffToProtoStored(quorumTickData)
	}

	return &protoQuorumTickData
}

func qubicDiffToProtoStored(tickVote types.QuorumTickVote) *protobuff.QuorumDiffStored {
	protoQuorumDiff := protobuff.QuorumDiffStored{
		ExpectedNextTickTxDigestHex: hex.EncodeToString(tickVote.ExpectedNextTickTxDigest[:]),
		SignatureHex:                hex.EncodeToString(tickVote.Signature[:]),
	}
	return &protoQuorumDiff
}

func ReconstructQuorumData(currentTickQuorumData, nextTickQuorumData *protobuff.QuorumTickDataStored, computors *protobuff.Computors) (*protobuff.QuorumTickData, error) {

	reconstructedQuorumData := protobuff.QuorumTickData{
		QuorumTickStructure:   currentTickQuorumData.QuorumTickStructure,
		QuorumDiffPerComputor: make(map[uint32]*protobuff.QuorumDiff),
	}

	spectrumDigest, err := hex.DecodeString(nextTickQuorumData.QuorumTickStructure.PrevSpectrumDigestHex)
	if err != nil {
		return nil, errors.Wrap(err, "obtaining spectrum digest from next tick quorum data")
	}
	universeDigest, err := hex.DecodeString(nextTickQuorumData.QuorumTickStructure.PrevUniverseDigestHex)
	if err != nil {
		return nil, errors.Wrap(err, "obtaining universe digest from next tick quorum data")
	}
	computerDigest, err := hex.DecodeString(nextTickQuorumData.QuorumTickStructure.PrevComputerDigestHex)
	if err != nil {
		return nil, errors.Wrap(err, "obtaining computer digest from next tick quorum data")
	}
	resourceDigest, err := hex.DecodeString(nextTickQuorumData.QuorumTickStructure.PrevResourceTestingDigestHex)
	if err != nil {
		return nil, errors.Wrap(err, "obtaining resource testing digest from next tick quorum data")
	}

	for id, voteDiff := range currentTickQuorumData.QuorumDiffPerComputor {

		identity := types.Identity(computors.Identities[id])

		computorPublicKey, err := identity.ToPubKey(false)
		if err != nil {
			return nil, errors.Wrapf(err, "obtaining public key for computor id: %d", id)
		}

		var tmp [64]byte
		copy(tmp[:32], computorPublicKey[:])

		copy(tmp[32:], spectrumDigest[:])
		saltedSpectrumDigest, err := utils.K12Hash(tmp[:])
		if err != nil {
			return nil, errors.Wrap(err, "hashing salted spectrum digest")
		}

		copy(tmp[32:], universeDigest[:])
		saltedUniverseDigest, err := utils.K12Hash(tmp[:])
		if err != nil {
			return nil, errors.Wrap(err, "hashing salted universe digest")
		}

		copy(tmp[32:], computerDigest[:])
		saltedComputerDigest, err := utils.K12Hash(tmp[:])
		if err != nil {
			return nil, errors.Wrap(err, "hashing salted computer digest")
		}

		var tmp2 [40]byte
		copy(tmp2[:32], computorPublicKey[:])
		copy(tmp2[32:], resourceDigest[:])
		saltedResourceTestingDigest, err := utils.K12Hash(tmp2[:])
		if err != nil {
			return nil, errors.Wrap(err, "hashing salted resource testing digest")
		}

		reconstructedQuorumData.QuorumDiffPerComputor[id] = &protobuff.QuorumDiff{
			SaltedResourceTestingDigestHex: hex.EncodeToString(saltedResourceTestingDigest[:8]),
			SaltedSpectrumDigestHex:        hex.EncodeToString(saltedSpectrumDigest[:]),
			SaltedUniverseDigestHex:        hex.EncodeToString(saltedUniverseDigest[:]),
			SaltedComputerDigestHex:        hex.EncodeToString(saltedComputerDigest[:]),
			ExpectedNextTickTxDigestHex:    voteDiff.ExpectedNextTickTxDigestHex,
			SignatureHex:                   voteDiff.SignatureHex,
		}
	}

	return &reconstructedQuorumData, nil
}
