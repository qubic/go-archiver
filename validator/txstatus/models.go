package txstatus

import (
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
)

func qubicToProto(model []types.IndividualTransactionStatus) *protobuff.TickTransactionsStatus {
	tickTransactions := make([]*protobuff.TransactionStatus, 0, len(model))
	for _, txStatus := range model {
		tx := &protobuff.TransactionStatus{
			TxId:      txStatus.TxID,
			MoneyFlew: txStatus.MoneyFlew,
		}

		tickTransactions = append(tickTransactions, tx)
	}

	return &protobuff.TickTransactionsStatus{Transactions: tickTransactions}
}
