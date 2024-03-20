package store

import (
	"encoding/binary"
)

const (
	TickData                     = 0x00
	QuorumData                   = 0x01
	ComputorList                 = 0x02
	Transaction                  = 0x03
	ProcessedTick                = 0x04
	LastProcessedTickPerEpoch    = 0x05
	SkippedTicksInterval         = 0x06
	IdentityTransferTransactions = 0x07
	ChainDigest                  = 0x08
)

func tickDataKey(tickNumber uint32) []byte {
	key := []byte{TickData}
	key = binary.BigEndian.AppendUint64(key, uint64(tickNumber))

	return key
}

func quorumTickDataKey(tickNumber uint32) []byte {
	key := []byte{QuorumData}
	key = binary.BigEndian.AppendUint64(key, uint64(tickNumber))

	return key
}

func computorsKey(epochNumber uint32) []byte {
	key := []byte{ComputorList}
	key = binary.BigEndian.AppendUint32(key, epochNumber)

	return key
}

func tickTxKey(txID string) ([]byte, error) {
	key := []byte{Transaction}
	key = append(key, []byte(txID)...)

	return key, nil
}

func lastProcessedTickKey() []byte {
	return []byte{ProcessedTick}
}

func lastProcessedTickKeyPerEpoch(epochNumber uint32) []byte {
	key := []byte{LastProcessedTickPerEpoch}
	key = binary.BigEndian.AppendUint32(key, epochNumber)

	return key
}

func skippedTicksIntervalKey() []byte {
	return []byte{SkippedTicksInterval}
}

func identityTransferTransactionsPerTickKey(identity string, tickNumber uint32) []byte {
	key := []byte{IdentityTransferTransactions}
	key = append(key, []byte(identity)...)
	key = binary.BigEndian.AppendUint64(key, uint64(tickNumber))

	return key
}

func identityTransferTransactions(identity string) []byte {
	key := []byte{IdentityTransferTransactions}
	key = append(key, []byte(identity)...)

	return key
}

func chainDigestKey(tickNumber uint32) []byte {
	key := []byte{ChainDigest}
	key = binary.BigEndian.AppendUint64(key, uint64(tickNumber))

	return key
}
