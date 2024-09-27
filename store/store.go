package store

import (
	"context"
	"encoding/binary"
	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"strconv"
)

const maxTickNumber = ^uint64(0)

var ErrNotFound = errors.New("store resource not found")

type PebbleStore struct {
	db     *pebble.DB
	logger *zap.Logger
}

func NewPebbleStore(db *pebble.DB, logger *zap.Logger) *PebbleStore {
	return &PebbleStore{db: db, logger: logger}
}

func (s *PebbleStore) GetTickData(ctx context.Context, tickNumber uint32) (*protobuff.TickData, error) {
	key := tickDataKey(tickNumber)
	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting tick data")
	}
	defer closer.Close()

	var td protobuff.TickData
	if err := proto.Unmarshal(value, &td); err != nil {
		return nil, errors.Wrap(err, "unmarshalling tick data to protobuff type")
	}

	return &td, err
}

func (s *PebbleStore) SetTickData(ctx context.Context, tickNumber uint32, td *protobuff.TickData) error {
	key := tickDataKey(tickNumber)
	serialized, err := proto.Marshal(td)
	if err != nil {
		return errors.Wrap(err, "serializing td proto")
	}

	err = s.db.Set(key, serialized, pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "setting tick data")
	}

	return nil
}

func (s *PebbleStore) SetQuorumTickData(ctx context.Context, tickNumber uint32, qtd *protobuff.QuorumTickDataStored) error {
	key := quorumTickDataKey(tickNumber)
	serialized, err := proto.Marshal(qtd)
	if err != nil {
		return errors.Wrap(err, "serializing qtdV2 proto")
	}

	err = s.db.Set(key, serialized, pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "setting quorum tick data")
	}

	return nil
}

func (s *PebbleStore) GetQuorumTickData(ctx context.Context, tickNumber uint32) (*protobuff.QuorumTickDataStored, error) {
	key := quorumTickDataKey(tickNumber)
	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting quorum tick data")
	}
	defer closer.Close()

	var qtd protobuff.QuorumTickDataStored
	if err := proto.Unmarshal(value, &qtd); err != nil {
		return nil, errors.Wrap(err, "unmarshalling qtdV2 to protobuf type")
	}

	return &qtd, err
}

func (s *PebbleStore) GetComputors(ctx context.Context, epoch uint32) (*protobuff.Computors, error) {
	key := computorsKey(epoch)

	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting quorum tick data")
	}
	defer closer.Close()

	var computors protobuff.Computors
	if err := proto.Unmarshal(value, &computors); err != nil {
		return nil, errors.Wrap(err, "unmarshalling computors to protobuff type")
	}

	return &computors, nil
}

func (s *PebbleStore) SetComputors(ctx context.Context, epoch uint32, computors *protobuff.Computors) error {
	key := computorsKey(epoch)

	serialized, err := proto.Marshal(computors)
	if err != nil {
		return errors.Wrap(err, "serializing computors proto")
	}

	err = s.db.Set(key, serialized, pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "setting computors")
	}

	return nil
}

func (s *PebbleStore) SetTransactions(ctx context.Context, txs []*protobuff.Transaction) error {
	batch := s.db.NewBatchWithSize(len(txs))
	defer batch.Close()

	for _, tx := range txs {
		key, err := tickTxKey(tx.TxId)
		if err != nil {
			return errors.Wrapf(err, "creating tx key for id: %s", tx.TxId)
		}

		serialized, err := proto.Marshal(tx)
		if err != nil {
			return errors.Wrap(err, "serializing tx proto")
		}

		err = batch.Set(key, serialized, nil)
		if err != nil {
			return errors.Wrap(err, "getting tick data")
		}
	}

	if err := batch.Commit(pebble.Sync); err != nil {
		return errors.Wrap(err, "committing batch")
	}

	return nil
}

func (s *PebbleStore) GetTickTransactions(ctx context.Context, tickNumber uint32) ([]*protobuff.Transaction, error) {
	td, err := s.GetTickData(ctx, tickNumber)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting tick data")
	}

	txs := make([]*protobuff.Transaction, 0, len(td.TransactionIds))
	for _, txID := range td.TransactionIds {
		tx, err := s.GetTransaction(ctx, txID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrNotFound
			}

			return nil, errors.Wrapf(err, "getting tx for id: %s", txID)
		}

		txs = append(txs, tx)
	}

	return txs, nil
}

