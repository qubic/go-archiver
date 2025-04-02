package store

import (
	"encoding/binary"
	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"google.golang.org/protobuf/proto"
	"path/filepath"
	"strconv"
)

//var ErrNotFound = errors.New("store resource not found")

type EpochStore struct {
	pebbleDb *pebble.DB
	epoch    uint32
}

func NewEpochStore(storageDirPath string, epoch uint32) (*EpochStore, error) {

	dbPath := storageDirPath + string(filepath.Separator) + strconv.Itoa(int(epoch))

	db, err := pebble.Open(dbPath, &pebble.Options{}) // TODO: Change to optimized settings
	if err != nil {
		return nil, errors.Wrapf(err, "opening db for epoch %d", epoch)
	}

	return &EpochStore{
		pebbleDb: db,
		epoch:    epoch,
	}, nil
}

func (es *EpochStore) SetComputorList(computorList *protobuff.Computors) error {

	key := []byte{computorListKey}
	serialized, err := proto.Marshal(computorList)
	if err != nil {
		return errors.Wrap(err, "marshalling computor list object")
	}
	err = es.pebbleDb.Set(key, serialized, pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "storing computor list")
	}
	return nil
}

func (es *EpochStore) GetComputorList() (*protobuff.Computors, error) {

	key := []byte{computorListKey}
	value, closer, err := es.pebbleDb.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting computor list")
	}
	defer closer.Close()

	var computorList protobuff.Computors
	if err := proto.Unmarshal(value, &computorList); err != nil {
		return nil, errors.Wrap(err, "unmarshalling computor list object")
	}
	return &computorList, err

}

func (es *EpochStore) SetTickData(tickNumber uint32, tickData *protobuff.TickData) error {

	key := assembleKey(tickDataKey, tickNumber)
	serialized, err := proto.Marshal(tickData)
	if err != nil {
		return errors.Wrap(err, "marshalling tick data object")
	}
	err = es.pebbleDb.Set(key, serialized, pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "storing tick data")
	}
	return nil
}

func (es *EpochStore) GetTickData(tickNumber uint32) (*protobuff.TickData, error) {

	key := assembleKey(tickDataKey, tickNumber)
	value, closer, err := es.pebbleDb.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting tick data")
	}
	defer closer.Close()

	var tickData protobuff.TickData
	if err := proto.Unmarshal(value, &tickData); err != nil {
		return nil, errors.Wrap(err, "unmarshalling tick data object")
	}
	return &tickData, err
}

func (es *EpochStore) SetTransactions(transactions []*protobuff.Transaction) error {

	batch := es.pebbleDb.NewBatchWithSize(len(transactions))
	defer batch.Close()

	for _, tx := range transactions {
		key := assembleKey(transactionKey, tx.TxId)
		serialized, err := proto.Marshal(tx)
		if err != nil {
			return errors.Wrap(err, "marshalling transaction object")
		}
		err = batch.Set(key, serialized, nil)
		if err != nil {
			return errors.Wrap(err, "adding transaction to batch")
		}
	}
	if err := batch.Commit(pebble.Sync); err != nil {
		return errors.Wrap(err, "committing transaction batch")
	}
	return nil
}

func (es *EpochStore) GetTransaction(txId string) (*protobuff.Transaction, error) {

	key := assembleKey(transactionKey, txId)
	value, closer, err := es.pebbleDb.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, ErrNotFound
		}

		return nil, errors.Wrap(err, "getting transaction")
	}
	defer closer.Close()

	var transaction protobuff.Transaction
	if err := proto.Unmarshal(value, &transaction); err != nil {
		return nil, errors.Wrap(err, "unmarshalling transaction")
	}

	return &transaction, nil
}

func (es *EpochStore) SetLastProcessedTick(tickNumber uint32) error {

	key := []byte{lastProcessedTickKey}

	value := make([]byte, 4)
	binary.LittleEndian.PutUint32(value, tickNumber)

	err := es.pebbleDb.Set(key, value, pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "storing last processed tick")
	}

	processedTickIntervals, err := es.GetProcessedTickIntervals()
	if err != nil {
		return errors.Wrap(err, "getting processed tick intervals")
	}

	if len(processedTickIntervals.Intervals) == 0 {
		processedTickIntervals = &protobuff.ProcessedTickIntervalsPerEpoch{Epoch: es.epoch, Intervals: []*protobuff.ProcessedTickInterval{{InitialProcessedTick: tickNumber, LastProcessedTick: tickNumber}}}
	} else {
		processedTickIntervals.Intervals[len(processedTickIntervals.Intervals)-1].LastProcessedTick = tickNumber
	}

	err = es.SetProcessedTickIntervals(processedTickIntervals)
	if err != nil {
		return errors.Wrap(err, "updating tick intervals")
	}

	return nil
}

