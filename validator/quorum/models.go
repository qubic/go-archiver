package quorum

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-archiver/validator/tick"
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

func GetQuorumTickData(tickNumber uint32, pebbleStore *store.PebbleStore) (*protobuff.QuorumTickData, error) {
	lastProcessedTick, err := pebbleStore.GetLastProcessedTick(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "getting last processed tick")
	}
	if tickNumber > lastProcessedTick.TickNumber {

		return nil, errors.New(fmt.Sprintf("requested tick number %d is greater than last processed tick %d", tickNumber, lastProcessedTick.TickNumber))
	}

	processedTickIntervalsPerEpoch, err := pebbleStore.GetProcessedTickIntervals(context.Background())
	if err != nil {
		return nil, errors.Wrap(err, "getting processed tick intervals per epoch")
	}

	epoch, err := tick.GetTickEpoch(tickNumber, processedTickIntervalsPerEpoch)
	if err != nil {
		return nil, errors.Wrap(err, "getting tick epoch")
	}

	lastTickFlag, err := tick.IsLastTick(tickNumber, epoch, processedTickIntervalsPerEpoch)
	if err != nil {
		return nil, errors.Wrap(err, "checking if tick is last tick in it's epoch")
	}

	if lastTickFlag {
		lastTickQuorumData, err := pebbleStore.GetLastTickQuorumDataPerEpoch(epoch)
		if err != nil {
			return nil, errors.Wrap(err, "getting quorum data for last processed tick")
		}

		return lastTickQuorumData, nil

	}

	wasSkipped, nextAvailableTick := tick.WasTickSkippedByArchive(tickNumber, processedTickIntervalsPerEpoch)
	if wasSkipped == true {

		return nil, errors.New(fmt.Sprintf("provided tick number %d was skipped by the system, next available tick is %d", tickNumber, nextAvailableTick))
	}

	if tickNumber == lastProcessedTick.TickNumber {
		tickData, err := pebbleStore.GetQuorumTickData(context.Background(), tickNumber)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, errors.Wrap(err, "quorum tick data not found")
			}
			return nil, errors.Wrap(err, "getting quorum tick data")
		}

		quorumTickData := &protobuff.QuorumTickData{
			QuorumTickStructure:   tickData.QuorumTickStructure,
			QuorumDiffPerComputor: make(map[uint32]*protobuff.QuorumDiff),
		}

		for id, diff := range tickData.QuorumDiffPerComputor {
			quorumTickData.QuorumDiffPerComputor[id] = &protobuff.QuorumDiff{
				ExpectedNextTickTxDigestHex: diff.ExpectedNextTickTxDigestHex,
				SignatureHex:                diff.SignatureHex,
			}
		}

		return quorumTickData, nil
	}

	nextTick := tickNumber + 1

	nextTickQuorumData, err := pebbleStore.GetQuorumTickData(context.Background(), nextTick)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, errors.Wrap(err, "quorum data for next tick was not found")
		}
		return nil, errors.Wrap(err, "getting tick data")
	}

	currentTickQuorumData, err := pebbleStore.GetQuorumTickData(context.Background(), tickNumber)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, errors.Wrap(err, "quorum data for  tick was not found")
		}
		return nil, errors.Wrap(err, "getting tick data")
	}

	computors, err := pebbleStore.GetComputors(context.Background(), currentTickQuorumData.QuorumTickStructure.Epoch)
	if err != nil {
		return nil, errors.Wrap(err, "getting computor list")
	}

	reconstructedQuorumData, err := ReconstructQuorumData(currentTickQuorumData, nextTickQuorumData, computors)
	if err != nil {
		return nil, errors.Wrap(err, "reconstructing quorum data")
	}

	return reconstructedQuorumData, nil
}

func convertHexToUint64(value string) (uint64, error) {
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return 0, errors.Wrap(err, "decoding uint64 hex string")
	}

	return binary.LittleEndian.Uint64(decoded), nil
}

func decode32ByteDigestFromString(value string) ([32]byte, error) {
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "decoding 32 byte digest string")
	}

	var returnedValue [32]byte
	copy(returnedValue[:], decoded[:])

	return returnedValue, nil
}

func decode64ByteDigestFromString(value string) ([64]byte, error) {
	decoded, err := hex.DecodeString(value)
	if err != nil {
		return [64]byte{}, errors.Wrap(err, "decoding 64 byte digest string")
	}

	var returnedValue [64]byte
	copy(returnedValue[:], decoded[:])

	return returnedValue, nil
}

