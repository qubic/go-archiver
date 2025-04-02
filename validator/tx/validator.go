package tx

import (
	"context"
	"encoding/hex"
	"github.com/pkg/errors"
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

	if len(transactions) != len(digestsMap) {
		return nil, errors.Errorf("tx count mismatch. tx count: %d, digests count: %d", len(transactions), len(digestsMap))
	}

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

func Store(ctx context.Context, store *store.PebbleStore, tickNumber uint32, transactions types.Transactions) error {
	err := storeTickTransactions(ctx, store, transactions)
	if err != nil {
		return errors.Wrap(err, "storing tick transactions")
	}
	return nil
}

func storeTickTransactions(ctx context.Context, store *store.PebbleStore, transactions types.Transactions) error {
	protoModel, err := qubicToProto(transactions)
	if err != nil {
		return errors.Wrap(err, "converting to proto")
	}

	err = store.GetCurrentEpochStore().SetTransactions(protoModel)
	if err != nil {
		return errors.Wrap(err, "storing tick transactions")
	}

	return nil
}
