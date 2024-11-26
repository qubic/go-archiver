package store

import (
	"encoding/binary"
)

const (
	TickData                     = 0x00
	QuorumData                   = 0x01
	ComputorList                 = 0x02
	Transaction                  = 0x03
	LastProcessedTick            = 0x04
	LastProcessedTickPerEpoch    = 0x05
	SkippedTicksInterval         = 0x06
	IdentityTransferTransactions = 0x07
	ChainDigest                  = 0x08
	ProcessedTickIntervals       = 0x09
	TickTransactionsStatus       = 0x10
	TransactionStatus            = 0x11
	StoreDigest                  = 0x12
	EmptyTicksPerEpoch           = 0x13
	LastTickQuorumDataPerEpoch   = 0x14
	EmptyTickListPerEpoch        = 0x15
	SyncLastSynchronizedTick     = 0x20
)

func emptyTickListPerEpochKey(epoch uint32) []byte {
	key := []byte{EmptyTickListPerEpoch}
	key = binary.BigEndian.AppendUint64(key, uint64(epoch))
	return key
}

func lastTickQuorumDataPerEpochKey(epoch uint32) []byte {
	key := []byte{LastTickQuorumDataPerEpoch}
	key = binary.BigEndian.AppendUint64(key, uint64(epoch))
	return key
}

func emptyTicksPerEpochKey(epoch uint32) []byte {
	key := []byte{EmptyTicksPerEpoch}
	key = binary.BigEndian.AppendUint64(key, uint64(epoch))
	return key
}

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
	return []byte{LastProcessedTick}
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

func storeDigestKey(tickNumber uint32) []byte {
	key := []byte{StoreDigest}
	key = binary.BigEndian.AppendUint64(key, uint64(tickNumber))

	return key
}

func processedTickIntervalsPerEpochKey(epoch uint32) []byte {
	key := []byte{ProcessedTickIntervals}
	key = binary.BigEndian.AppendUint32(key, epoch)

	return key
}

func processedTickIntervalsKey() []byte {
	key := []byte{ProcessedTickIntervals}

	return key
}

func txStatusKey(txID string) []byte {
	key := []byte{TransactionStatus}
	key = append(key, []byte(txID)...)

	return key
}

func tickTxStatusKey(tickNumber uint64) []byte {
	key := []byte{TickTransactionsStatus}
	key = binary.BigEndian.AppendUint64(key, tickNumber)

	return key
}

func syncLastSynchronizedTick() []byte {
	return []byte{SyncLastSynchronizedTick}
}

type IDType interface {
	uint32 | uint64 | string
}

func AssembleKey[T IDType](keyPrefix int, id T) []byte {

	prefix := byte(keyPrefix)

	key := []byte{prefix}

	switch any(id).(type) {

	case uint32:
		asserted := any(id).(uint32)

		if keyPrefix == LastProcessedTickPerEpoch || keyPrefix == ProcessedTickIntervals {
			key = binary.BigEndian.AppendUint32(key, asserted)
			break
		}

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
