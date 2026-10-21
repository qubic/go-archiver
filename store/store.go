package store

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/cockroachdb/pebble"
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

		return nil, fmt.Errorf("getting tick data: %w", err)
	}
	defer closer.Close()

	var td protobuff.TickData
	if err := proto.Unmarshal(value, &td); err != nil {
		return nil, fmt.Errorf("unmarshalling tick data to protobuff type: %w", err)
	}

	return &td, err
}

func (s *PebbleStore) SetTickData(ctx context.Context, tickNumber uint32, td *protobuff.TickData) error {
	key := tickDataKey(tickNumber)
	serialized, err := proto.Marshal(td)
	if err != nil {
		return fmt.Errorf("serializing td proto: %w", err)
	}

	err = s.db.Set(key, serialized, pebble.Sync)
	if err != nil {
		return fmt.Errorf("setting tick data: %w", err)
	}

	return nil
}

func (s *PebbleStore) SetQuorumTickData(ctx context.Context, tickNumber uint32, qtd *protobuff.QuorumTickDataStored) error {
	key := quorumTickDataKey(tickNumber)
	serialized, err := proto.Marshal(qtd)
	if err != nil {
		return fmt.Errorf("serializing qtdV2 proto: %w", err)
	}

	err = s.db.Set(key, serialized, pebble.Sync)
	if err != nil {
		return fmt.Errorf("setting quorum tick data: %w", err)
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

		return nil, fmt.Errorf("getting quorum tick data: %w", err)
	}
	defer closer.Close()

	var qtd protobuff.QuorumTickDataStored
	if err := proto.Unmarshal(value, &qtd); err != nil {
		return nil, fmt.Errorf("unmarshalling qtdV2 to protobuf type: %w", err)
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

		return nil, fmt.Errorf("getting quorum tick data: %w", err)
	}
	defer closer.Close()

	var computors protobuff.Computors
	if err := proto.Unmarshal(value, &computors); err != nil {
		return nil, fmt.Errorf("unmarshalling computors to protobuff type: %w", err)
	}

	return &computors, nil
}

func (s *PebbleStore) SetComputors(ctx context.Context, epoch uint32, computors *protobuff.Computors) error {
	key := computorsKey(epoch)

	serialized, err := proto.Marshal(computors)
	if err != nil {
		return fmt.Errorf("serializing computors proto: %w", err)
	}

	err = s.db.Set(key, serialized, pebble.Sync)
	if err != nil {
		return fmt.Errorf("setting computors: %w", err)
	}

	return nil
}

func (s *PebbleStore) SetTransactions(ctx context.Context, txs []*protobuff.Transaction) error {
	batch := s.db.NewBatchWithSize(len(txs))
	defer batch.Close()

	for _, tx := range txs {
		key, err := tickTxKey(tx.TxId)
		if err != nil {
			return fmt.Errorf("creating tx key for id: %s: %w", tx.TxId, err)
		}

		serialized, err := proto.Marshal(tx)
		if err != nil {
			return fmt.Errorf("serializing tx proto: %w", err)
		}

		err = batch.Set(key, serialized, nil)
		if err != nil {
			return fmt.Errorf("getting tick data: %w", err)
		}
	}

	if err := batch.Commit(pebble.Sync); err != nil {
		return fmt.Errorf("committing batch: %w", err)
	}

	return nil
}

