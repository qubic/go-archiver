package processor

import (
	"context"
	"errors"
	"fmt"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/validator"
	qubic "github.com/qubic/go-node-connector/v2"
	"github.com/qubic/go-node-connector/v2/types"
	"log"
	"time"
)

func newTickInTheFutureError(requestedTick uint32, latestTick uint32) *TickInTheFutureError {
	return &TickInTheFutureError{requestedTick: requestedTick, latestTick: latestTick}
}

type TickInTheFutureError struct {
	requestedTick uint32
	latestTick    uint32
}

func (e *TickInTheFutureError) Error() string {
	return fmt.Errorf("Requested tick %d is in the future. Latest tick is: %d", e.requestedTick, e.latestTick).Error()
}

type Processor struct {
	pool               *qubic.Pool
	ps                 *store.PebbleStore
	arbitratorPubKey   [32]byte
	processTickTimeout time.Duration
	disableStatusAddon bool
}

func NewProcessor(p *qubic.Pool, ps *store.PebbleStore, processTickTimeout time.Duration, arbitratorPubKey [32]byte, disableStatusAddon bool) *Processor {
	return &Processor{
		pool:               p,
		ps:                 ps,
		processTickTimeout: processTickTimeout,
		arbitratorPubKey:   arbitratorPubKey,
		disableStatusAddon: disableStatusAddon,
	}
}

func (p *Processor) Start() error {
	for {
		err := p.processOneByOne()
		if err != nil {
			log.Printf("Processing failed: %s", err.Error())
			time.Sleep(1 * time.Second)
		}
	}
}

func (p *Processor) processOneByOne() error {
	ctx, cancel := context.WithTimeout(context.Background(), p.processTickTimeout)
	defer cancel()

	var err error
	client, err := p.pool.Get()
	if err != nil {
		return fmt.Errorf("getting qubic pooled client connection: %w", err)
	}
	defer func() {
		if err == nil {
			log.Printf("Putting conn back to pool")
			pErr := p.pool.Put(client)
			if pErr != nil {
				log.Printf("Putting conn back to pool failed: %s", pErr.Error())
			}
		} else {
			log.Printf("Closing conn")
			cErr := p.pool.Close(client)
			if cErr != nil {
				log.Printf("Closing conn failed: %s", cErr.Error())
			}
		}
	}()

	tickInfo, err := client.GetTickInfo(ctx)
	if err != nil {
		return fmt.Errorf("getting tick info: %w", err)
	}

	lastTick, err := p.getLastProcessedTick(ctx, tickInfo)
	if err != nil {
		return fmt.Errorf("getting last processed tick: %w", err)
	}

	nextTick, err := p.getNextProcessingTick(ctx, lastTick, tickInfo)
	if err != nil {
		return fmt.Errorf("getting next processing tick: %w", err)
	}
	log.Printf("Next tick to process: %d\n", nextTick.TickNumber)

	if tickInfo.Tick < nextTick.TickNumber {
		err = newTickInTheFutureError(nextTick.TickNumber, tickInfo.Tick)
		return err
	}

	val := validator.New(client, p.ps, p.arbitratorPubKey)
	err = val.ValidateTick(ctx, tickInfo.InitialTick, nextTick.TickNumber, p.disableStatusAddon)
	if err != nil {
		return fmt.Errorf("validating tick %d: %w", nextTick.TickNumber, err)
	}

	err = p.processStatus(ctx, lastTick, nextTick)
	if err != nil {
		return fmt.Errorf("processing status for lastTick %+v and nextTick %+v: %w", lastTick, nextTick, err)
	}

	return nil
}

func (p *Processor) processStatus(ctx context.Context, lastTick *protobuff.ProcessedTick, nextTick *protobuff.ProcessedTick) error {
	err := p.processSkippedTicks(ctx, lastTick, nextTick)
	if err != nil {
		return fmt.Errorf("processing skipped ticks: %w", err)
	}

	err = p.ps.SetLastProcessedTick(ctx, nextTick)
	if err != nil {
		return fmt.Errorf("setting last processed tick %d: %w", nextTick.TickNumber, err)
	}

	return nil
}

func (p *Processor) getNextProcessingTick(_ context.Context, lastTick *protobuff.ProcessedTick, currentTickInfo types.TickInfo) (*protobuff.ProcessedTick, error) {
	//handles the case where the initial tick of epoch returned by the node is greater than the last processed tick
	// which means that we are in the next epoch and we should start from the initial tick of the current epoch
	if currentTickInfo.InitialTick > lastTick.TickNumber {
		return &protobuff.ProcessedTick{TickNumber: currentTickInfo.InitialTick, Epoch: uint32(currentTickInfo.Epoch)}, nil
	}

	// otherwise we are in the same epoch and we should start from the last processed tick + 1
	return &protobuff.ProcessedTick{TickNumber: lastTick.TickNumber + 1, Epoch: lastTick.Epoch}, nil
}

func (p *Processor) getLastProcessedTick(ctx context.Context, currentTickInfo types.TickInfo) (*protobuff.ProcessedTick, error) {
	lastTick, err := p.ps.GetLastProcessedTick(ctx)
	if err != nil {
		//handles first run of the archiver where there is nothing in storage
		// in this case last tick is 0 and epoch is current tick info epoch
		if errors.Is(err, store.ErrNotFound) {
			return &protobuff.ProcessedTick{TickNumber: 0, Epoch: uint32(currentTickInfo.Epoch)}, nil
		}

		return nil, fmt.Errorf("getting last processed tick: %w", err)
	}

	return lastTick, nil
}

func (p *Processor) processSkippedTicks(ctx context.Context, lastTick *protobuff.ProcessedTick, nextTick *protobuff.ProcessedTick) error {
	// nothing to process, no skipped ticks
	if nextTick.TickNumber-lastTick.TickNumber == 1 {
		return nil
	}

	if nextTick.TickNumber-lastTick.TickNumber == 0 {
		return fmt.Errorf("Next tick should not be equal to last tick %d", nextTick.TickNumber)
	}

	err := p.ps.AppendProcessedTickInterval(ctx, nextTick.Epoch, &protobuff.ProcessedTickInterval{InitialProcessedTick: nextTick.TickNumber, LastProcessedTick: nextTick.TickNumber})
	if err != nil {
		return fmt.Errorf("appending processed tick interval: %w", err)
	}

	err = p.ps.SetSkippedTicksInterval(ctx, &protobuff.SkippedTicksInterval{
		StartTick: lastTick.TickNumber + 1,
		EndTick:   nextTick.TickNumber - 1,
	})
	if err != nil {
		return fmt.Errorf("setting skipped ticks interval: %w", err)
	}

	return nil
}
