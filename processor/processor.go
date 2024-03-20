package processor

import (
	"context"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/validator"
	qubic "github.com/qubic/go-node-connector"
	"github.com/qubic/go-node-connector/types"
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
	return errors.Errorf("Requested tick %d is in the future. Latest tick is: %d", e.requestedTick, e.latestTick).Error()
}

type Processor struct {
	pool               *qubic.Pool
	ps                 *store.PebbleStore
	processTickTimeout time.Duration
}

func NewProcessor(p *qubic.Pool, ps *store.PebbleStore, processTickTimeout time.Duration) *Processor {
	return &Processor{
		pool:               p,
		ps:                 ps,
		processTickTimeout: processTickTimeout,
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
		return errors.Wrap(err, "getting qubic pooled client connection")
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
		return errors.Wrap(err, "getting tick info")
	}

	lastTick, err := p.getLastProcessedTick(ctx, tickInfo)
	if err != nil {
		return errors.Wrap(err, "getting last processed tick")
	}

	nextTick, err := p.getNextProcessingTick(ctx, lastTick, tickInfo)
	if err != nil {
		return errors.Wrap(err, "getting next processing tick")
	}
	log.Printf("Next tick to process: %d\n", nextTick)

	if tickInfo.Tick < nextTick {
		err = newTickInTheFutureError(nextTick, tickInfo.Tick)
		return err
	}

	val := validator.New(client, p.ps)
	err = val.ValidateTick(ctx, tickInfo.InitialTick, nextTick)
	if err != nil {
		return errors.Wrapf(err, "validating tick %d", nextTick)
	}

	err = p.processSkippedTicks(ctx, lastTick, nextTick)
	if err != nil {
		return errors.Wrap(err, "processing skipped ticks")
	}

	err = p.ps.SetLastProcessedTick(ctx, &protobuff.LastProcessedTick{TickNumber: nextTick, Epoch: uint32(tickInfo.Epoch)})
	if err != nil {
		return errors.Wrapf(err, "setting last processed tick %d", nextTick)
	}

	return nil
}

func (p *Processor) getNextProcessingTick(ctx context.Context, lastTick uint32, currentTickInfo types.TickInfo) (uint32, error) {
	//handles the case where the initial tick of epoch returned by the node is greater than the last processed tick
	// which means that we are in the next epoch and we should start from the initial tick of the current epoch
	if currentTickInfo.InitialTick > lastTick {
		return currentTickInfo.InitialTick, nil
	}

	// otherwise we are in the same epoch and we should start from the last processed tick + 1
	return lastTick + 1, nil
}

func (p *Processor) getLastProcessedTick(ctx context.Context, currentTickInfo types.TickInfo) (uint32, error) {
	lastTick, err := p.ps.GetLastProcessedTick(ctx)
	if err != nil {
		//handles first run of the archiver where there is nothing in storage
		// in this case we last tick is the initial tick of the current epoch - 1
		if errors.Is(err, store.ErrNotFound) {
			return currentTickInfo.InitialTick - 1, nil
		}

		return 0, errors.Wrap(err, "getting last processed tick")
	}

	return lastTick.TickNumber, nil
}

func (p *Processor) processSkippedTicks(ctx context.Context, lastTick uint32, nextTick uint32) error {
	// nothing to process, no skipped ticks
	if nextTick-lastTick == 1 {
		return nil
	}

	err := p.ps.SetSkippedTicksInterval(ctx, &protobuff.SkippedTicksInterval{
		StartTick: lastTick + 1,
		EndTick:   nextTick - 1,
	})
	if err != nil {
		return errors.Wrap(err, "setting skipped ticks interval")
	}

	return nil
}
