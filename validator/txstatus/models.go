package txstatus

import (
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
	"log"
)

func qubicToProto(txs types.Transactions, model types.TransactionStatus) (*protobuff.TickTransactionsStatus, error) {
	tickTransactions := make([]*protobuff.TransactionStatus, 0, model.TxCount)
	txsIdMap, err := createTxsIdMap(txs)
	if err != nil {
		return nil, errors.Wrap(err, "error creating txs id map")
	}

	for index, txDigest := range model.TransactionDigests {
		var id types.Identity
		id, err := id.FromPubKey(txDigest, true)
		if err != nil {
			return nil, errors.Wrap(err, "converting digest to id")
		}
		if _, ok := txsIdMap[id.String()]; !ok {
			log.Printf("Skipping tx status with id: %s\n", id.String())
			continue
		}

		moneyFlew := getMoneyFlewFromBits(model.MoneyFlew, index)

		tx := &protobuff.TransactionStatus{
			TxId:      id.String(),
			MoneyFlew: moneyFlew,
		}

		tickTransactions = append(tickTransactions, tx)
	}

	return &protobuff.TickTransactionsStatus{Transactions: tickTransactions}, nil
}

func createTxsIdMap(txs types.Transactions) (map[string]struct{}, error) {
	txsIdMap := make(map[string]struct{}, len(txs))
	for _, tx := range txs {
		digest, err := tx.Digest()
		if err != nil {
			return nil, errors.Wrapf(err, "creating tx digest for tx with src pubkey %s", tx.SourcePublicKey)
		}

		id, err := tx.ID()
		if err != nil {
			return nil, errors.Wrapf(err, "converting tx with digest %s to id", digest)
		}

		txsIdMap[id] = struct{}{}
	}

	return txsIdMap, nil
}

func getMoneyFlewFromBits(input [(types.NumberOfTransactionsPerTick + 7) / 8]byte, digestIndex int) bool {
	pos := digestIndex / 8
	bitIndex := digestIndex % 8

	return getNthBit(input[pos], bitIndex)
}

func getNthBit(input byte, bitIndex int) bool {
	// Shift the input byte to the right by the bitIndex positions
	// This isolates the bit at the bitIndex position at the least significant bit position
	shifted := input >> bitIndex

	// Extract the least significant bit using a bitwise AND operation with 1
	// If the least significant bit is 1, the result will be 1; otherwise, it will be 0
	return shifted&1 == 1
}