func (s *PebbleStore) GetTickTransferTransactions(ctx context.Context, tickNumber uint32) ([]*protobuff.Transaction, error) {
	td, err := s.GetTickData(ctx, tickNumber)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting tick data")
	}

	txs := make([]*protobuff.Transaction, 0, len(td.TransactionIds))
	for _, txID := range td.TransactionIds {
		tx, err := s.GetTransaction(ctx, txID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrNotFound
			}

			return nil, errors.Wrapf(err, "getting tx for id: %s", txID)
		}
		if tx.Amount <= 0 {
			continue
		}

		txs = append(txs, tx)
	}

	return txs, nil
}

func (s *PebbleStore) GetTransaction(ctx context.Context, txID string) (*protobuff.Transaction, error) {
	key, err := tickTxKey(txID)
	if err != nil {
		return nil, errors.Wrap(err, "getting tx key")
	}

	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting tx")
	}
	defer closer.Close()

	var tx protobuff.Transaction
	if err := proto.Unmarshal(value, &tx); err != nil {
		return nil, errors.Wrap(err, "unmarshalling tx to protobuff type")
	}

	return &tx, nil
}

func (s *PebbleStore) SetLastProcessedTick(ctx context.Context, lastProcessedTick *protobuff.ProcessedTick) error {
	batch := s.db.NewBatch()
	defer batch.Close()

	key := lastProcessedTickKeyPerEpoch(lastProcessedTick.Epoch)
	value := make([]byte, 4)
	binary.LittleEndian.PutUint32(value, lastProcessedTick.TickNumber)

	err := batch.Set(key, value, pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "setting last processed tick")
	}

	key = lastProcessedTickKey()
	serialized, err := proto.Marshal(lastProcessedTick)
	if err != nil {
		return errors.Wrap(err, "serializing skipped tick proto")
	}

	err = batch.Set(key, serialized, pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "setting last processed tick")
	}

	err = batch.Commit(pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "committing batch")
	}

	ptie, err := s.getProcessedTickIntervalsPerEpoch(ctx, lastProcessedTick.Epoch)
	if err != nil {
		return errors.Wrap(err, "getting ptie")
	}

	if len(ptie.Intervals) == 0 {
		ptie = &protobuff.ProcessedTickIntervalsPerEpoch{Epoch: lastProcessedTick.Epoch, Intervals: []*protobuff.ProcessedTickInterval{{InitialProcessedTick: lastProcessedTick.TickNumber, LastProcessedTick: lastProcessedTick.TickNumber}}}
	} else {
		ptie.Intervals[len(ptie.Intervals)-1].LastProcessedTick = lastProcessedTick.TickNumber
	}

	err = s.SetProcessedTickIntervalPerEpoch(ctx, lastProcessedTick.Epoch, ptie)
	if err != nil {
		return errors.Wrap(err, "setting ptie")
	}

	return nil
}

func (s *PebbleStore) GetLastProcessedTick(ctx context.Context) (*protobuff.ProcessedTick, error) {
	key := lastProcessedTickKey()
	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting last processed tick")
	}
	defer closer.Close()

	// handle old data format, to be removed in the future
	if len(value) == 8 {
		tickNumber := uint32(binary.LittleEndian.Uint64(value))
		ticksPerEpoch, err := s.GetLastProcessedTicksPerEpoch(ctx)
		if err != nil {
			return nil, errors.Wrap(err, "getting last processed ticks per epoch")
		}
		var epoch uint32
		for e, tick := range ticksPerEpoch {
			if tick == tickNumber {
				epoch = e
				break
			}
		}
		return &protobuff.ProcessedTick{
			TickNumber: tickNumber,
			Epoch:      epoch,
		}, nil
	}

	var lpt protobuff.ProcessedTick
	if err := proto.Unmarshal(value, &lpt); err != nil {
		return nil, errors.Wrap(err, "unmarshalling lpt to protobuff type")
	}

	return &lpt, nil
}

