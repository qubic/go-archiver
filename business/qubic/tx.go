package qubic

import (
	"context"
	"encoding/hex"
	"github.com/pkg/errors"
	pb "github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
)

type TickTransactionsStatusFetcher interface {
	GetTxStatus(ctx context.Context, tick uint32) (types.TransactionStatus, error)
}

func GetTickTransactionsStatus(ctx context.Context, fetcher TickTransactionsStatusFetcher, tick uint32) (*pb.TickTransactionsStatus, error) {
	qubicTxStatus, err := fetcher.GetTxStatus(ctx, tick)
	if err != nil {
		return nil, errors.Wrap(err, "fetching tx status")
	}

	tts, err := txStatusQubicToProto(qubicTxStatus)
	if err != nil {
		return nil, errors.Wrap(err, "converting qubic tx status to proto")
	}

	return tts, nil
}

type TickTransactionsFetcher interface {
	GetTickTransactions(ctx context.Context, tickNumber uint32) (types.Transactions, error)
}

func GetTickTransactions(ctx context.Context, fetcher TickTransactionsFetcher, tick uint32) ([]*pb.Transaction, error) {
	qubicTickTxs, err := fetcher.GetTickTransactions(ctx, tick)
	if err != nil {
		return nil, errors.Wrap(err, "fetching tick transactions")
	}

	tt, err := transactionsQubicToProto(qubicTickTxs)
	if err != nil {
		return nil, errors.Wrap(err, "converting qubic tick transactions to proto")
	}

	return tt, nil
}

func txStatusQubicToProto(qubicModel types.TransactionStatus) (*pb.TickTransactionsStatus, error) {
	tickTransactions := make([]*pb.TransactionStatus, 0, qubicModel.TxCount)
	for index, txDigest := range qubicModel.TransactionDigests {
		var id types.Identity
		id, err := id.FromPubKey(txDigest, true)
		if err != nil {
			return nil, errors.Wrap(err, "converting digest to id")
		}
		moneyFlew := getMoneyFlewFromBits(qubicModel.MoneyFlew, index)
		tx := pb.TransactionStatus{
			TxId:      id.String(),
			MoneyFlew: moneyFlew,
		}

		tickTransactions = append(tickTransactions, &tx)
	}
	return &pb.TickTransactionsStatus{Transactions: tickTransactions}, nil
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

func transactionsQubicToProto(txs types.Transactions) ([]*pb.Transaction, error) {
	protoTxs := make([]*pb.Transaction, len(txs))
	for i, tx := range txs {
		txProto, err := txToProto(tx)
		if err != nil {
			return nil, errors.Wrapf(err, "converting tx to proto")
		}
		protoTxs[i] = txProto
	}

	return protoTxs, nil
}

func txToProto(tx types.Transaction) (*pb.Transaction, error) {
	digest, err := tx.Digest()
	if err != nil {
		return nil, errors.Wrap(err, "getting tx digest")
	}
	var txID types.Identity
	txID, err = txID.FromPubKey(digest, true)
	if err != nil {
		return nil, errors.Wrap(err, "getting tx id")
	}

	var sourceID types.Identity
	sourceID, err = sourceID.FromPubKey(tx.SourcePublicKey, false)
	if err != nil {
		return nil, errors.Wrap(err, "getting source id")
	}

	var destID types.Identity
	destID, err = destID.FromPubKey(tx.DestinationPublicKey, false)
	if err != nil {
		return nil, errors.Wrap(err, "getting dest id")
	}

	return &pb.Transaction{
		SourceId:     sourceID.String(),
		DestId:       destID.String(),
		Amount:       tx.Amount,
		TickNumber:   tx.Tick,
		InputType:    uint32(tx.InputType),
		InputSize:    uint32(tx.InputSize),
		InputHex:     hex.EncodeToString(tx.Input[:]),
		SignatureHex: hex.EncodeToString(tx.Signature[:]),
		TxId:         txID.String(),
	}, nil
}
