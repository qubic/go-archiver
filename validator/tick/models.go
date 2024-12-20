package tick

import (
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
	"time"
)

func QubicToProto(tickData types.TickData) (*protobuff.TickData, error) {
	if tickData.IsEmpty() {
		return nil, nil
	}

	date := time.Date(2000+int(tickData.Year), time.Month(tickData.Month), int(tickData.Day), int(tickData.Hour), int(tickData.Minute), int(tickData.Second), 0, time.UTC)
	timestamp := date.UnixMilli() + int64(tickData.Millisecond)
	transactionIds, err := digestsToIdentities(tickData.TransactionDigests)
	if err != nil {
		return nil, errors.Wrap(err, "getting transaction ids from digests")
	}
	return &protobuff.TickData{
		ComputorIndex:  uint32(tickData.ComputorIndex),
		Epoch:          uint32(tickData.Epoch),
		TickNumber:     tickData.Tick,
		Timestamp:      uint64(timestamp),
		TimeLock:       tickData.Timelock[:],
		TransactionIds: transactionIds,
		ContractFees:   contractFeesToProto(tickData.ContractFees),
		SignatureHex:   hex.EncodeToString(tickData.Signature[:]),
	}, nil
}

func digestsToIdentities(digests [types.NumberOfTransactionsPerTick][32]byte) ([]string, error) {
	identities := make([]string, 0)
	for _, digest := range digests {
		if digest == [32]byte{} {
			continue
		}
		var id types.Identity
		id, err := id.FromPubKey(digest, true)
		if err != nil {
			return nil, errors.Wrapf(err, "getting identity from digest hex %s", hex.EncodeToString(digest[:]))
		}
		identities = append(identities, id.String())
	}

	return identities, nil
}

func contractFeesToProto(contractFees [1024]int64) []int64 {
	protoContractFees := make([]int64, 0, len(contractFees))
	for _, fee := range contractFees {
		if fee == 0 {
			continue
		}
		protoContractFees = append(protoContractFees, fee)
	}
	return protoContractFees
}

func WasSkippedByArchive(tick uint32, processedTicksIntervalPerEpoch []*protobuff.ProcessedTickIntervalsPerEpoch) (bool, uint32) {
	if len(processedTicksIntervalPerEpoch) == 0 {
		return false, 0
	}
	for _, epochInterval := range processedTicksIntervalPerEpoch {
		for _, interval := range epochInterval.Intervals {
			if tick < interval.InitialProcessedTick {
				return true, interval.InitialProcessedTick
			}
			if tick >= interval.InitialProcessedTick && tick <= interval.LastProcessedTick {
				return false, 0
			}
		}
	}
	return false, 0
}

func GetTickEpoch(tickNumber uint32, intervals []*protobuff.ProcessedTickIntervalsPerEpoch) (uint32, error) {
	if len(intervals) == 0 {
		return 0, errors.New("processed tick interval list is empty")
	}

	for _, epochInterval := range intervals {
		for _, interval := range epochInterval.Intervals {
			if tickNumber >= interval.InitialProcessedTick && tickNumber <= interval.LastProcessedTick {
				return epochInterval.Epoch, nil
			}
		}
	}

	return 0, errors.Errorf("unable to find the epoch for tick %d", tickNumber)
}

func GetProcessedTickIntervalsForEpoch(epoch uint32, intervals []*protobuff.ProcessedTickIntervalsPerEpoch) (*protobuff.ProcessedTickIntervalsPerEpoch, error) {
	for _, interval := range intervals {
		if interval.Epoch != epoch {
			continue
		}
		return interval, nil
	}

	return nil, errors.Errorf("unable to find processed tick intervals for epoch %d", epoch)
}

func IsTickLastInAnyEpochInterval(tickNumber uint32, epoch uint32, intervals []*protobuff.ProcessedTickIntervalsPerEpoch) (bool, int, error) {
	epochIntervals, err := GetProcessedTickIntervalsForEpoch(epoch, intervals)
	if err != nil {
		return false, -1, errors.Wrap(err, "getting processed tick intervals per epoch")
	}

	for index, interval := range epochIntervals.Intervals {
		if interval.LastProcessedTick == tickNumber {
			return true, index, nil
		}
	}

	return false, -1, nil
}

func identitiesToDigests(identities []string) ([types.NumberOfTransactionsPerTick][32]byte, error) {

	var digests [types.NumberOfTransactionsPerTick][32]byte

	for index, identity := range identities {

		id := types.Identity(identity)
		digest, err := id.ToPubKey(true)
		if err != nil {
			return [1024][32]byte{}, errors.Wrapf(err, "obtaining digest from transaction id %s", identity)
		}
		digests[index] = digest
	}

	return digests, nil
}

