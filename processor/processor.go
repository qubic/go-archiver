package processor

import (
	"context"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/validator"
	qubic "github.com/qubic/go-node-connector"
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
	pool                       pool.Pool
	ps                         *store.PebbleStore
	processTickTimeout         time.Duration
	fallbackNextProcessingTick uint64
}

func NewProcessor(pool pool.Pool, ps *store.PebbleStore, fallbackNextProcessingTick uint64, processTickTimeout time.Duration) *Processor {
	return &Processor{
		pool:                       pool,
		ps:                         ps,
		fallbackNextProcessingTick: fallbackNextProcessingTick,
		processTickTimeout:         processTickTimeout,
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

	nextTick, err := p.getNextProcessingTick(ctx)
	if err != nil {
		return errors.Wrap(err, "getting next processing tick")
	}
	log.Printf("Next tick to process: %d\n", nextTick)
	tickInfo, err := client.GetTickInfo(ctx)
	if err != nil {
		return errors.Wrap(err, "getting tick info")
	}
	if uint64(tickInfo.Tick) < nextTick {
		err = newTickInTheFutureError(nextTick, uint64(tickInfo.Tick))
		return err
	}

	val := validator.New(client, p.ps)
	err = val.ValidateTick(ctx, nextTick)
	if err != nil {
		return errors.Wrapf(err, "validating tick %d", nextTick)
	}

	err = p.ps.SetLastProcessedTick(ctx, nextTick)
	if err != nil {
		return errors.Wrapf(err, "setting last processed tick %d", nextTick)
	}

	return nil
}

func (p *Processor) getNextProcessingTick(ctx context.Context) (uint64, error) {
	lastTick, err := p.ps.GetLastProcessedTick(ctx)
	if err == nil {
		return lastTick + 1, nil
	}

	if errors.Cause(err) == store.ErrNotFound {
		return p.fallbackNextProcessingTick, nil
	}

	return 0, errors.Wrap(err, "getting last processed tick")
}
