package tx

import (
	"context"
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-node-connector/types"
)

var emptyTxDigest [32]byte

func Validate(ctx context.Context, sigVerifierFunc utils.SigVerifierFunc, transactions []types.Transaction, tickData types.TickData) ([]types.Transaction, error) {
	digestsMap := createTxDigestsMap(tickData)
	// handles empty tick but with transactions
	if len(digestsMap) == 0 {
		return []types.Transaction{}, nil
	}

	validTxs, err := validateTransactions(ctx, sigVerifierFunc, transactions, digestsMap)
	if err != nil {
		return nil, errors.Wrap(err, "validating transactions")
	}

	return validTxs, nil
}

// validateTransactions validates the tick transactions against the digests map, if a transaction is not part of the
// digests map, it is considered invalid. if we have more transactions than digests, then we don't care.
// Implementation relies on the fact that for each valid transaction, the associated digest is removed
// from the digest map and at the end of the function, the map should be empty.
func validateTransactions(ctx context.Context, sigVerifierFunc utils.SigVerifierFunc, transactions []types.Transaction, digestsMap map[string]struct{}) ([]types.Transaction, error) {
	validTransactions := make([]types.Transaction, 0, len(transactions))
	for _, tx := range transactions {
		txDigest, err := getDigestFromTransaction(tx)
		if err != nil {
			return nil, errors.Wrap(err, "getting digest from tx data")
		}

		txId, err := tx.ID()
		if err != nil {
			return nil, errors.Wrap(err, "getting tx id")
		}

		hexDigest := hex.EncodeToString(txDigest[:])
		if _, ok := digestsMap[hexDigest]; !ok {
			return nil, errors.Errorf("tx id: %s not found in digests map", txId)
		}

		txDataBytes, err := tx.MarshallBinary()
		if err != nil {
			return nil, errors.Wrap(err, "marshalling tx data")
		}

		constructedDigest, err := utils.K12Hash(txDataBytes[:len(txDataBytes)-64])
		if err != nil {
			return nil, errors.Wrap(err, "constructing digest from tx data")
		}

		err = sigVerifierFunc(ctx, tx.SourcePublicKey, constructedDigest, tx.Signature)
		if err != nil {
			return nil, errors.Wrap(err, "verifying tx signature")
		}
		validTransactions = append(validTransactions, tx)
		delete(digestsMap, hexDigest)
	}

	if len(digestsMap) > 0 {
		return nil, errors.Errorf("not all digests were matched, remaining: %d", len(digestsMap))
	}

	return validTransactions, nil
}

func getDigestFromTransaction(tx types.Transaction) ([32]byte, error) {
	txDataMarshalledBytes, err := tx.MarshallBinary()
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "marshalling")
	}

	digest, err := utils.K12Hash(txDataMarshalledBytes)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "hashing tx tx")
	}

	return digest, nil
}

func createTxDigestsMap(tickData types.TickData) map[string]struct{} {
	digestsMap := make(map[string]struct{})

	for _, digest := range tickData.TransactionDigests {
		if digest == emptyTxDigest {
			continue
		}

		hexDigest := hex.EncodeToString(digest[:])
		digestsMap[hexDigest] = struct{}{}
	}

	return digestsMap
}

func Store(ctx context.Context, store *store.PebbleStore, tickNumber uint32, transactions types.Transactions) error {
	err := storeTickTransactions(ctx, store, transactions)
	if err != nil {
		return errors.Wrap(err, "storing tick transactions")
	}

	err = storeTransferTransactions(ctx, store, tickNumber, transactions)
	if err != nil {
		return errors.Wrap(err, "storing transfer transactions")
	}

	return nil
}

func storeTickTransactions(ctx context.Context, store *store.PebbleStore, transactions types.Transactions) error {
	protoModel, err := qubicToProto(transactions)
	if err != nil {
		return errors.Wrap(err, "converting to proto")
	}

	err = store.SetTransactions(ctx, protoModel)
	if err != nil {
		return errors.Wrap(err, "storing tick transactions")
	}

	return nil
}

func storeTransferTransactions(ctx context.Context, store *store.PebbleStore, tickNumber uint32, transactions types.Transactions) error {
	transferTransactions, err := removeNonTransferTransactionsAndConvert(transactions)
	if err != nil {
		return errors.Wrap(err, "removing non transfer transactions")
	}
	txsPerIdentity, err := createTransferTransactionsIdentityMap(ctx, transferTransactions)
	if err != nil {
		return errors.Wrap(err, "filtering transfer transactions")
	}

	for id, txs := range txsPerIdentity {
		err = store.PutTransferTransactionsPerTick(ctx, id, tickNumber, &protobuff.TransferTransactionsPerTick{TickNumber: uint32(tickNumber), Identity: id, Transactions: txs})
		if err != nil {
			return errors.Wrap(err, "storing transfer transactions")
		}
	}

	return nil
}

func removeNonTransferTransactionsAndConvert(transactions []types.Transaction) ([]*protobuff.Transaction, error) {
	transferTransactions := make([]*protobuff.Transaction, 0)
	for _, tx := range transactions {
		if tx.Amount == 0 {
			continue
		}

		protoTx, err := txToProto(tx)
		if err != nil {
			return nil, errors.Wrap(err, "converting to proto")
		}

		transferTransactions = append(transferTransactions, protoTx)
	}

	return transferTransactions, nil
}

func createTransferTransactionsIdentityMap(ctx context.Context, txs []*protobuff.Transaction) (map[string][]*protobuff.Transaction, error) {
	txsPerIdentity := make(map[string][]*protobuff.Transaction)
	for _, tx := range txs {
		_, ok := txsPerIdentity[tx.DestId]
		if !ok {
			txsPerIdentity[tx.DestId] = make([]*protobuff.Transaction, 0)
		}

		_, ok = txsPerIdentity[tx.SourceId]
		if !ok {
			txsPerIdentity[tx.SourceId] = make([]*protobuff.Transaction, 0)
		}

		txsPerIdentity[tx.DestId] = append(txsPerIdentity[tx.DestId], tx)
		txsPerIdentity[tx.SourceId] = append(txsPerIdentity[tx.SourceId], tx)
	}

	return txsPerIdentity, nil
}
