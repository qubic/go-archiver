package tick

import (
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
	"time"
)

func qubicToProto(tickData types.TickData) (*protobuff.TickData, error) {
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

func WasTickSkippedByArchive(tick uint32, processedTicksIntervalPerEpoch []*protobuff.ProcessedTickIntervalsPerEpoch) (bool, uint32) {
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

	return 0, errors.New(fmt.Sprintf("unable to find the epoch for tick %d", tickNumber))
}

func GetProcessedTickIntervalsForEpoch(epoch uint32, intervals []*protobuff.ProcessedTickIntervalsPerEpoch) (*protobuff.ProcessedTickIntervalsPerEpoch, error) {
	for _, interval := range intervals {
		if interval.Epoch != epoch {
			continue
		}
		return interval, nil
	}

	return nil, errors.New(fmt.Sprintf("unable to find processed tick intervals for epoch %d", epoch))
}

func IsLastTick(tickNumber uint32, epoch uint32, intervals []*protobuff.ProcessedTickIntervalsPerEpoch) (bool, error) {
	epochIntervals, err := GetProcessedTickIntervalsForEpoch(epoch, intervals)
	if err != nil {
		return false, errors.Wrap(err, "getting processed tick intervals per epoch")
	}

	for _, interval := range epochIntervals.Intervals {
		if interval.LastProcessedTick == tickNumber {
			return true, nil
		}
	}

	return false, nil
}