func (s *PebbleStore) GetLastProcessedTicksPerEpoch(ctx context.Context) (map[uint32]uint32, error) {
	upperBound := append([]byte{LastProcessedTickPerEpoch}, []byte(strconv.FormatUint(maxTickNumber, 10))...)
	iter, err := s.db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{LastProcessedTickPerEpoch},
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, errors.Wrap(err, "creating iter")
	}
	defer iter.Close()

	ticksPerEpoch := make(map[uint32]uint32)
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()

		value, err := iter.ValueAndErr()
		if err != nil {
			return nil, errors.Wrap(err, "getting value from iter")
		}

		epochNumber := binary.BigEndian.Uint32(key[1:])
		tickNumber := binary.LittleEndian.Uint32(value)
		ticksPerEpoch[epochNumber] = tickNumber
	}

	return ticksPerEpoch, nil
}

func (s *PebbleStore) SetSkippedTicksInterval(ctx context.Context, skippedTick *protobuff.SkippedTicksInterval) error {
	newList := protobuff.SkippedTicksIntervalList{}
	current, err := s.GetSkippedTicksInterval(ctx)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return errors.Wrap(err, "getting skipped tick interval")
		}
	} else {
		newList.SkippedTicks = current.SkippedTicks
	}

	newList.SkippedTicks = append(newList.SkippedTicks, skippedTick)

	key := skippedTicksIntervalKey()
	serialized, err := proto.Marshal(&newList)
	if err != nil {
		return errors.Wrap(err, "serializing skipped tick proto")
	}

	err = s.db.Set(key, serialized, pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "setting skipped tick interval")
	}

	return nil
}

func (s *PebbleStore) GetSkippedTicksInterval(ctx context.Context) (*protobuff.SkippedTicksIntervalList, error) {
	key := skippedTicksIntervalKey()
	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting skipped tick interval")
	}
	defer closer.Close()

	var stil protobuff.SkippedTicksIntervalList
	if err := proto.Unmarshal(value, &stil); err != nil {
		return nil, errors.Wrap(err, "unmarshalling skipped tick interval to protobuff type")
	}

	return &stil, nil
}

func (s *PebbleStore) PutTransferTransactionsPerTick(ctx context.Context, identity string, tickNumber uint32, txs *protobuff.TransferTransactionsPerTick) error {
	key := identityTransferTransactionsPerTickKey(identity, tickNumber)

	serialized, err := proto.Marshal(txs)
	if err != nil {
		return errors.Wrap(err, "serializing tx proto")
	}

	err = s.db.Set(key, serialized, pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "setting transfer tx")
	}

	return nil
}

func (s *PebbleStore) GetTransferTransactions(ctx context.Context, identity string, startTick, endTick uint64) ([]*protobuff.TransferTransactionsPerTick, error) {
	partialKey := identityTransferTransactions(identity)
	iter, err := s.db.NewIter(&pebble.IterOptions{
		LowerBound: binary.BigEndian.AppendUint64(partialKey, startTick),
		UpperBound: binary.BigEndian.AppendUint64(partialKey, endTick+1),
	})
	if err != nil {
		return nil, errors.Wrap(err, "creating iter")
	}
	defer iter.Close()

	transferTxs := make([]*protobuff.TransferTransactionsPerTick, 0)

	for iter.First(); iter.Valid(); iter.Next() {
		value, err := iter.ValueAndErr()
		if err != nil {
			return nil, errors.Wrap(err, "getting value from iter")
		}

		var perTick protobuff.TransferTransactionsPerTick

		err = proto.Unmarshal(value, &perTick)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshalling transfer tx per tick to protobuff type")
		}

		transferTxs = append(transferTxs, &perTick)
	}

	return transferTxs, nil
}

func (s *PebbleStore) PutChainDigest(ctx context.Context, tickNumber uint32, digest []byte) error {
	key := chainDigestKey(tickNumber)

	err := s.db.Set(key, digest, pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "setting chain digest")
	}

	return nil
}

func (s *PebbleStore) GetChainDigest(ctx context.Context, tickNumber uint32) ([]byte, error) {
	key := chainDigestKey(tickNumber)
	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting chain digest")
	}
	defer closer.Close()

	return value, nil
}