func (s *PebbleStore) GetTickTransactions(ctx context.Context, tickNumber uint32) ([]*protobuff.Transaction, error) {
	td, err := s.GetTickData(ctx, tickNumber)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("getting tick data: %w", err)
	}

	txs := make([]*protobuff.Transaction, 0, len(td.TransactionIds))
	for _, txID := range td.TransactionIds {
		tx, err := s.GetTransaction(ctx, txID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrNotFound
			}

			return nil, fmt.Errorf("getting tx for id: %s: %w", txID, err)
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

		return nil, fmt.Errorf("getting tick data: %w", err)
	}

	txs := make([]*protobuff.Transaction, 0, len(td.TransactionIds))
	for _, txID := range td.TransactionIds {
		tx, err := s.GetTransaction(ctx, txID)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				return nil, ErrNotFound
			}

			return nil, fmt.Errorf("getting tx for id: %s: %w", txID, err)
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
		return nil, fmt.Errorf("getting tx key: %w", err)
	}

	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, fmt.Errorf("getting tx: %w", err)
	}
	defer closer.Close()

	var tx protobuff.Transaction
	if err := proto.Unmarshal(value, &tx); err != nil {
		return nil, fmt.Errorf("unmarshalling tx to protobuff type: %w", err)
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
		return fmt.Errorf("setting last processed tick: %w", err)
	}

	key = lastProcessedTickKey()
	serialized, err := proto.Marshal(lastProcessedTick)
	if err != nil {
		return fmt.Errorf("serializing skipped tick proto: %w", err)
	}

	err = batch.Set(key, serialized, pebble.Sync)
	if err != nil {
		return fmt.Errorf("setting last processed tick: %w", err)
	}

	err = batch.Commit(pebble.Sync)
	if err != nil {
		return fmt.Errorf("committing batch: %w", err)
	}

	ptie, err := s.getProcessedTickIntervalsPerEpoch(ctx, lastProcessedTick.Epoch)
	if err != nil {
		return fmt.Errorf("getting ptie: %w", err)
	}

	if len(ptie.Intervals) == 0 {
		ptie = &protobuff.ProcessedTickIntervalsPerEpoch{Epoch: lastProcessedTick.Epoch, Intervals: []*protobuff.ProcessedTickInterval{{InitialProcessedTick: lastProcessedTick.TickNumber, LastProcessedTick: lastProcessedTick.TickNumber}}}
	} else {
		ptie.Intervals[len(ptie.Intervals)-1].LastProcessedTick = lastProcessedTick.TickNumber
	}

	err = s.SetProcessedTickIntervalPerEpoch(ctx, lastProcessedTick.Epoch, ptie)
	if err != nil {
		return fmt.Errorf("setting ptie: %w", err)
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

		return nil, fmt.Errorf("getting last processed tick: %w", err)
	}
	defer closer.Close()

	var lpt protobuff.ProcessedTick
	if err := proto.Unmarshal(value, &lpt); err != nil {
		return nil, fmt.Errorf("unmarshalling lpt to protobuff type: %w", err)
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
		return nil, fmt.Errorf("creating iter: %w", err)
	}
	defer iter.Close()

	ticksPerEpoch := make(map[uint32]uint32)
	for iter.First(); iter.Valid(); iter.Next() {
		key := iter.Key()

		value, err := iter.ValueAndErr()
		if err != nil {
			return nil, fmt.Errorf("getting value from iter: %w", err)
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
			return fmt.Errorf("getting skipped tick interval: %w", err)
		}
	} else {
		newList.SkippedTicks = current.SkippedTicks
	}

	newList.SkippedTicks = append(newList.SkippedTicks, skippedTick)

	key := skippedTicksIntervalKey()
	serialized, err := proto.Marshal(&newList)
	if err != nil {
		return fmt.Errorf("serializing skipped tick proto: %w", err)
	}

	err = s.db.Set(key, serialized, pebble.Sync)
	if err != nil {
		return fmt.Errorf("setting skipped tick interval: %w", err)
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

		return nil, fmt.Errorf("getting skipped tick interval: %w", err)
	}
	defer closer.Close()

	var stil protobuff.SkippedTicksIntervalList
	if err := proto.Unmarshal(value, &stil); err != nil {
		return nil, fmt.Errorf("unmarshalling skipped tick interval to protobuff type: %w", err)
	}

	return &stil, nil
}

func (s *PebbleStore) PutTransferTransactionsPerTick(ctx context.Context, identity string, tickNumber uint32, txs *protobuff.TransferTransactionsPerTick) error {
	key := identityTransferTransactionsPerTickKey(identity, tickNumber)

	serialized, err := proto.Marshal(txs)
	if err != nil {
		return fmt.Errorf("serializing tx proto: %w", err)
	}

	err = s.db.Set(key, serialized, pebble.Sync)
	if err != nil {
		return fmt.Errorf("setting transfer tx: %w", err)
	}

	return nil
}

