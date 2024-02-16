package store

import (
	"encoding/binary"
)

const (
	TickData          = 0x00
	QuorumData        = 0x01
	ComputorList      = 0x02
	Transaction       = 0x03
	LastProcessedTick = 0x04
)

func tickDataKey(tickNumber uint64) []byte {
	key := []byte{TickData}
	key = binary.BigEndian.AppendUint64(key, tickNumber)

	return key
}

func quorumTickDataKey(tickNumber uint64) []byte {
	key := []byte{QuorumData}
	key = binary.BigEndian.AppendUint64(key, tickNumber)

	return key
}

func computorsKey(epochNumber uint64) []byte {
	key := []byte{ComputorList}
	key = binary.BigEndian.AppendUint64(key, epochNumber)

	return key
}

func tickTxKey(digest []byte) ([]byte, error) {
	key := []byte{Transaction}
	key = append(key, digest...)

	return key, nil
}

func lastProcessedTickKey() []byte {
	return []byte{LastProcessedTick}

}
