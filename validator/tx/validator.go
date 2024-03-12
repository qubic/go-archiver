package tx

import (
	"context"
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-node-connector/types"
	"log"
)

var emptyTxDigest [32]byte

func Validate(ctx context.Context, sigVerifierFunc utils.SigVerifierFunc, transactions []types.Transaction, tickData types.TickData) ([]types.Transaction, error) {
	digestsMap := createTxDigestsMap(tickData)
	//if len(transactions) != len(digestsMap) {
	//	return nil, errors.Errorf("tx count mismatch. tx count: %d, digests count: %d", len(transactions), len(digestsMap))
	//}

	validTxs, err := validateTransactions(ctx, sigVerifierFunc, transactions, digestsMap)
	if err != nil {
		return nil, errors.Wrap(err, "validating transactions")
	}

	return validTxs, nil
}

func validateTransactions(ctx context.Context, sigVerifierFunc utils.SigVerifierFunc, transactions []types.Transaction, digestsMap map[string]struct{}) ([]types.Transaction, error) {
	validTransactions := make([]types.Transaction, 0, len(transactions))
	for _, tx := range transactions {
		txDigest, err := getDigestFromTransaction(tx)
		if err != nil {
			return nil, errors.Wrap(err, "getting digest from tx data")
		}

		hexDigest := hex.EncodeToString(txDigest[:])
		if _, ok := digestsMap[hexDigest]; !ok {
			log.Printf("tx not found in digests map: %s. not \n", hexDigest)
			continue
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

		//log.Printf("Validated tx: %s. Count: %d\n", hexDigest, index)
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

func Store(ctx context.Context, store *store.PebbleStore, tickNumber uint64, transactions types.Transactions) error {
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

func storeTransferTransactions(ctx context.Context, store *store.PebbleStore, tickNumber uint64, transactions types.Transactions) error {
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

func removeNonTransferTransactionsAndConvert(transactions []types.Transaction) (*protobuff.Transactions, error) {
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

	return &protobuff.Transactions{Transactions: transferTransactions}, nil
}

func createTransferTransactionsIdentityMap(ctx context.Context, txs *protobuff.Transactions) (map[string]*protobuff.Transactions, error) {
	txsPerIdentity := make(map[string]*protobuff.Transactions)
	for _, tx := range txs.Transactions {
		_, ok := txsPerIdentity[tx.DestId]
		if !ok {
			txsPerIdentity[tx.DestId] = &protobuff.Transactions{
				Transactions: make([]*protobuff.Transaction, 0),
			}
		}

		_, ok = txsPerIdentity[tx.SourceId]
		if !ok {
			txsPerIdentity[tx.SourceId] = &protobuff.Transactions{
				Transactions: make([]*protobuff.Transaction, 0),
			}
		}

		txsPerIdentity[tx.DestId].Transactions = append(txsPerIdentity[tx.DestId].Transactions, tx)
		txsPerIdentity[tx.SourceId].Transactions = append(txsPerIdentity[tx.SourceId].Transactions, tx)
	}

	return txsPerIdentity, nil
}
