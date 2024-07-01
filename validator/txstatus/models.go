package txstatus

import (
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
)

func qubicToProto(model types.TransactionStatus) (*protobuff.TickTransactionsStatus, error) {
	tickTransactions := make([]*protobuff.TransactionStatus, 0, model.TxCount)
	for index, txDigest := range model.TransactionDigests {
		var id types.Identity
		id, err := id.FromPubKey(txDigest, true)
		if err != nil {
			return nil, errors.Wrap(err, "converting digest to id")
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