func (es *EpochStore) GetLastProcessedTick() (uint32, error) {
	key := []byte{lastProcessedTickKey}

	value, closer, err := es.pebbleDb.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return 0, ErrNotFound
		}

		return 0, errors.Wrap(err, "getting last processed tick")
	}
	defer closer.Close()

	return binary.LittleEndian.Uint32(value), nil
}

func (es *EpochStore) SetProcessedTickIntervals(processedTickIntervals *protobuff.ProcessedTickIntervalsPerEpoch) error {

	key := []byte{processedTickIntervalsKey}
	serialized, err := proto.Marshal(processedTickIntervals)
	if err != nil {
		return errors.Wrap(err, "marshalling processed tick intervals object")
	}
	err = es.pebbleDb.Set(key, serialized, pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "storing processed tick intervals")
	}
	return nil
}

func (es *EpochStore) GetProcessedTickIntervals() (*protobuff.ProcessedTickIntervalsPerEpoch, error) {
	key := []byte{processedTickIntervalsKey}
	value, closer, err := es.pebbleDb.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return &protobuff.ProcessedTickIntervalsPerEpoch{Intervals: make([]*protobuff.ProcessedTickInterval, 0), Epoch: es.epoch}, nil
		}
		return nil, errors.Wrap(err, "getting processed tick intervals")
	}
	defer closer.Close()

	var processedTickIntervals protobuff.ProcessedTickIntervalsPerEpoch
	if err := proto.Unmarshal(value, &processedTickIntervals); err != nil {
		return nil, errors.Wrap(err, "unmarshalling processed tick intervals")
	}

	return &processedTickIntervals, nil
}

func (es *EpochStore) AppendProcessedTickInterval(tickInterval *protobuff.ProcessedTickInterval) error {

	existing, err := es.GetProcessedTickIntervals()
	if err != nil {
		return errors.Wrap(err, "getting existing processed tick intervals")
	}
	existing.Intervals = append(existing.Intervals, tickInterval)
	err = es.SetProcessedTickIntervals(existing)
	if err != nil {
		return errors.Wrap(err, "storing updated processed tick intervals")
	}
	return nil
}

const (
	computorListKey = 0xa0
	tickDataKey     = 0xa1
	transactionKey

	lastProcessedTickKey      = 0xb0
	processedTickIntervalsKey = 0xb1
)

type iDType interface {
	uint32 | uint64 | string
}

func assembleKey[T iDType](keyPrefix int, id T) []byte {

	prefix := byte(keyPrefix)

	key := []byte{prefix}

	switch any(id).(type) {

	case uint32:
		asserted := any(id).(uint32)
		key = binary.BigEndian.AppendUint64(key, uint64(asserted))
		break

	case uint64:
		asserted := any(id).(uint64)
		key = binary.BigEndian.AppendUint64(key, asserted)
		break

	case string:
		asserted := any(id).(string)
		key = append(key, []byte(asserted)...)
	}
	return key
}

// Compaction options. To be added back
/*

l1Options := pebble.LevelOptions{
		BlockRestartInterval: 16,
		BlockSize:            4096,
		BlockSizeThreshold:   90,
		Compression:          pebble.NoCompression,
		FilterPolicy:         nil,
		FilterType:           pebble.TableFilter,
		IndexBlockSize:       4096,
		TargetFileSize:       268435456, // 256 MB
	}
	l2Options := pebble.LevelOptions{
		BlockRestartInterval: 16,
		BlockSize:            4096,
		BlockSizeThreshold:   90,
		Compression:          pebble.ZstdCompression,
		FilterPolicy:         nil,
		FilterType:           pebble.TableFilter,
		IndexBlockSize:       4096,
		TargetFileSize:       l1Options.TargetFileSize * 10, // 2.5 GB
	}
	l3Options := pebble.LevelOptions{
		BlockRestartInterval: 16,
		BlockSize:            4096,
		BlockSizeThreshold:   90,
		Compression:          pebble.ZstdCompression,
		FilterPolicy:         nil,
		FilterType:           pebble.TableFilter,
		IndexBlockSize:       4096,
		TargetFileSize:       l2Options.TargetFileSize * 10, // 25 GB
	}
	l4Options := pebble.LevelOptions{
		BlockRestartInterval: 16,
		BlockSize:            4096,
		BlockSizeThreshold:   90,
		Compression:          pebble.ZstdCompression,
		FilterPolicy:         nil,
		FilterType:           pebble.TableFilter,
		IndexBlockSize:       4096,
		TargetFileSize:       l3Options.TargetFileSize * 10, // 250 GB
	}

	pebbleOptions := pebble.Options{
		Levels:                   []pebble.LevelOptions{l1Options, l2Options, l3Options, l4Options},
		MaxConcurrentCompactions: func() int { return runtime.NumCPU() },
		MemTableSize:             268435456, // 256 MB
		EventListener:            store.NewPebbleEventListener(),
	}
*/