type Pageable struct {
	Page, Size uint32
}

type Sortable struct {
	Descending bool
}

type Filterable struct {
	ScOnly bool
}

func (s *PebbleStore) GetTransactionsForEntity(ctx context.Context, identity string, startTick, endTick uint64) ([]*protobuff.TransferTransactionsPerTick, error) {
	const limitForRequestWithoutPaging = 1000 // old implementation was unlimited.
	transfers, _, err := s.GetTransactionsForEntityPaged(ctx, identity, startTick, endTick,
		Pageable{Size: limitForRequestWithoutPaging},
		Sortable{},
		Filterable{},
	)
	return transfers, err
}

func (s *PebbleStore) GetTransactionsForEntityPaged(_ context.Context, identity string, startTick, endTick uint64, page Pageable, sort Sortable, filter Filterable) ([]*protobuff.TransferTransactionsPerTick, int, error) {

	var index, start, end int
	start = int(page.Page) * int(page.Size)
	end = start + int(page.Size)

	var transferTxs []*protobuff.TransferTransactionsPerTick
	transferTxs = make([]*protobuff.TransferTransactionsPerTick, 0, min(page.Size, 1000))

	partialKey := identityTransferTransactions(identity)
	iter, err := s.db.NewIter(&pebble.IterOptions{
		LowerBound: binary.BigEndian.AppendUint64(partialKey, startTick),
		UpperBound: binary.BigEndian.AppendUint64(partialKey, endTick+1),
	})
	if err != nil {
		return nil, -1, fmt.Errorf("creating iterator: %w", err)
	}
	defer iter.Close()

	if sort.Descending {
		for iter.Last(); iter.Valid(); iter.Prev() {
			index, transferTxs, err = getTransfersPage(iter, index, transferTxs, start, end, filter)
		}
	} else {
		for iter.First(); iter.Valid(); iter.Next() { // per tick
			index, transferTxs, err = getTransfersPage(iter, index, transferTxs, start, end, filter)
		}
	}
	if err != nil {
		return nil, -1, fmt.Errorf("getting transfers page: %w", err)
	}

	return transferTxs, index, nil
}

func getTransfersPage(iter *pebble.Iterator, index int, transferTxs []*protobuff.TransferTransactionsPerTick, pageStart int, pageEnd int, filter Filterable) (int, []*protobuff.TransferTransactionsPerTick, error) {
	value, err := iter.ValueAndErr()
	if err != nil {
		return -1, nil, fmt.Errorf("getting value from iter: %w", err)
	}

	var perTick protobuff.TransferTransactionsPerTick
	var toBeAdded *protobuff.TransferTransactionsPerTick

	err = proto.Unmarshal(value, &perTick)
	if err != nil {
		return -1, nil, fmt.Errorf("unmarshalling transfer tx per tick to protobuff type: %w", err)
	}

	transactions := filterTransactions(filter, &perTick)

	count := len(transactions)
	if count > 0 && index+count >= pageStart && index < pageEnd {

		startIndex := max(pageStart-index, 0) // if index < pageStart we need to skip first items
		endIndex := min(pageEnd-index, count)

		if index+count >= pageStart && endIndex > startIndex { // covers case index >= pageStart and index+count >= pageStart
			toBeAdded = &protobuff.TransferTransactionsPerTick{
				TickNumber:   perTick.GetTickNumber(),
				Identity:     perTick.GetIdentity(),
				Transactions: transactions[startIndex:endIndex],
			}
			transferTxs = append(transferTxs, toBeAdded)
		}
	}
	index += count
	return index, transferTxs, nil
}

func filterTransactions(filter Filterable, perTick *protobuff.TransferTransactionsPerTick) []*protobuff.Transaction {
	var transactions []*protobuff.Transaction
	if filter.ScOnly { // filter if necessary
		transactions = make([]*protobuff.Transaction, 0)
		for _, tx := range perTick.GetTransactions() {
			if tx.InputType != 0 {
				transactions = append(transactions, tx)
			}
		}
	} else {
		transactions = perTick.GetTransactions()
	}
	return transactions
}

