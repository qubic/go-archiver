package store

import (
	"encoding/binary"
	"fmt"
	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"path/filepath"
	"runtime"
	"slices"
)

const maxTickNumber = ^uint64(0)

var ErrNotFound = errors.New("store resource not found")

type PebbleStore struct {
	maxEpochCount     int
	loadedEpochs      map[uint32]*EpochStore
	currentEpochStore *EpochStore

	storagePath string

	infoDb *pebble.DB
	logger *zap.Logger
}

func NewPebbleStore(storagePath string, logger *zap.Logger, epochCount int) (*PebbleStore, error) {

	infoDbPath := storagePath + string(filepath.Separator) + "info"

	dbOptions := getDBOptions()
	infoDb, err := pebble.Open(infoDbPath, &dbOptions)
	if err != nil {
		return nil, errors.Wrap(err, "opening db with zstd compression")
	}

	ps := PebbleStore{
		maxEpochCount: epochCount,
		loadedEpochs:  make(map[uint32]*EpochStore),
		storagePath:   storagePath,
		infoDb:        infoDb,
		logger:        logger,
	}

	lastProcessedTick, err := ps.GetLastProcessedTick()
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return nil, errors.Wrap(err, "failed to get last processed tick")
		}
	}

	relevantEpochs, err := ps.GetRelevantEpochList()
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			return nil, errors.Wrap(err, "failed to get relevant epoch list")
		}
	}

	for _, epoch := range relevantEpochs {
		if epoch == 0 {
			continue
		}
		epochStore, err := NewEpochStore(storagePath, epoch)
		if err != nil {
			return nil, errors.Wrapf(err, "creating store for epoch %d", epoch)
		}

		if lastProcessedTick != nil && epoch == lastProcessedTick.Epoch {
			ps.currentEpochStore = epochStore
		}
		ps.loadedEpochs[epoch] = epochStore
	}

	return &ps, err
}

func (s *PebbleStore) SetLastProcessedTick(processedTick *protobuff.ProcessedTick) error {

	key := []byte{lastProcessedTickKey}
	value, err := proto.Marshal(processedTick)
	if err != nil {
		return errors.Wrap(err, "marshalling last processed tick")
	}
	err = s.infoDb.Set(key, value, pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "storing last processed tick")
	}

	err = s.currentEpochStore.SetLastProcessedTick(processedTick.TickNumber)
	if err != nil {
		return errors.Wrap(err, "storing last processed tick to epoch storage")
	}

	return nil
}

func (s *PebbleStore) GetLastProcessedTick() (*protobuff.ProcessedTick, error) {

	key := []byte{lastProcessedTickKey}

	value, closer, err := s.infoDb.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return &protobuff.ProcessedTick{
				TickNumber: 0,
				Epoch:      0,
			}, nil
		}

		return nil, errors.Wrap(err, "getting last processed tick")
	}
	defer closer.Close()

	var lastProcessedTick protobuff.ProcessedTick
	err = proto.Unmarshal(value, &lastProcessedTick)
	if err != nil {
		return nil, errors.Wrap(err, "unmarshalling last processed tick")
	}

	return &lastProcessedTick, nil
}

func (s *PebbleStore) SetRelevantEpochList(epochs []uint32) error {

	key := []byte{relevantEpochsKey}

	value := make([]byte, len(epochs)*4)
	for index, epoch := range epochs {
		binary.LittleEndian.PutUint32(value[index*4:index*4+4], epoch)
	}

	err := s.infoDb.Set(key, value, pebble.Sync)
	if err != nil {
		return errors.Wrap(err, "storing relevant epoch list")
	}

	return nil
}

func (s *PebbleStore) GetRelevantEpochList() ([]uint32, error) {

	key := []byte{relevantEpochsKey}

	value, closer, err := s.infoDb.Get(key)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return []uint32{}, nil
		}

		return nil, errors.Wrap(err, "getting relevant epoch list")
	}
	defer closer.Close()

	epochs := make([]uint32, len(value)/4)

	for index := 0; index < len(value)/4; index++ {
		epoch := binary.LittleEndian.Uint32(value[index*4 : index*4+4])
		epochs = append(epochs, epoch)
	}

	return epochs, nil
}

func (s *PebbleStore) GetCurrentEpochStore() *EpochStore {
	return s.currentEpochStore
}

