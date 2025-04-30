package store

import (
	"cmp"
	"encoding/binary"
	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"log"
	"path/filepath"
	"slices"
	"sync"
)

const maxTickNumber = ^uint64(0)

var ErrNotFound = errors.New("store resource not found")

type PebbleStore struct {
	maxEpochCount int

	storeInfo *stores

	storagePath string

	infoDb *pebble.DB
	logger *zap.Logger
}

func NewPebbleStore(storagePath string, logger *zap.Logger, epochCount int) (*PebbleStore, error) {

	infoDbPath := storagePath + string(filepath.Separator) + "info"

	infoDb, err := pebble.Open(infoDbPath, &pebble.Options{}) // Use default options
	if err != nil {
		return nil, errors.Wrap(err, "opening info db")
	}

	ps := PebbleStore{
		maxEpochCount: epochCount,
		storeInfo:     &stores{},
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

	loadedEpochs := make(map[uint32]*EpochStore)

	for _, epoch := range relevantEpochs {
		if epoch == 0 {
			continue
		}
		epochStore, err := NewEpochStore(storagePath, epoch)
		if err != nil {
			return nil, errors.Wrapf(err, "creating store for epoch %d", epoch)
		}

		if lastProcessedTick != nil && epoch == lastProcessedTick.Epoch {
			ps.storeInfo.setCurrentEpochStore(epochStore)
		}
		loadedEpochs[epoch] = epochStore
	}
	ps.storeInfo.setLoadedEpochs(loadedEpochs)

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

	currentEpochStore := s.storeInfo.getCurrentEpochStore()
	err = currentEpochStore.SetLastProcessedTick(processedTick.TickNumber)
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
			}, ErrNotFound
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
	return s.storeInfo.getCurrentEpochStore()
}

func (s *PebbleStore) HandleEpochTransition(epoch uint32) error {

	epochStore, err := NewEpochStore(s.storagePath, epoch)
	if err != nil {
		return errors.Wrap(err, "creating new epoch store")
	}

	s.storeInfo.setCurrentEpochStore(epochStore)

	loadedEpochs := s.storeInfo.getLoadedEpochs()
	loadedEpochs[epoch] = epochStore

	epochs := getRelevantEpochListFromMap(loadedEpochs)

	log.Printf("Epoch list pre transition: %v\n", epochs)

	if len(epochs) > s.maxEpochCount {
		slices.Sort(epochs)

		discardedEpochs := epochs[:len(epochs)-s.maxEpochCount]
		for _, e := range discardedEpochs {

			//Commented for now. Closing the DB will hang for an unknown period of time. If the program is terminated before this finished, the updated list will not be saved, leading to a broken state
			//oldStore := s.loadedEpochs[e]
			delete(loadedEpochs, e)
			//err := oldStore.pebbleDb.Close()
			//return errors.Wrapf(err, "closing old epoch store %d", e)
		}
	}
	s.storeInfo.setLoadedEpochs(loadedEpochs)

	epochs = getRelevantEpochListFromMap(loadedEpochs)

	log.Printf("Epoch list post transition: %v\n", epochs)

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

	loadedEpochs := s.storeInfo.getLoadedEpochs()
	for _, store := range loadedEpochs {
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

	loadedEpochs := s.storeInfo.getLoadedEpochs()
	for _, store := range loadedEpochs {
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

	loadedEpochs := s.storeInfo.getLoadedEpochs()
	for _, store := range loadedEpochs {
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

	loadedEpochs := s.storeInfo.getLoadedEpochs()
	for _, epochStore := range loadedEpochs {
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

	loadedEpochs := s.storeInfo.getLoadedEpochs()
	for _, epochStore := range loadedEpochs {
		processedTickIntervals, err := epochStore.GetProcessedTickIntervals()
		if err != nil {
			return nil, errors.Wrapf(err, "getting processed tick intervals of epoch %d", epochStore.epoch)
		}
		processedTickIntervalsPerEpoch = append(processedTickIntervalsPerEpoch, processedTickIntervals)
	}
	slices.SortFunc(processedTickIntervalsPerEpoch, func(a, b *protobuff.ProcessedTickIntervalsPerEpoch) int {
		return cmp.Compare(a.Epoch, b.Epoch)
	})
	return processedTickIntervalsPerEpoch, nil

}

func (s *PebbleStore) Close() {
	s.infoDb.Close()
	s.storeInfo.closeStores()
}

type stores struct {
	currentEpochStore *EpochStore
	loadedEpochs      map[uint32]*EpochStore
	mutex             sync.RWMutex
}

func (s *stores) getCurrentEpochStore() *EpochStore {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.currentEpochStore

}

func (s *stores) setCurrentEpochStore(store *EpochStore) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.currentEpochStore = store
}

func (s *stores) getLoadedEpochs() map[uint32]*EpochStore {
	s.mutex.RLock()
	defer s.mutex.RUnlock()
	return s.loadedEpochs
}

func (s *stores) setLoadedEpochs(loadedEpochs map[uint32]*EpochStore) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.loadedEpochs = loadedEpochs
}

func (s *stores) closeStores() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	for _, epochStore := range s.loadedEpochs {
		err := epochStore.pebbleDb.Close()
		if err != nil {
			return errors.Wrapf(err, "closing store for epoch %d", epochStore.epoch)
		}
	}
	return nil
}

const (
	relevantEpochsKey = 0xb3
)