func (s *PebbleStore) PutChainDigest(ctx context.Context, tickNumber uint32, digest []byte) error {
	key := chainDigestKey(tickNumber)

	err := s.db.Set(key, digest, pebble.Sync)
	if err != nil {
		return fmt.Errorf("setting chain digest: %w", err)
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

		return nil, fmt.Errorf("getting chain digest: %w", err)
	}
	defer closer.Close()

	return value, nil
}

func (s *PebbleStore) PutStoreDigest(ctx context.Context, tickNumber uint32, digest []byte) error {
	key := storeDigestKey(tickNumber)

	err := s.db.Set(key, digest, pebble.Sync)
	if err != nil {
		return fmt.Errorf("setting chain digest: %w", err)
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

		return nil, fmt.Errorf("getting chain digest: %w", err)
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

		return nil, fmt.Errorf("getting transactions status: %w", err)
	}
	defer closer.Close()

	var tts protobuff.TickTransactionsStatus
	if err := proto.Unmarshal(value, &tts); err != nil {
		return nil, fmt.Errorf("unmarshalling tick transactions status: %w", err)
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

		return nil, fmt.Errorf("getting transaction status: %w", err)
	}
	defer closer.Close()

	var ts protobuff.TransactionStatus
	if err := proto.Unmarshal(value, &ts); err != nil {
		return nil, fmt.Errorf("unmarshalling transaction status: %w", err)
	}

	return &ts, err
}

func (s *PebbleStore) SetTickTransactionsStatus(ctx context.Context, tickNumber uint64, tts *protobuff.TickTransactionsStatus) error {
	key := tickTxStatusKey(tickNumber)
	batch := s.db.NewBatchWithSize(len(tts.Transactions) + 1)
	defer batch.Close()

	serialized, err := proto.Marshal(tts)
	if err != nil {
		return fmt.Errorf("serializing tts proto: %w", err)
	}

	err = batch.Set(key, serialized, pebble.Sync)
	if err != nil {
		return fmt.Errorf("setting tts data: %w", err)
	}

	for _, tx := range tts.Transactions {
		key := txStatusKey(tx.TxId)

		serialized, err := proto.Marshal(tx)
		if err != nil {
			return fmt.Errorf("serializing tx status proto: %w", err)
		}

		err = batch.Set(key, serialized, nil)
		if err != nil {
			return fmt.Errorf("setting tx status data: %w", err)
		}
	}

	err = batch.Commit(pebble.Sync)
	if err != nil {
		return fmt.Errorf("committing batch: %w", err)
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

		return nil, fmt.Errorf("getting processed tick intervals per epoch from store: %w", err)
	}
	defer closer.Close()

	var ptie protobuff.ProcessedTickIntervalsPerEpoch
	if err := proto.Unmarshal(value, &ptie); err != nil {
		return nil, fmt.Errorf("unmarshalling processed tick intervals per epoch: %w", err)
	}

	return &ptie, nil
}

func (s *PebbleStore) SetProcessedTickIntervalPerEpoch(ctx context.Context, epoch uint32, ptie *protobuff.ProcessedTickIntervalsPerEpoch) error {
	key := processedTickIntervalsPerEpochKey(epoch)
	serialized, err := proto.Marshal(ptie)
	if err != nil {
		return fmt.Errorf("serializing ptie proto: %w", err)
	}

	err = s.db.Set(key, serialized, pebble.Sync)
	if err != nil {
		return fmt.Errorf("setting ptie: %w", err)
	}

	return nil
}