func (s *PebbleStore) HandleEpochTransition(epoch uint32) error {

	epochStore, err := NewEpochStore(s.storagePath, epoch)
	if err != nil {
		return errors.Wrap(err, "creating new epoch store")
	}

	s.currentEpochStore = epochStore
	s.loadedEpochs[epoch] = epochStore

	epochs := getRelevantEpochListFromMap(s.loadedEpochs)

	fmt.Printf("DEBUG: Epoch list pre cleanup: %v\n", epochs)

	if len(epochs) > s.maxEpochCount {
		slices.Sort(epochs)

		discardedEpochs := epochs[:len(epochs)-s.maxEpochCount]
		for _, e := range discardedEpochs {

			//Commented for now. Closing the DB will hang for an unknown period of time. If the program is terminated before this finished, the updated list will not be saved, leading to a broken state
			//oldStore := s.loadedEpochs[e]
			delete(s.loadedEpochs, e)
			//err := oldStore.pebbleDb.Close()
			//return errors.Wrapf(err, "closing old epoch store %d", e)
		}

	}

	epochs = getRelevantEpochListFromMap(s.loadedEpochs)

	fmt.Printf("DEBUG: Epoch list post cleanup: %v\n", epochs)

	err = s.SetRelevantEpochList(epochs)
	if err != nil {
		return errors.Wrap(err, "saving relevant epoch list")
	}

	return nil
}

func getRelevantEpochListFromMap(epochMap map[uint32]*EpochStore) []uint32 {
	var epochs []uint32
	for e, _ := range epochMap {
		epochs = append(epochs, e)
	}

	return epochs
}

func (s *PebbleStore) GetTickData(tickNumber uint32) (*protobuff.TickData, error) {
	for _, store := range s.loadedEpochs {
		tickData, err := store.GetTickData(tickNumber)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, errors.Wrap(err, "finding tick data")
		}

		return tickData, nil
	}

	return nil, errors.Errorf("unable to find tick data for tick %d", tickNumber)
}

func (s *PebbleStore) GetTickTransactions(tickNumber uint32) ([]*protobuff.Transaction, error) {

	var transactions []*protobuff.Transaction

	for _, store := range s.loadedEpochs {
		tickData, err := store.GetTickData(tickNumber)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, errors.Wrap(err, "finding tick transactions")
		}

		for _, txId := range tickData.TransactionIds {
			transaction, err := store.GetTransaction(txId)
			if err != nil {
				return nil, errors.Wrapf(err, "fetching transaction for tick %d", tickNumber)
			}
			transactions = append(transactions, transaction)
		}

		return transactions, nil

	}

	return nil, errors.Errorf("unable to find transactions for tick %d", tickNumber)
}

func (s *PebbleStore) GetTransaction(txId string) (*protobuff.Transaction, error) {
	for _, store := range s.loadedEpochs {
		transaction, err := store.GetTransaction(txId)
		if err != nil {
			if errors.Is(err, ErrNotFound) {
				continue
			}
			return nil, errors.Wrap(err, "finding transaction")
		}

		return transaction, nil
	}

	return nil, errors.Errorf("unable to find transaction %s", txId)
}

func (s *PebbleStore) GetLastProcessedTicksPerEpoch() (map[uint32]uint32, error) {

	epochLastProcessedTickMap := make(map[uint32]uint32)

	for _, epochStore := range s.loadedEpochs {
		lastProcessedTick, err := epochStore.GetLastProcessedTick()
		if err != nil {
			return nil, errors.Wrapf(err, "getting last processed tick of epoch %d", epochStore.epoch)
		}
		epochLastProcessedTickMap[epochStore.epoch] = lastProcessedTick
	}

	return epochLastProcessedTickMap, nil
}

func (s *PebbleStore) GetProcessedTickIntervals() ([]*protobuff.ProcessedTickIntervalsPerEpoch, error) {

	var processedTickIntervalsPerEpoch []*protobuff.ProcessedTickIntervalsPerEpoch

	for _, epochStore := range s.loadedEpochs {
		processedTickIntervals, err := epochStore.GetProcessedTickIntervals()
		if err != nil {
			return nil, errors.Wrapf(err, "getting processed tick intervals of epoch %d", epochStore.epoch)
		}
		processedTickIntervalsPerEpoch = append(processedTickIntervalsPerEpoch, processedTickIntervals)
	}
	return processedTickIntervalsPerEpoch, nil

}

func (s *PebbleStore) Close() {
	s.infoDb.Close()

	for _, store := range s.loadedEpochs {
		store.pebbleDb.Close()
	}
}

func getDBOptions() pebble.Options {
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

	return pebble.Options{
		Levels:                   []pebble.LevelOptions{l1Options, l2Options, l3Options, l4Options},
		MaxConcurrentCompactions: func() int { return runtime.NumCPU() },
		MemTableSize:             268435456, // 256 MB
		EventListener:            NewPebbleEventListener(),
	}
}

const (
	relevantEpochsKey = 0xb3
)
