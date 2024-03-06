package processor

import (
	"context"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/validator"
	qubic "github.com/qubic/go-node-connector"
	"github.com/qubic/go-node-connector/types"
	"github.com/silenceper/pool"
	"log"
	"time"
)

func newTickInTheFutureError(requestedTick uint64, latestTick uint64) *TickInTheFutureError {
	return &TickInTheFutureError{requestedTick: requestedTick, latestTick: latestTick}
}

type TickInTheFutureError struct {
	requestedTick uint64
	latestTick    uint64
}

func (e *TickInTheFutureError) Error() string {
	return errors.Errorf("Requested tick %d is in the future. Latest tick is: %d", e.requestedTick, e.latestTick).Error()
}

type Processor struct {
	pool               pool.Pool
	ps                 *store.PebbleStore
	processTickTimeout time.Duration
}

func NewProcessor(pool pool.Pool, ps *store.PebbleStore, processTickTimeout time.Duration) *Processor {
	return &Processor{
		pool:               pool,
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
	qcv, err := p.pool.Get()
	if err != nil {
		return errors.Wrap(err, "getting qubic pooled client connection")
	}
	client := qcv.(*qubic.Client)
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

	nextTick, err := p.getNextProcessingTick(ctx, tickInfo)
	if err != nil {
		return errors.Wrap(err, "getting next processing tick")
	}
	log.Printf("Next tick to process: %d\n", nextTick)

	if uint64(tickInfo.Tick) < nextTick {
		err = newTickInTheFutureError(nextTick, uint64(tickInfo.Tick))
		return err
	}

	val := validator.New(client, p.ps)
	err = val.ValidateTick(ctx, nextTick)
	if err != nil {
		return errors.Wrapf(err, "validating tick %d", nextTick)
	}

	err = p.ps.SetLastProcessedTick(ctx, nextTick, uint32(tickInfo.Epoch))
	if err != nil {
		return errors.Wrapf(err, "setting last processed tick %d", nextTick)
	}

	return nil
}

func (p *Processor) getNextProcessingTick(ctx context.Context, currentTickInfo types.TickInfo) (uint64, error) {
	lastTick, err := p.ps.GetLastProcessedTick(ctx)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return uint64(currentTickInfo.InitialTick), nil
		}

		return 0, errors.Wrap(err, "getting last processed tick")
	}

	if uint64(currentTickInfo.InitialTick) > lastTick {
		return uint64(currentTickInfo.InitialTick), nil
	}

	return lastTick + 1, nil
}
