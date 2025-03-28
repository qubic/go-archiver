package tx

import (
	"context"
	"errors"
	"fmt"
	"github.com/qubic/go-archiver/protobuff"
	"go.uber.org/zap"
	"log"
	"sync"
	"time"
)

type LongTermStorageClient interface {
	PushSingleTx(ctx context.Context, tx Tx) error
	PushMultipleTx(ctx context.Context, txs []Tx) error
}

type Processor struct {
	internalStore                 *ProcessorStore
	archiveClient                 protobuff.ArchiveServiceClient
	ltsClient                     LongTermStorageClient
	batchSize                     int
	logger                        *zap.SugaredLogger
	archiveTickTransactionTimeout time.Duration
	ltsBatchInsertTimeout         time.Duration
}

func NewProcessor(store *ProcessorStore, archiveClient protobuff.ArchiveServiceClient, ltsClient LongTermStorageClient, batchSize int, logger *zap.SugaredLogger, archiveTickTransactionTimeout time.Duration, ltsBatchInsertTimeout time.Duration) (*Processor, error) {
	return &Processor{internalStore: store, archiveClient: archiveClient, ltsClient: ltsClient, batchSize: batchSize, logger: logger, archiveTickTransactionTimeout: archiveTickTransactionTimeout, ltsBatchInsertTimeout: ltsBatchInsertTimeout}, nil
}

func (p *Processor) Start(nrWorkers int) error {
	for {
		status, err := p.archiveClient.GetStatus(context.Background(), nil)
		if err != nil {
			return fmt.Errorf("getting status: %v", err)
		}

		startingTicksForEpochs, err := p.getStartingTicksForEpochs(status.ProcessedTickIntervalsPerEpoch)
		if err != nil {
			return fmt.Errorf("getting starting ticks for epochs: %v", err)
		}

		var startedWorkers int
		var wg sync.WaitGroup
		for _, epochIntervals := range status.ProcessedTickIntervalsPerEpoch {
			if startedWorkers == nrWorkers {
				wg.Wait()
				startedWorkers = 0
			}

			startingTick, ok := startingTicksForEpochs[epochIntervals.Epoch]
			if !ok {
				return fmt.Errorf("starting tick not found for epoch %d", epochIntervals.Epoch)
			}

			if startingTick == epochIntervals.Intervals[len(epochIntervals.Intervals)-1].LastProcessedTick+1 {
				continue
			}

			wg.Add(1)
			startedWorkers++
			go func() {
				defer wg.Done()
				err = p.processEpoch(epochIntervals.Epoch, startingTick, epochIntervals)
				if err != nil {
					p.logger.Errorw("error processing epoch", "epoch", epochIntervals.Epoch, "error", err)
				} else {
					p.logger.Infow("Finished processing epoch", "epoch", epochIntervals.Epoch)
				}
			}()
		}
	}
}

func (p *Processor) getStartingTicksForEpochs(epochsIntervals []*protobuff.ProcessedTickIntervalsPerEpoch) (map[uint32]uint32, error) {
	startingTicks := make(map[uint32]uint32)

	for _, epochIntervals := range epochsIntervals {
		lastProcessedTick, err := p.internalStore.GetLastProcessedTick(epochIntervals.Epoch)
		if errors.Is(err, ErrNotFound) {
			startingTicks[epochIntervals.Epoch] = epochIntervals.Intervals[0].InitialProcessedTick
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("getting last processed tick: %v", err)
		}

		startingTicks[epochIntervals.Epoch] = lastProcessedTick + 1
	}

	return startingTicks, nil
}

func (p *Processor) processEpoch(epoch uint32, startTick uint32, epochTickIntervals *protobuff.ProcessedTickIntervalsPerEpoch) error {
	p.logger.Infow("Starting epoch processor", "epoch", epoch, "startTick", startTick)

	// length needs to be equal to configured batchSize + 1024 which is the max number of transaction per tick
	// this is to ensure that we are not inserting only a subset of transactions for a certain tick
	batchTxToInsert := make([]Tx, 0, p.batchSize+1024)

	for _, interval := range epochTickIntervals.Intervals {
		for tick := interval.InitialProcessedTick; tick <= interval.LastProcessedTick; tick++ {
			if tick < startTick {
				continue
			}

			txs, err := p.getTransactionsFromArchiver(tick)
			if errors.Is(err, ErrEmptyTick) {
				continue
			}

			if err != nil {
				p.logger.Errorw("error processing tick; retrying...", "epoch", epoch, "tick", tick, "error", err)
				tick--
				continue
			}

			batchTxToInsert = append(batchTxToInsert, txs...)
			if len(batchTxToInsert) >= p.batchSize || tick == interval.LastProcessedTick {
				err = p.PushMultipleTxWithRetry(batchTxToInsert)
				if err != nil {
					return fmt.Errorf("inserting batch: %v", err)
				}

				p.logger.Infow("Storing last processed tick", "epoch", epoch, "tick", tick)
				err = p.internalStore.SetLastProcessedTick(epoch, tick)
				if err != nil {
					return fmt.Errorf("setting last processed tick: %v", err)
				}

				batchTxToInsert = make([]Tx, 0, p.batchSize+1024)
			}
		}
	}

	return nil
}

var ErrEmptyTick = errors.New("empty tick")

func (p *Processor) getTransactionsFromArchiver(tick uint32) ([]Tx, error) {
	ctx, cancel := context.WithTimeout(context.Background(), p.archiveTickTransactionTimeout)
	defer cancel()

	txs, err := p.archiveClient.GetTickTransactionsV2(ctx, &protobuff.GetTickTransactionsRequestV2{TickNumber: tick})
	if err != nil {
		if err.Error() == "rpc error: code = NotFound desc = tick transactions for specified tick not found" {
			return []Tx{}, ErrEmptyTick
		}

		return nil, fmt.Errorf("getting tick transactions: %v", err)
	}

	if len(txs.Transactions) == 0 {
		return []Tx{}, ErrEmptyTick
	}

	tickTransactions := make([]Tx, 0, len(txs.Transactions))

	for _, archiveTx := range txs.Transactions {
		ltsTx, err := ArchiveTxToLtsTx(archiveTx)
		if err != nil {
			return nil, fmt.Errorf("converting archive tx to lts tx: %v", err)
		}

		tickTransactions = append(tickTransactions, ltsTx)
	}

	return tickTransactions, nil
}

func (p *Processor) PushMultipleTxWithRetry(txs []Tx) error {
	ctx, cancel := context.WithTimeout(context.Background(), p.ltsBatchInsertTimeout)
	defer cancel()

	var lastErr error
	p.logger.Infow("Inserting batch of transactions", "inserted_batch_size", len(txs))
	for i := 0; i < 20; i++ {
		err := p.ltsClient.PushMultipleTx(ctx, txs)
		if err != nil {
			log.Printf("Error inserting batch. Retrying... %s", err.Error())
			lastErr = err
			continue
		} else {
			lastErr = nil
			break
		}

	}

	if lastErr != nil {
		return fmt.Errorf("inserting tx batch with retry: %v", lastErr)
	}

	return nil
}