func (s *PebbleStore) AppendProcessedTickInterval(ctx context.Context, epoch uint32, pti *protobuff.ProcessedTickInterval) error {
	existing, err := s.getProcessedTickIntervalsPerEpoch(ctx, epoch)
	if err != nil {
		return fmt.Errorf("getting existing processed tick intervals: %w", err)
	}

	existing.Intervals = append(existing.Intervals, pti)

	err = s.SetProcessedTickIntervalPerEpoch(ctx, epoch, existing)
	if err != nil {
		return fmt.Errorf("setting ptie: %w", err)
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
		return nil, fmt.Errorf("creating iter: %w", err)
	}
	defer iter.Close()

	processedTickIntervals := make([]*protobuff.ProcessedTickIntervalsPerEpoch, 0)
	for iter.First(); iter.Valid(); iter.Next() {
		value, err := iter.ValueAndErr()
		if err != nil {
			return nil, fmt.Errorf("getting value from iter: %w", err)
		}

		var ptie protobuff.ProcessedTickIntervalsPerEpoch
		err = proto.Unmarshal(value, &ptie)
		if err != nil {
			return nil, fmt.Errorf("unmarshalling iter ptie: %w", err)
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
		return fmt.Errorf("saving emptyTickCount for epoch %d: %w", epoch, err)
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

		return 0, fmt.Errorf("getting emptyTickCount for epoch %d: %w", epoch, err)
	}
	defer closer.Close()

	emptyTicksCount := binary.LittleEndian.Uint32(value)

	return emptyTicksCount, nil
}

func (s *PebbleStore) GetEmptyTicksForEpochs(firstEpoch, lastEpoch uint32) (map[uint32]uint32, error) {

	iter, err := s.db.NewIter(&pebble.IterOptions{
		LowerBound: emptyTicksPerEpochKey(firstEpoch),
		UpperBound: emptyTicksPerEpochKey(lastEpoch + 1), // Increment as upper bound is exclusive
	})
	if err != nil {
		return nil, fmt.Errorf("creating iter: %w", err)
	}
	defer iter.Close()

	emptyTickMap := make(map[uint32]uint32)

	for iter.First(); iter.Valid(); iter.Next() {

		value, err := iter.ValueAndErr()
		if err != nil {
			return nil, fmt.Errorf("getting value from iter: %w", err)
		}

		key := iter.Key()
		epochNumber := binary.BigEndian.Uint64(key[1:])
		emptyTicksCount := binary.LittleEndian.Uint32(value)

		emptyTickMap[uint32(epochNumber)] = emptyTicksCount
	}

	return emptyTickMap, nil
}

func (s *PebbleStore) DeleteEmptyTicksKeyForEpoch(epoch uint32) error {
	key := emptyTicksPerEpochKey(epoch)

	err := s.db.Delete(key, pebble.Sync)
	if err != nil {
		return fmt.Errorf("deleting empty ticks key for epoch %d: %w", epoch, err)
	}
	return nil
}

func (s *PebbleStore) SetLastTickQuorumDataPerEpochIntervals(epoch uint32, lastQuorumDataPerEpochIntervals *protobuff.LastTickQuorumDataPerEpochIntervals) error {

	key := lastTickQuorumDataPerEpochIntervalKey(epoch)

	value, err := proto.Marshal(lastQuorumDataPerEpochIntervals)
	if err != nil {
		return fmt.Errorf("serializing last quorum data per epoch intervals for epoch %d: %w", epoch, err)
	}

	err = s.db.Set(key, value, pebble.Sync)
	if err != nil {
		return fmt.Errorf("setting last quorum data per epoch intervals for epoch %d: %w", epoch, err)
	}
	return nil
}

func (s *PebbleStore) GetLastTickQuorumDataListPerEpochInterval(epoch uint32) (*protobuff.LastTickQuorumDataPerEpochIntervals, error) {
	key := lastTickQuorumDataPerEpochIntervalKey(epoch)

	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return &protobuff.LastTickQuorumDataPerEpochIntervals{
				QuorumDataPerInterval: make(map[int32]*protobuff.QuorumTickData),
			}, nil
		}
		return nil, fmt.Errorf("getting quorum data list for the intervals of epoch %d: %w", epoch, err)
	}
	defer closer.Close()

	var lastQuorumDataPerEpochIntervals protobuff.LastTickQuorumDataPerEpochIntervals
	err = proto.Unmarshal(value, &lastQuorumDataPerEpochIntervals)
	if err != nil {
		return nil, fmt.Errorf("de-serializing last quorum data per epoch intervals for epoch %d: %w", epoch, err)
	}

	return &lastQuorumDataPerEpochIntervals, err
}