func ProtoToQubic(quorumData *protobuff.QuorumTickData) (types.QuorumVotes, error) {

	votes := types.QuorumVotes{}

	tickStructure := quorumData.QuorumTickStructure

	tickTime := time.UnixMilli(int64(tickStructure.Timestamp)).UTC()
	tickTimeNoMilli := time.Date(tickTime.Year(), tickTime.Month(), tickTime.Day(), tickTime.Hour(), tickTime.Minute(), tickTime.Second(), 0, time.UTC)
	milli := tickTime.UnixMilli() - tickTimeNoMilli.UnixMilli()

	prevResourceTestingDigest, err := convertHexToUint64(tickStructure.PrevResourceTestingDigestHex)
	if err != nil {
		return nil, errors.Wrap(err, "decoding previous resource testing digest")
	}
	prevSpectrumDigest, err := decode32ByteDigestFromString(tickStructure.PrevSpectrumDigestHex)
	if err != nil {
		return nil, errors.Wrap(err, "decoding previous spectrum digest")
	}
	prevUniverseDigest, err := decode32ByteDigestFromString(tickStructure.PrevUniverseDigestHex)
	if err != nil {
		return nil, errors.Wrap(err, "decoding previous universe digest")
	}
	prevComputerDigest, err := decode32ByteDigestFromString(tickStructure.PrevComputerDigestHex)
	if err != nil {
		return nil, errors.Wrap(err, "decoding previous computer digest")
	}
	txDigest, err := decode32ByteDigestFromString(tickStructure.TxDigestHex)
	if err != nil {
		return nil, errors.Wrap(err, "decoding tx digest")
	}

	for index, diff := range quorumData.QuorumDiffPerComputor {

		saltedResourceTestingDigest, err := convertHexToUint64(diff.SaltedResourceTestingDigestHex)
		if err != nil {
			return nil, errors.Wrap(err, "decoding salted resource testing digest")
		}
		saltedSpectrumDigest, err := decode32ByteDigestFromString(diff.SaltedSpectrumDigestHex)
		if err != nil {
			return nil, errors.Wrap(err, "decoding salted spectrum digest")
		}
		saltedUniverseDigest, err := decode32ByteDigestFromString(diff.SaltedUniverseDigestHex)
		if err != nil {
			return nil, errors.Wrap(err, "decoding salted universe digest")
		}
		saltedComputerDigest, err := decode32ByteDigestFromString(diff.SaltedComputerDigestHex)
		if err != nil {
			return nil, errors.Wrap(err, "decoding salted computer digest")
		}
		expectedNextTickTxDigest, err := decode32ByteDigestFromString(diff.ExpectedNextTickTxDigestHex)
		if err != nil {
			return nil, errors.Wrap(err, "decoding expected next tick transaction digest")
		}
		signature, err := decode64ByteDigestFromString(diff.SignatureHex)
		if err != nil {
			return nil, errors.Wrap(err, "decoding signature")
		}

		computorVote := types.QuorumTickVote{
			ComputorIndex:                 uint16(index),
			Epoch:                         uint16(tickStructure.Epoch),
			Tick:                          tickStructure.TickNumber,
			Millisecond:                   uint16(milli),
			Second:                        uint8(tickTime.Second()),
			Minute:                        uint8(tickTime.Minute()),
			Hour:                          uint8(tickTime.Hour()),
			Day:                           uint8(tickTime.Day()),
			Month:                         uint8(tickTime.Month()),
			Year:                          uint8(tickTime.Year() - 2000),
			PreviousResourceTestingDigest: prevResourceTestingDigest,
			SaltedResourceTestingDigest:   saltedResourceTestingDigest,
			PreviousSpectrumDigest:        prevSpectrumDigest,
			PreviousUniverseDigest:        prevUniverseDigest,
			PreviousComputerDigest:        prevComputerDigest,
			SaltedSpectrumDigest:          saltedSpectrumDigest,
			SaltedUniverseDigest:          saltedUniverseDigest,
			SaltedComputerDigest:          saltedComputerDigest,
			TxDigest:                      txDigest,
			ExpectedNextTickTxDigest:      expectedNextTickTxDigest,
			Signature:                     signature,
		}

		votes = append(votes, computorVote)

	}
	return votes, nil
}
