package tick

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-node-connector/types"
)

func CalculateEmptyTicksForEpoch(ctx context.Context, ps *store.PebbleStore, epoch uint32) (uint32, error) {

	epochs, err := ps.GetProcessedTickIntervals(ctx)
	if err != nil {
		return 0, errors.Wrap(err, "getting stored epochs")
	}

	for _, e := range epochs {
		if e.Epoch != epoch {
			continue
		}

		var emptyTicks uint32

		for _, interval := range e.Intervals {
			fmt.Printf("Interval: %d -> %d\n", interval.InitialProcessedTick, interval.LastProcessedTick)
			for tickOffset := range interval.LastProcessedTick - interval.InitialProcessedTick + 1 {
				tickNumber := tickOffset + interval.InitialProcessedTick

				tickData, err := ps.GetTickData(ctx, tickNumber)
				if err != nil {
					return 0, errors.Wrapf(err, "getting tick data for tick %d", tickNumber)
				}

				if CheckIfTickIsEmptyProto(tickData) {
					fmt.Printf("Found empty tick: %d\n", tickNumber)
					emptyTicks += 1
					continue
				}
			}
		}
		return emptyTicks, err
	}
	return 0, nil
}

func CheckIfTickIsEmptyProto(tickData *protobuff.TickData) bool {
	if tickData == nil || tickData.VarStruct == nil {
		return true
	}

	return false
}

func CheckIfTickIsEmpty(tickData types.TickData) (bool, error) {
	data, err := qubicToProto(tickData)
	if err != nil {
		return false, errors.Wrap(err, "converting tick data to protobuf format")
	}

	return CheckIfTickIsEmptyProto(data), nil
}
