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
		Epoch:      uint32(tickVote.Epoch),
		TickNumber: tickVote.Tick,
		Timestamp:  uint64(timestamp),

		PrevResourceTestingDigestHex: convertUint32ToHex(tickVote.PreviousResourceTestingDigest),
		PrevTransactionBodyHex:       convertUint32ToHex(tickVote.PreviousTransactionBodyDigest),

		PrevSpectrumDigestHex: hex.EncodeToString(tickVote.PreviousSpectrumDigest[:]),
		PrevUniverseDigestHex: hex.EncodeToString(tickVote.PreviousUniverseDigest[:]),
		PrevComputerDigestHex: hex.EncodeToString(tickVote.PreviousComputerDigest[:]),
		TxDigestHex:           hex.EncodeToString(tickVote.TxDigest[:]),
	}

	return &protoQuorumTickStructure
}

func qubicDiffToProto(tickVote types.QuorumTickVote) *protobuff.QuorumDiff {
	protoQuorumDiff := protobuff.QuorumDiff{
		SaltedResourceTestingDigestHex: convertUint32ToHex(tickVote.SaltedResourceTestingDigest),
		SaltedTransactionBodyHex:       convertUint32ToHex(tickVote.SaltedTransactionBodyDigest),

		SaltedSpectrumDigestHex:     hex.EncodeToString(tickVote.SaltedSpectrumDigest[:]),
		SaltedUniverseDigestHex:     hex.EncodeToString(tickVote.SaltedUniverseDigest[:]),
		SaltedComputerDigestHex:     hex.EncodeToString(tickVote.SaltedComputerDigest[:]),
		ExpectedNextTickTxDigestHex: hex.EncodeToString(tickVote.ExpectedNextTickTxDigest[:]),
		SignatureHex:                hex.EncodeToString(tickVote.Signature[:]),
	}
	return &protoQuorumDiff
}

func convertUint64ToHex(value uint64) string {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, value)
	return hex.EncodeToString(b)
}

func convertUint32ToHex(value uint32) string {
	b := make([]byte, 4)
	binary.LittleEndian.PutUint32(b, value)
	return hex.EncodeToString(b)
}

func convertHexToUint32(value string) (uint32, error) {
	b, err := hex.DecodeString(value)
	if err != nil {
		return 0, errors.Wrap(err, "decoding string value")
	}
	return binary.LittleEndian.Uint32(b), nil
}

func convertUint32ToBytes(value uint32) [4]byte {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], value)
	return b
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

	epoch := currentTickQuorumData.QuorumTickStructure.Epoch

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

	transactionBodyDigest, err := hex.DecodeString(nextTickQuorumData.QuorumTickStructure.PrevTransactionBodyHex)
	if err != nil {
		return nil, errors.Wrap(err, "obtaining transaction body digest from the next tick quorum data")
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

		newFormat := epoch >= 151

		// Establish length of returned digest based on format
		digestSize := 8
		if newFormat {
			digestSize = 4
		}

		// Create resource testing digest no matter what
		saltedResourceTestingDigestBytes, err := reconstructShortSaltedDigest(computorPublicKey[:], resourceDigest, newFormat)
		if err != nil {
			return nil, errors.Wrap(err, "reconstructing salted resource testing digest")
		}
		saltedResourceTestingDigest := hex.EncodeToString(saltedResourceTestingDigestBytes[:digestSize])

		// Declare transaction body digest no matter what and populate only if new data format
		var saltedTransactionBodyDigest string
		if newFormat {
			saltedTransactionBodyDigestBytes, err := reconstructShortSaltedDigest(computorPublicKey[:], transactionBodyDigest, true)
			if err != nil {
				return nil, errors.Wrap(err, "reconstructing salted transaction body digest")
			}
			saltedTransactionBodyDigest = hex.EncodeToString(saltedTransactionBodyDigestBytes[:digestSize])
		}

		reconstructedQuorumData.QuorumDiffPerComputor[id] = &protobuff.QuorumDiff{
			SaltedResourceTestingDigestHex: saltedResourceTestingDigest,
			SaltedTransactionBodyHex:       saltedTransactionBodyDigest,
			SaltedSpectrumDigestHex:        hex.EncodeToString(saltedSpectrumDigest[:]),
			SaltedUniverseDigestHex:        hex.EncodeToString(saltedUniverseDigest[:]),
			SaltedComputerDigestHex:        hex.EncodeToString(saltedComputerDigest[:]),
			ExpectedNextTickTxDigestHex:    voteDiff.ExpectedNextTickTxDigestHex,
			SignatureHex:                   voteDiff.SignatureHex,
		}
	}

	return &reconstructedQuorumData, nil
}

func reconstructShortSaltedDigest(computorPubkey, prevDigest []byte, newFormat bool) ([32]byte, error) {

	var buf [40]byte
	copy(buf[:32], computorPubkey)
	copy(buf[32:], prevDigest)

	buf2 := buf[:]
	if newFormat {
		buf2 = buf2[:36]
	}

	saltedDigest, err := utils.K12Hash(buf2)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "hashing salted digest")
	}
	return saltedDigest, nil
}