func (s *PebbleStore) PutStoreDigest(ctx context.Context, tickNumber uint32, digest []byte) error {
	key := storeDigestKey(tickNumber)

	err := s.db.Set(key, digest, pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "setting chain digest")
	}

	return nil
}

func (s *PebbleStore) GetStoreDigest(ctx context.Context, tickNumber uint32) ([]byte, error) {
	key := storeDigestKey(tickNumber)
	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting chain digest")
	}
	defer closer.Close()

	return value, nil
}

func (s *PebbleStore) GetTickTransactionsStatus(ctx context.Context, tickNumber uint64) (*protobuff.TickTransactionsStatus, error) {
	key := tickTxStatusKey(tickNumber)
	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting transactions status")
	}
	defer closer.Close()

	var tts protobuff.TickTransactionsStatus
	if err := proto.Unmarshal(value, &tts); err != nil {
		return nil, errors.Wrap(err, "unmarshalling tick transactions status")
	}

	return &tts, err
}

func (s *PebbleStore) GetTransactionStatus(ctx context.Context, txID string) (*protobuff.TransactionStatus, error) {
	key := txStatusKey(txID)
	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting transaction status")
	}
	defer closer.Close()

	var ts protobuff.TransactionStatus
	if err := proto.Unmarshal(value, &ts); err != nil {
		return nil, errors.Wrap(err, "unmarshalling transaction status")
	}

	return &ts, err
}

func (s *PebbleStore) SetTickTransactionsStatus(ctx context.Context, tickNumber uint64, tts *protobuff.TickTransactionsStatus) error {
	key := tickTxStatusKey(tickNumber)
	batch := s.db.NewBatchWithSize(len(tts.Transactions) + 1)
	defer batch.Close()

	serialized, err := proto.Marshal(tts)
	if err != nil {
		return errors.Wrap(err, "serializing tts proto")
	}

	err = batch.Set(key, serialized, pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "setting tts data")
	}

	for _, tx := range tts.Transactions {
		key := txStatusKey(tx.TxId)

		serialized, err := proto.Marshal(tx)
		if err != nil {
			return errors.Wrap(err, "serializing tx status proto")
		}

		err = batch.Set(key, serialized, nil)
		if err != nil {
			return errors.Wrap(err, "setting tx status data")
		}
	}

	err = batch.Commit(pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "committing batch")
	}

	return nil
}

func (s *PebbleStore) getProcessedTickIntervalsPerEpoch(ctx context.Context, epoch uint32) (*protobuff.ProcessedTickIntervalsPerEpoch, error) {
	key := processedTickIntervalsPerEpochKey(epoch)
	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return &protobuff.ProcessedTickIntervalsPerEpoch{Intervals: make([]*protobuff.ProcessedTickInterval, 0), Epoch: epoch}, nil
		}

		return nil, errors.Wrap(err, "getting processed tick intervals per epoch from store")
	}
	defer closer.Close()

	var ptie protobuff.ProcessedTickIntervalsPerEpoch
	if err := proto.Unmarshal(value, &ptie); err != nil {
		return nil, errors.Wrap(err, "unmarshalling processed tick intervals per epoch")
	}

	return &ptie, nil
}

func (s *PebbleStore) SetProcessedTickIntervalPerEpoch(ctx context.Context, epoch uint32, ptie *protobuff.ProcessedTickIntervalsPerEpoch) error {
	key := processedTickIntervalsPerEpochKey(epoch)
	serialized, err := proto.Marshal(ptie)
	if err != nil {
		return errors.Wrap(err, "serializing ptie proto")
	}

	err = s.db.Set(key, serialized, pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "setting ptie")
	}

	return nil
}

func (s *PebbleStore) AppendProcessedTickInterval(ctx context.Context, epoch uint32, pti *protobuff.ProcessedTickInterval) error {
	existing, err := s.getProcessedTickIntervalsPerEpoch(ctx, epoch)
	if err != nil {
		return errors.Wrap(err, "getting existing processed tick intervals")
	}

	existing.Intervals = append(existing.Intervals, pti)

	err = s.SetProcessedTickIntervalPerEpoch(ctx, epoch, existing)
	if err != nil {
		return errors.Wrap(err, "setting ptie")
	}

	return nil
}

