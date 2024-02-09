package tx

import (
	"context"
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-node-connector/types"
	"log"
)

var emptyTxDigest [32]byte

func Validate(ctx context.Context, transactions []types.Transaction, tickData types.TickData) error {
	digestsMap := createTxDigestsMap(tickData)
	if len(transactions) != len(digestsMap) {
		return errors.Errorf("tx count mismatch. tx count: %d, digests count: %d", len(transactions), len(digestsMap))
	}

	err := validateTransactions(ctx, transactions, digestsMap)
	if err != nil {
		return errors.Wrap(err, "validating transactions")
	}
	return nil
}

func createTxDigestsMap(tickData types.TickData) map[string]struct{}{
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

func validateTransactions(ctx context.Context, transactions []types.Transaction, digestsMap map[string]struct{}) error {
	for index, tx := range transactions {
		txDigest, err := getDigestFromTransaction(tx.Data)
		if err != nil {
			return errors.Wrap(err, "getting digest from tx data")
		}
		hexDigest := hex.EncodeToString(txDigest[:])
		if _, ok := digestsMap[hexDigest]; !ok {
			return errors.Errorf("tx not found in digests map: %s", hexDigest)
		}

		txDataBytes, err := tx.Data.MarshallBinary()
		if err != nil {
			return errors.Wrap(err, "marshalling tx data")
		}

		constructedDigest, err := utils.K12Hash(txDataBytes[:len(txDataBytes)-64])
		if err != nil {
			return errors.Wrap(err, "constructing digest from tx data")
		}

		err = utils.FourQSigVerify(ctx, tx.Data.Header.SourcePublicKey, constructedDigest, tx.Data.Signature)
		if err != nil {
			return errors.Wrap(err, "verifying tx signature")
		}

		log.Printf("Validated tx: %s. Count: %d\n", hexDigest, index)
	}

	return nil
}

func getDigestFromTransaction(data types.TransactionData) ([32]byte, error) {
	txDataMarshalledBytes, err := data.MarshallBinary()
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "marshalling")
	}

	digest, err := utils.K12Hash(txDataMarshalledBytes)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "hashing tx data")
	}

	return digest, nil
}
