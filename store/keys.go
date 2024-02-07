package store

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/pkg/errors"
)

const (
	TickData     = 0x00
	QuorumData   = 0x01
	ComputorList = 0x02
	Transaction  = 0x03
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

func tickTxKey(hashHex string) ([]byte, error) {
	txHash, err := hex.DecodeString(hashHex)
	if err != nil {
		return nil, errors.Wrap(err, "hex decoding tx hash")
	}

	key := []byte{Transaction}
	key = append(key, txHash...)

	return key, nil
}
