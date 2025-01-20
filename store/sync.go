package store

import (
	"encoding/binary"
	"github.com/cockroachdb/pebble"
	"github.com/pingcap/errors"
	"github.com/qubic/go-archiver/protobuff"
	"google.golang.org/protobuf/proto"
)

func AddQuorumTickDataStoredToBatch(batch *pebble.Batch, quorumDataStored *protobuff.QuorumTickDataStored, tickNumber uint32) error {

	key := AssembleKey(QuorumData, tickNumber)

	serialized, err := proto.Marshal(quorumDataStored)
	if err != nil {
		return errors.Wrap(err, "serializing quorum data stored")
	}

	err = batch.Set(key, serialized, nil)
	if err != nil {
		return errors.Wrap(err, "adding quorum data stored to batch")
	}
	return nil
}

func AddTickDataToBatch(batch *pebble.Batch, tickData *protobuff.TickData, tickNumber uint32) error {

	key := AssembleKey(TickData, tickNumber)

	serialized, err := proto.Marshal(tickData)
	if err != nil {
		return errors.Wrap(err, "serializing quorum data")
	}

	err = batch.Set(key, serialized, nil)
	if err != nil {
		return errors.Wrap(err, "adding tick data to batch")
	}
	return nil
}

func AddTransactionsToBatch(batch *pebble.Batch, transactions []*protobuff.Transaction) error {

	for _, transaction := range transactions {
		key := AssembleKey(Transaction, transaction.TxId)
		serialized, err := proto.Marshal(transaction)
		if err != nil {
			return errors.Wrap(err, "serializing transaction")
		}
		err = batch.Set(key, serialized, nil)
		if err != nil {
			return errors.Wrap(err, "adding transaction to batch")
		}
	}
	return nil
}

func AddTransactionsPerIdentityToBatch(batch *pebble.Batch, transactions []*protobuff.Transaction, tickNumber uint32) error {

	transactionsPerIdentity := removeNonTransferTransactionsAndSortPerIdentity(transactions)

	for identity, txs := range transactionsPerIdentity {

		key := AssembleKey(IdentityTransferTransactions, identity)
		key = binary.BigEndian.AppendUint64(key, uint64(tickNumber))

		serialized, err := proto.Marshal(&protobuff.TransferTransactionsPerTick{
			TickNumber:   tickNumber,
			Identity:     identity,
			Transactions: txs,
		})
		if err != nil {
			return errors.Wrap(err, "serializing transfer transactions per tick")
		}

		err = batch.Set(key, serialized, nil)
		if err != nil {
			return errors.Wrap(err, "adding transfer transactions per tick to batch")
		}
	}

	return nil
}

func AddApprovedTransactionsToBatch(batch *pebble.Batch, approvedTransactions *protobuff.TickTransactionsStatus, tickNumber uint32) error {
	key := AssembleKey(TickTransactionsStatus, uint64(tickNumber))

	serialized, err := proto.Marshal(approvedTransactions)
	if err != nil {
		return errors.Wrap(err, "serializing approved transactions")
	}

	err = batch.Set(key, serialized, nil)
	if err != nil {
		return errors.Wrap(err, "adding approved transactions to batch")
	}

	return nil
}

func AddTransactionsStatusToBatch(batch *pebble.Batch, approvedTransactions *protobuff.TickTransactionsStatus) error {

	for _, transaction := range approvedTransactions.Transactions {
		key := AssembleKey(TransactionStatus, transaction.TxId)
		serialized, err := proto.Marshal(transaction)
		if err != nil {
			return errors.Wrap(err, "serializing transaction status")
		}

		err = batch.Set(key, serialized, nil)
		if err != nil {
			return errors.Wrap(err, "adding transaction status to batch")
		}
	}

	return nil
}

func AddChainDigestToBatch(batch *pebble.Batch, chainDigest [32]byte, tickNumber uint32) error {

	key := AssembleKey(ChainDigest, tickNumber)
	err := batch.Set(key, chainDigest[:], nil)
	if err != nil {
		return errors.Wrap(err, "adding chain digest to batch")
	}

	return nil
}

func AddStoreDigestToBatch(batch *pebble.Batch, storeDigest [32]byte, tickNumber uint32) error {

	key := AssembleKey(StoreDigest, tickNumber)
	err := batch.Set(key, storeDigest[:], nil)
	if err != nil {
		return errors.Wrap(err, "adding store digest to batch")
	}

	return nil
}

func AddLastProcessedTickPerEpochToBatch(batch *pebble.Batch, epoch, tickNumber uint32) error {

	key := AssembleKey(LastProcessedTickPerEpoch, epoch)
	value := make([]byte, 4)
	binary.LittleEndian.PutUint32(value, tickNumber)
	err := batch.Set(key, value, nil)
	if err != nil {
		return errors.Wrap(err, "adding last processed tick per epoch to batch")
	}

	return nil
}

func AddLastProcessedTickToBatch(batch *pebble.Batch, epoch, tickNumber uint32) error {

	lastProcessedTickProto := protobuff.ProcessedTick{
		TickNumber: tickNumber,
		Epoch:      epoch,
	}

	key := []byte{LastProcessedTick}
	serialized, err := proto.Marshal(&lastProcessedTickProto)
	if err != nil {
		return errors.Wrap(err, "serializing last processed tick")
	}
	err = batch.Set(key, serialized, nil)
	if err != nil {
		return errors.Wrap(err, "adding last processed tick to batch")
	}
	return nil
}

func AddProcessedTickIntervalsPerEpochToBatch(batch *pebble.Batch, processedTickIntervalsPerEpoch *protobuff.ProcessedTickIntervalsPerEpoch, epoch uint32) error {
	key := AssembleKey(ProcessedTickIntervals, epoch)

	serialized, err := proto.Marshal(processedTickIntervalsPerEpoch)
	if err != nil {
		return errors.Wrap(err, "serializing processed tick intervals per epoch")
	}
	err = batch.Set(key, serialized, nil)
	if err != nil {
		return errors.Wrap(err, "adding processed tick intervals to batch")
	}
	return nil
}

func AddLastSynchronizedTickToBatch(batch *pebble.Batch, lastSynchronizedTick *protobuff.SyncLastSynchronizedTick) error {
	key := []byte{SyncLastSynchronizedTick}

	serialized, err := proto.Marshal(lastSynchronizedTick)
	if err != nil {
		return errors.Wrap(err, "serializing last synchronized tick")
	}
	err = batch.Set(key, serialized, nil)
	if err != nil {
		return errors.Wrap(err, "adding last synchronized tick to batch")
	}

	return nil
}

func removeNonTransferTransactionsAndSortPerIdentity(transactions []*protobuff.Transaction) map[string][]*protobuff.Transaction {

	transferTransactions := make([]*protobuff.Transaction, 0)
	for _, transaction := range transactions {
		if transaction.Amount == 0 {
			continue
		}
		transferTransactions = append(transferTransactions, transaction)
	}
	transactionsPerIdentity := make(map[string][]*protobuff.Transaction)
	for _, transaction := range transferTransactions {
		_, exists := transactionsPerIdentity[transaction.DestId]
		if !exists {
			transactionsPerIdentity[transaction.DestId] = make([]*protobuff.Transaction, 0)
		}
		_, exists = transactionsPerIdentity[transaction.SourceId]
		if !exists {
			transactionsPerIdentity[transaction.SourceId] = make([]*protobuff.Transaction, 0)
		}
	}

	return transactionsPerIdentity

}
