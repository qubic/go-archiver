package tx

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/cockroachdb/pebble"
	"path/filepath"
)

var ErrNotFound = errors.New("store resource not found")

const lastProcessedTickPerEpochKey = 0x00

type ProcessorStore struct {
	db *pebble.DB
}

func NewProcessorStore(storeDir string) (*ProcessorStore, error) {
	db, err := pebble.Open(filepath.Join(storeDir, "lts-processor-internalStore"), &pebble.Options{})
	if err != nil {
		return nil, fmt.Errorf("opening pebble db: %v", err)
	}

	return &ProcessorStore{db: db}, nil
}

func (ps *ProcessorStore) SetLastProcessedTick(epoch, tick uint32) error {
	key := []byte{lastProcessedTickPerEpochKey}
	key = binary.BigEndian.AppendUint32(key, epoch)

	var value []byte
	value = binary.BigEndian.AppendUint32(value, tick)

	err := ps.db.Set(key, value, pebble.Sync)
	if err != nil {
		return fmt.Errorf("setting last processed tick: %v", err)
	}

	return nil
}

func (ps *ProcessorStore) GetLastProcessedTick(epoch uint32) (tick uint32, err error) {
	key := []byte{lastProcessedTickPerEpochKey}
	key = binary.BigEndian.AppendUint32(key, epoch)

	value, closer, err := ps.db.Get(key)
	if errors.Is(err, pebble.ErrNotFound) {
		return 0, ErrNotFound
	}

	if err != nil {
		return 0, fmt.Errorf("getting last processed tick: %v", err)
	}
	defer closer.Close()

	tick = binary.BigEndian.Uint32(value)

	return tick, nil
}

func (ps *ProcessorStore) Close() error {
	return ps.db.Close()
}