func (s *PebbleStore) SetQuorumDataForCurrentEpochInterval(epoch uint32, quorumData *protobuff.QuorumTickData) error {

	processedIntervals, err := s.getProcessedTickIntervalsPerEpoch(nil, epoch)
	if err != nil {
		return fmt.Errorf("getting processed tick intervals for epoch %d: %w", epoch, err)
	}

	intervalIndex := len(processedIntervals.Intervals) - 1
	if intervalIndex < 0 {
		intervalIndex = 0
	}

	quorumDataPerIntervals, err := s.GetLastTickQuorumDataListPerEpochInterval(epoch)
	if err != nil {
		return fmt.Errorf("getting last quorum data list for epoch intervals: %w", err)
	}

	quorumDataPerIntervals.QuorumDataPerInterval[int32(intervalIndex)] = quorumData

	err = s.SetLastTickQuorumDataPerEpochIntervals(epoch, quorumDataPerIntervals)
	if err != nil {
		return fmt.Errorf("setting last quorum data list for epoch intervals: %w", err)
	}

	return nil
}

func (s *PebbleStore) SetEmptyTickListPerEpoch(epoch uint32, emptyTicks []uint32) error {
	key := emptyTickListPerEpochKey(epoch)

	value := make([]byte, len(emptyTicks)*4)
	for index, tickNumber := range emptyTicks {
		binary.LittleEndian.PutUint32(value[index*4:index*4+4], tickNumber)
	}

	err := s.db.Set(key, value, pebble.Sync)
	if err != nil {
		return fmt.Errorf("saving empty tick list for epoch %d: %w", epoch, err)
	}
	return nil
}

func (s *PebbleStore) GetEmptyTickListPerEpoch(epoch uint32) ([]uint32, error) {
	key := emptyTickListPerEpochKey(epoch)

	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, err
		}

		return nil, fmt.Errorf("getting empty tick list for epoch %d: %w", epoch, err)
	}
	defer closer.Close()

	if len(value)%4 != 0 {
		return nil, fmt.Errorf("corrupted empty tick list for epoch %d. array length mod 4 != 0. length: %d", epoch, len(value))
	}

	var emptyTicks []uint32

	for index := 0; index < (len(value) / 4); index++ {
		tickNumber := binary.LittleEndian.Uint32(value[index*4 : index*4+4])
		emptyTicks = append(emptyTicks, tickNumber)
	}

	return emptyTicks, nil

}

func (s *PebbleStore) AppendEmptyTickToEmptyTickListPerEpoch(epoch uint32, tickNumber uint32) error {

	emptyTicks, err := s.GetEmptyTickListPerEpoch(epoch)
	if err != nil {
		return fmt.Errorf("getting empty tick list for epoch %d: %w", epoch, err)
	}

	emptyTicks = append(emptyTicks, tickNumber)

	err = s.SetEmptyTickListPerEpoch(epoch, emptyTicks)
	if err != nil {
		return fmt.Errorf("saving appended empty tick list: %w", err)
	}

	return nil
}

func (s *PebbleStore) DeleteEmptyTickListKeyForEpoch(epoch uint32) error {
	key := emptyTickListPerEpochKey(epoch)

	err := s.db.Delete(key, pebble.Sync)
	if err != nil {
		return fmt.Errorf("deleting empty tick list key for epoch %d: %w", epoch, err)
	}
	return nil
}

func (s *PebbleStore) SetTargetTickVoteSignature(epoch, value uint32) error {
	key := targetTickVoteSignatureKey(epoch)

	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, value)

	err := s.db.Set(key, data, pebble.Sync)
	if err != nil {
		return fmt.Errorf("saving target tick vote signature for epoch %d: %w", epoch, err)
	}
	return nil
}

func (s *PebbleStore) GetTargetTickVoteSignature(epoch uint32) (uint32, error) {
	key := targetTickVoteSignatureKey(epoch)

	value, closer, err := s.db.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("getting target tick vote signature: %w", err)
	}
	defer closer.Close()

	return binary.LittleEndian.Uint32(value), nil
}