func contractFeesFromProto(feesProto []int64) ([1024]int64, error) {

	if len(feesProto) > 1024 {
		return [1024]int64{}, errors.New(fmt.Sprintf("fees array length larger than maximum allowed: %d > 1024 ", len(feesProto)))
	}

	var contractFees [1024]int64
	for index, fee := range feesProto {
		contractFees[index] = fee
	}
	return contractFees, nil

}

func ProtoToQubic(tickData *protobuff.TickData) (types.TickData, error) {

	if CheckIfTickIsEmptyProto(tickData) {
		return types.TickData{}, nil
	}

	tickTime := time.UnixMilli(int64(tickData.Timestamp)).UTC()
	tickTimeNoMilli := time.Date(tickTime.Year(), tickTime.Month(), tickTime.Day(), tickTime.Hour(), tickTime.Minute(), tickTime.Second(), 0, time.UTC)
	milli := tickTime.UnixMilli() - tickTimeNoMilli.UnixMilli()

	var timeLock [32]byte
	copy(timeLock[:], tickData.TimeLock[:])

	transactionDigests, err := identitiesToDigests(tickData.TransactionIds)
	if err != nil {
		return types.TickData{}, errors.Wrap(err, "decoding transaction ids to digests")
	}

	contractFees, err := contractFeesFromProto(tickData.ContractFees)
	if err != nil {
		return types.TickData{}, errors.Wrap(err, "converting contract fees")
	}

	decodedSignature, err := hex.DecodeString(tickData.SignatureHex)
	if err != nil {
		return types.TickData{}, errors.Wrap(err, "decoding signature")
	}

	var signature [types.SignatureSize]byte
	copy(signature[:], decodedSignature[:])

	data := types.TickData{
		ComputorIndex:      uint16(tickData.ComputorIndex),
		Epoch:              uint16(tickData.Epoch),
		Tick:               tickData.TickNumber,
		Millisecond:        uint16(milli),
		Second:             uint8(tickTime.Second()),
		Minute:             uint8(tickTime.Minute()),
		Hour:               uint8(tickTime.Hour()),
		Day:                uint8(tickTime.Day()),
		Month:              uint8(tickTime.Month()),
		Year:               uint8(tickTime.Year() - 2000),
		Timelock:           timeLock,
		TransactionDigests: transactionDigests,
		ContractFees:       contractFees,
		Signature:          signature,
	}

	return data, nil
}

func ProtoToQubicFull(tickData *protobuff.TickData) (FullTickData, error) {

	qubicTickData, err := ProtoToQubic(tickData)
	if err != nil {
		return FullTickData{}, errors.Wrap(err, "converting tick data to qubic format")
	}

	varStruct := tickData.VarStruct
	var unionData [256]byte
	copy(unionData[:], varStruct[:])

	return FullTickData{
		ComputorIndex:      qubicTickData.ComputorIndex,
		Epoch:              qubicTickData.Epoch,
		Tick:               qubicTickData.Tick,
		Millisecond:        qubicTickData.Millisecond,
		Second:             qubicTickData.Second,
		Minute:             qubicTickData.Minute,
		Hour:               qubicTickData.Hour,
		Day:                qubicTickData.Day,
		Month:              qubicTickData.Month,
		Year:               qubicTickData.Year,
		Timelock:           qubicTickData.Timelock,
		UnionData:          unionData,
		TransactionDigests: qubicTickData.TransactionDigests,
		ContractFees:       qubicTickData.ContractFees,
		Signature:          qubicTickData.Signature,
	}, nil
}

func QubicFullToProto(tickData FullTickData) (*protobuff.TickData, error) {

	oldTickData := types.TickData{
		ComputorIndex:      tickData.ComputorIndex,
		Epoch:              tickData.Epoch,
		Tick:               tickData.Tick,
		Millisecond:        tickData.Millisecond,
		Second:             tickData.Second,
		Minute:             tickData.Minute,
		Hour:               tickData.Hour,
		Day:                tickData.Day,
		Month:              tickData.Month,
		Year:               tickData.Year,
		Timelock:           tickData.Timelock,
		TransactionDigests: tickData.TransactionDigests,
		ContractFees:       tickData.ContractFees,
		Signature:          tickData.Signature,
	}

	proto, err := QubicToProto(oldTickData)
	if err != nil {
		return nil, errors.Wrapf(err, "qubic to proto")
	}

	proto.VarStruct = tickData.UnionData[:]

	return proto, nil

}