func (s *PebbleStore) GetProcessedTickIntervals(ctx context.Context) ([]*protobuff.ProcessedTickIntervalsPerEpoch, error) {
	upperBound := append([]byte{ProcessedTickIntervals}, []byte(strconv.FormatUint(maxTickNumber, 10))...)
	iter, err := s.db.NewIter(&pebble.IterOptions{
		LowerBound: []byte{ProcessedTickIntervals},
		UpperBound: upperBound,
	})
	if err != nil {
		return nil, errors.Wrap(err, "creating iter")
	}
	defer iter.Close()

	processedTickIntervals := make([]*protobuff.ProcessedTickIntervalsPerEpoch, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		value, err := iter.ValueAndErr()
		if err != nil {
			return nil, errors.Wrap(err, "getting value from iter")
		}

		var ptie protobuff.ProcessedTickIntervalsPerEpoch
		err = proto.Unmarshal(value, &ptie)
		if err != nil {
			return nil, errors.Wrap(err, "unmarshalling iter ptie")
		}
		processedTickIntervals = append(processedTickIntervals, &ptie)
	}

	return processedTickIntervals, nil
}

func (s *PebbleStore) SetEmptyTicksForEpoch(epoch uint32, emptyTicksCount uint32) error {
	key := emptyTicksPerEpochKey(epoch)

	value := make([]byte, 4)
	binary.LittleEndian.PutUint32(value, emptyTicksCount)

	err := s.db.Set(key, value, pebble.Sync)
	if err != nil {
		return errors.Wrapf(err, "saving emptyTickCount for epoch %d", epoch)
	}
	return nil
}

func (s *PebbleStore) GetEmptyTicksForEpoch(epoch uint32) (uint32, error) {
	key := emptyTicksPerEpochKey(epoch)

	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return 0, err
		}

		return 0, errors.Wrapf(err, "getting emptyTickCount for epoch %d", epoch)
	}
	defer closer.Close()

	emptyTicksCount := binary.LittleEndian.Uint32(value)

	return emptyTicksCount, nil
}

func (s *PebbleStore) GetEmptyTicksForEpochs(epochs []uint32) (map[uint32]uint32, error) {

	emptyTickMap := make(map[uint32]uint32, len(epochs))

	for _, epoch := range epochs {
		emptyTicks, err := s.GetEmptyTicksForEpoch(epoch)
		if err != nil {
			if !errors.Is(err, pebble.ErrNotFound) {
				return nil, errors.Wrapf(err, "getting empty ticks for epoch %d", epoch)
			}
		}
		emptyTickMap[epoch] = emptyTicks
	}

	return emptyTickMap, nil

}

func (s *PebbleStore) DeleteEmptyTicksKeyForEpoch(epoch uint32) error {
	key := emptyTicksPerEpochKey(epoch)

	err := s.db.Delete(key, pebble.Sync)
	if err != nil {
		return errors.Wrapf(err, "deleting empty ticks key for epoch %d", epoch)
	}
	return nil
}

func (s *PebbleStore) SetLastTickQuorumDataPerEpoch(quorumData *protobuff.QuorumTickData, epoch uint32) error {
	key := lastTickQuorumDataPerEpochKey(epoch)

	serialized, err := proto.Marshal(quorumData)
	if err != nil {
		return errors.Wrapf(err, "serializing quorum tick data for last tick of epoch %d", epoch)
	}

	err = s.db.Set(key, serialized, pebble.Sync)
	if err != nil {
		return errors.Wrapf(err, "setting last tick quorum tick data for epoch %d", epoch)
	}
	return nil
}

func (s *PebbleStore) GetLastTickQuorumDataPerEpoch(epoch uint32) (*protobuff.QuorumTickData, error) {
	key := lastTickQuorumDataPerEpochKey(epoch)

	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, err
		}
		return nil, errors.Wrapf(err, "getting last tick quorum data for epoch %d", epoch)
	}
	defer closer.Close()

	var quorumData *protobuff.QuorumTickData

	err = proto.Unmarshal(value, quorumData)
	if err != nil {
		return nil, errors.Wrapf(err, "deserializing quorum tick data for last tick of epoch %d", epoch)
	}

	return quorumData, nil
}
