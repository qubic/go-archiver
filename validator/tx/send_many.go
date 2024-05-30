package tx

import (
	"context"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-node-connector/types"
)

func StoreSendMany(ctx context.Context, store *store.PebbleStore, tickNumber uint32, transactions types.Transactions, transactionStatus *protobuff.TickTransactionsStatus) error {

	transactionMap, err := filterAndMapTransactions(transactions)
	if err != nil {
		return errors.Wrap(err, "creating send many transaction map")
	}

	transactionsToSave := make(map[string]*protobuff.SendManyTransaction)

	for _, status := range transactionStatus.Transactions {

		transaction := transactionMap[status.TxId]

		var sourceIdentity types.Identity
		sourceIdentity, err := sourceIdentity.FromPubKey(transaction.SourcePublicKey, false)
		if err != nil {
			return errors.Wrap(err, "getting source public key")
		}

		var sendManyPayload types.SendManyTransferPayload
		err = sendManyPayload.UnmarshallBinary(transaction.Input)
		if err != nil {
			return errors.Wrap(err, "unmarshalling send many payload")
		}

		transfers := make([]*protobuff.SendManyTransfer, 0)

		for _, transfer := range transfers {
			transfers = append(transfers, &protobuff.SendManyTransfer{
				DestId: transfer.DestId,
				Amount: transfer.Amount,
			})
		}

		sendManyTransaction := protobuff.SendManyTransaction{
			SourceId:    sourceIdentity.String(),
			Transfers:   transfers,
			TotalAmount: sendManyPayload.GetTotalAmount(),
			Status:      status,
			Tick:        tickNumber,
		}

		transactionsToSave[status.TxId] = &sendManyTransaction
	}

	err = store.SetSendManyTransactions(ctx, transactionsToSave)
	if err != nil {
		return errors.Wrap(err, "storing send many transactions")
	}

	return nil
}

func filterAndMapTransactions(transactions types.Transactions) (map[string]types.Transaction, error) {

	qutilIdentity := types.Identity(types.QutilAddress)

	qutilPublicKey, err := qutilIdentity.ToPubKey(false)
	if err != nil {
		return nil, errors.Wrap(err, "obtaining qutil public key")
	}

	transactionMap := make(map[string]types.Transaction)

	for _, transaction := range transactions {

		if transaction.InputType != 1 || transaction.DestinationPublicKey != qutilPublicKey {
			continue
		}

		transactionId, err := transaction.ID()
		if err != nil {
			return nil, errors.Wrap(err, "getting transaction id")
		}

		transactionMap[transactionId] = transaction

	}

	return transactionMap, nil
}
