package tick

import (
	"context"
	"fmt"
	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-node-connector/types"
	"google.golang.org/protobuf/proto"
	"time"
)

var emptyTickData = &protobuff.TickData{}

func CalculateEmptyTicksForEpoch(ctx context.Context, ps *store.PebbleStore, epoch uint32) ([]uint32, error) {

	epochs, err := ps.GetProcessedTickIntervals(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "getting stored epochs")
	}

	for _, e := range epochs {
		if e.Epoch != epoch {
			continue
		}

		var emptyTicks []uint32

		for _, interval := range e.Intervals {
			fmt.Printf("Interval: %d -> %d\n", interval.InitialProcessedTick, interval.LastProcessedTick)
			for tickOffset := range interval.LastProcessedTick - interval.InitialProcessedTick + 1 {
				tickNumber := tickOffset + interval.InitialProcessedTick

				tickData, err := ps.GetTickData(ctx, tickNumber)
				if err != nil {
					return nil, errors.Wrapf(err, "getting tick data for tick %d", tickNumber)
				}

				if CheckIfTickIsEmptyProto(tickData) {
					fmt.Printf("Found empty tick: %d\n", tickNumber)
					emptyTicks = append(emptyTicks, tickNumber)
					continue
				}
			}
		}
		return emptyTicks, err
	}
	return make([]uint32, 0), nil
}

func CheckIfTickIsEmptyProto(tickData *protobuff.TickData) bool {

	if tickData == nil || proto.Equal(tickData, emptyTickData) {
		return true
	}

	return false
}

func CheckIfTickIsEmpty(tickData types.TickData) (bool, error) {
	data, err := QubicToProto(tickData)
	if err != nil {
		return false, errors.Wrap(err, "converting tick data to protobuf format")
	}

	return CheckIfTickIsEmptyProto(data), nil
}

func CalculateEmptyTicksForAllEpochs(ps *store.PebbleStore) error {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	epochs, err := ps.GetLastProcessedTicksPerEpoch(ctx)
	if err != nil {
		return errors.Wrap(err, "getting epoch list from db")
	}

	for epoch, _ := range epochs {

		_, err := ps.GetEmptyTicksForEpoch(epoch)
		if err == nil {
			return nil // We have the empty ticks
		}
		if !errors.Is(err, pebble.ErrNotFound) {
			return errors.Wrap(err, "checking if epoch has empty ticks") // Some other error occured
		}

		fmt.Printf("Calculating empty ticks for epoch %d\n", epoch)
		emptyTicksPerEpoch, err := CalculateEmptyTicksForEpoch(ctx, ps, epoch)
		if err != nil {
			return errors.Wrapf(err, "calculating empty ticks for epoch %d", epoch)
		}

		err = ps.SetEmptyTicksForEpoch(epoch, uint32(len(emptyTicksPerEpoch)))
		if err != nil {
			return errors.Wrap(err, "saving emptyTickCount to database")
		}
		err = ps.SetEmptyTickListPerEpoch(epoch, emptyTicksPerEpoch)
		if err != nil {
			return errors.Wrap(err, "saving empty tick list to database")
		}

	}
	return nil
}

func ResetEmptyTicksForAllEpochs(ps *store.PebbleStore) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	epochs, err := ps.GetLastProcessedTicksPerEpoch(ctx)
	if err != nil {
		return errors.Wrap(err, "getting epoch list from db")
	}

	for epoch, _ := range epochs {
		fmt.Printf("Reseting empty ticks for epoch: %d\n", epoch)
		err := ps.DeleteEmptyTicksKeyForEpoch(epoch)
		if err != nil {
			return errors.Wrap(err, "deleting empty tick key")
		}
		err = ps.DeleteEmptyTickListKeyForEpoch(epoch)
		if err != nil {
			return errors.Wrap(err, "deleting empty tick list key")
		}
	}

	return nil
}
