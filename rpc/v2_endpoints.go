package rpc

import (
	"cmp"
	"context"
	"encoding/hex"
	"slices"

	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-node-connector/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
)

func (s *Server) GetLatestTickV2(ctx context.Context, req *emptypb.Empty) (*protobuff.GetLatestTickResponse, error) {
	return s.GetLatestTick(ctx, req)
}

func (s *Server) GetTickV2(ctx context.Context, req *protobuff.GetTickRequestV2) (*protobuff.GetTickResponseV2, error) {

	//TODO: Implement

	return nil, status.Errorf(codes.Unimplemented, "Not yet implemented")
}

func (s *Server) GetTickQuorumDataV2(ctx context.Context, req *protobuff.GetTickRequestV2) (*protobuff.GetQuorumTickDataResponse, error) {
	return s.GetQuorumTickData(ctx, &protobuff.GetQuorumTickDataRequest{
		TickNumber: req.TickNumber,
	})
}

func (s *Server) GetTickChainHashV2(ctx context.Context, req *protobuff.GetTickRequestV2) (*protobuff.GetChainHashResponse, error) {
	return s.GetChainHash(ctx, &protobuff.GetChainHashRequest{
		TickNumber: req.TickNumber,
	})
}

func (s *Server) GetTickStoreHashV2(ctx context.Context, req *protobuff.GetTickRequestV2) (*protobuff.GetChainHashResponse, error) {
	return s.GetStoreHash(ctx, &protobuff.GetChainHashRequest{
		TickNumber: req.TickNumber,
	})
}

func (s *Server) GetTickTransactionsV2(ctx context.Context, req *protobuff.GetTickTransactionsRequestV2) (*protobuff.GetTickTransactionsResponseV2, error) {

	tickRequest := protobuff.GetTickRequestV2{
		TickNumber: req.TickNumber,
	}

	if req.Approved {
		return s.GetApprovedTickTransactionsV2(ctx, &tickRequest)
	}

	if req.Transfers {
		return s.GetTransferTickTransactionsV2(ctx, &tickRequest)
	}

	return s.GetAllTickTransactionsV2(ctx, &tickRequest)

}

func (s *Server) GetAllTickTransactionsV2(ctx context.Context, req *protobuff.GetTickRequestV2) (*protobuff.GetTickTransactionsResponseV2, error) {
	lastProcessedTick, err := s.store.GetLastProcessedTick(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting last processed tick: %v", err)
	}
	if req.TickNumber > lastProcessedTick.TickNumber {
		st := status.Newf(codes.FailedPrecondition, "requested tick number %d is greater than last processed tick %d", req.TickNumber, lastProcessedTick.TickNumber)
		st, err = st.WithDetails(&protobuff.LastProcessedTick{LastProcessedTick: lastProcessedTick.TickNumber})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "creating custom status")
		}

		return nil, st.Err()
	}

	processedTickIntervalsPerEpoch, err := s.store.GetProcessedTickIntervals(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting processed tick intervals per epoch")
	}

	wasSkipped, nextAvailableTick := wasTickSkippedByArchive(req.TickNumber, processedTickIntervalsPerEpoch)
	if wasSkipped == true {
		st := status.Newf(codes.OutOfRange, "provided tick number %d was skipped by the system, next available tick is %d", req.TickNumber, nextAvailableTick)
		st, err = st.WithDetails(&protobuff.NextAvailableTick{NextTickNumber: nextAvailableTick})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "creating custom status")
		}

		return nil, st.Err()
	}

	txs, err := s.store.GetTickTransactions(ctx, req.TickNumber)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "tick transactions for specified tick not found")
		}
		return nil, status.Errorf(codes.Internal, "getting tick transactions: %v", err)
	}

	var transactions []*protobuff.TransactionData

	for _, transaction := range txs {

		transactionInfo, err := getTransactionInfo(ctx, s.store, transaction.TxId, transaction.TickNumber)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get transaction info: %v", err)
		}

		transactions = append(transactions, &protobuff.TransactionData{
			Transaction: transaction,
			Timestamp:   transactionInfo.timestamp,
			MoneyFlew:   transactionInfo.moneyFlew,
		})

	}

	return &protobuff.GetTickTransactionsResponseV2{Transactions: transactions}, nil
}

func (s *Server) GetTransferTickTransactionsV2(ctx context.Context, req *protobuff.GetTickRequestV2) (*protobuff.GetTickTransactionsResponseV2, error) {
	lastProcessedTick, err := s.store.GetLastProcessedTick(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting last processed tick: %v", err)
	}
	if req.TickNumber > lastProcessedTick.TickNumber {
		st := status.Newf(codes.FailedPrecondition, "requested tick number %d is greater than last processed tick %d", req.TickNumber, lastProcessedTick.TickNumber)
		st, err = st.WithDetails(&protobuff.LastProcessedTick{LastProcessedTick: lastProcessedTick.TickNumber})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "creating custom status")
		}

		return nil, st.Err()
	}

	processedTickIntervalsPerEpoch, err := s.store.GetProcessedTickIntervals(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting processed tick intervals per epoch")
	}

	wasSkipped, nextAvailableTick := wasTickSkippedByArchive(req.TickNumber, processedTickIntervalsPerEpoch)
	if wasSkipped == true {
		st := status.Newf(codes.OutOfRange, "provided tick number %d was skipped by the system, next available tick is %d", req.TickNumber, nextAvailableTick)
		st, err = st.WithDetails(&protobuff.NextAvailableTick{NextTickNumber: nextAvailableTick})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "creating custom status")
		}

		return nil, st.Err()
	}

	txs, err := s.store.GetTickTransferTransactions(ctx, req.TickNumber)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "tick transfer transactions for specified tick not found")
		}
		return nil, status.Errorf(codes.Internal, "getting tick transactions: %v", err)
	}

	var transactions []*protobuff.TransactionData

	for _, transaction := range txs {

		transactionInfo, err := getTransactionInfo(ctx, s.store, transaction.TxId, transaction.TickNumber)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get transaction info: %v", err)
		}

		transactions = append(transactions, &protobuff.TransactionData{
			Transaction: transaction,
			Timestamp:   transactionInfo.timestamp,
			MoneyFlew:   transactionInfo.moneyFlew,
		})

	}

	return &protobuff.GetTickTransactionsResponseV2{Transactions: transactions}, nil
}

func (s *Server) GetApprovedTickTransactionsV2(ctx context.Context, req *protobuff.GetTickRequestV2) (*protobuff.GetTickTransactionsResponseV2, error) {
	lastProcessedTick, err := s.store.GetLastProcessedTick(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting last processed tick: %v", err)
	}
	if req.TickNumber > lastProcessedTick.TickNumber {
		st := status.Newf(codes.FailedPrecondition, "requested tick number %d is greater than last processed tick %d", req.TickNumber, lastProcessedTick.TickNumber)
		st, err = st.WithDetails(&protobuff.LastProcessedTick{LastProcessedTick: lastProcessedTick.TickNumber})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "creating custom status")
		}

		return nil, st.Err()
	}

	processedTickIntervalsPerEpoch, err := s.store.GetProcessedTickIntervals(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting processed tick intervals per epoch")
	}

	wasSkipped, nextAvailableTick := wasTickSkippedByArchive(req.TickNumber, processedTickIntervalsPerEpoch)
	if wasSkipped == true {
		st := status.Newf(codes.OutOfRange, "provided tick number %d was skipped by the system, next available tick is %d", req.TickNumber, nextAvailableTick)
		st, err = st.WithDetails(&protobuff.NextAvailableTick{NextTickNumber: nextAvailableTick})
		if err != nil {
			return nil, status.Errorf(codes.Internal, "creating custom status")
		}

		return nil, st.Err()
	}

	txs, err := s.store.GetTickTransferTransactions(ctx, req.TickNumber)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "tick transfer transactions for specified tick not found")
		}
		return nil, status.Errorf(codes.Internal, "getting tick transactions: %v", err)
	}

	var transactions []*protobuff.TransactionData

	for _, transaction := range txs {

		transactionInfo, err := getTransactionInfo(ctx, s.store, transaction.TxId, transaction.TickNumber)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to get transaction info: %v", err)
		}

		if transactionInfo.moneyFlew == false {
			continue
		}

		transactions = append(transactions, &protobuff.TransactionData{
			Transaction: transaction,
			Timestamp:   transactionInfo.timestamp,
			MoneyFlew:   transactionInfo.moneyFlew,
		})

	}

	return &protobuff.GetTickTransactionsResponseV2{Transactions: transactions}, nil
}

func (s *Server) GetTransactionV2(ctx context.Context, req *protobuff.GetTransactionRequestV2) (*protobuff.GetTransactionResponseV2, error) {

	tx, err := s.store.GetTransaction(ctx, req.TxId)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "transaction not found")
		}
		return nil, status.Errorf(codes.Internal, "getting transaction: %v", err)
	}

	transactionInfo, err := getTransactionInfo(ctx, s.store, tx.TxId, tx.TickNumber)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting transaction info")
	}

	res := protobuff.GetTransactionResponseV2{
		Transaction: tx,
		MoneyFlew:   transactionInfo.moneyFlew,
		Timestamp:   transactionInfo.timestamp,
	}
	return &res, nil

}

func (s *Server) GetSendManyTransactionV2(ctx context.Context, req *protobuff.GetSendManyTransactionRequestV2) (*protobuff.GetSendManyTransactionResponseV2, error) {
	transaction, err := s.store.GetTransaction(ctx, req.TxId)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "transaction not found")
		}
		return nil, status.Errorf(codes.Internal, "getting transaction: %v", err)
	}

	if transaction.InputType != 1 || transaction.DestId != types.QutilAddress {
		return nil, status.Errorf(codes.NotFound, "request transaction is not of send-many type")
	}

	transactionInfo, err := getTransactionInfo(ctx, s.store, transaction.TxId, transaction.TickNumber)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting transaction info")
	}

	rawPayload, err := hex.DecodeString(transaction.InputHex)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to decode raw payload")
	}

	var sendManyPayload types.SendManyTransferPayload
	err = sendManyPayload.UnmarshallBinary(rawPayload)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to unmarshall payload data")
	}

	sendManyTransfers := make([]*protobuff.SendManyTransfer, 0)

	transfers, err := sendManyPayload.GetTransfers()

	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting send many transfers")
	}

	for _, transfer := range transfers {
		sendManyTransfers = append(sendManyTransfers, &protobuff.SendManyTransfer{
			DestId: transfer.AddressID.String(),
			Amount: transfer.Amount,
		})
	}

	sendManyTransaction := protobuff.SendManyTransaction{
		SourceId:     transaction.SourceId,
		Transfers:    sendManyTransfers,
		TotalAmount:  sendManyPayload.GetTotalAmount(),
		TickNumber:   transaction.TickNumber,
		TxId:         transaction.TxId,
		SignatureHex: transaction.SignatureHex,
	}

	return &protobuff.GetSendManyTransactionResponseV2{
		Transaction: &sendManyTransaction,
		Timestamp:   transactionInfo.timestamp,
		MoneyFlew:   transactionInfo.moneyFlew,
	}, nil
}

func (s *Server) GetIdentityTransfersInTickRangeV2(ctx context.Context, req *protobuff.GetTransferTransactionsPerTickRequestV2) (*protobuff.GetIdentityTransfersInTickRangeResponseV2, error) {
	txs, err := s.store.GetTransferTransactions(ctx, req.Identity, uint64(req.GetStartTick()), uint64(req.GetEndTick()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting transfer transactions: %v", err)
	}

	var totalTransactions []*protobuff.PerTickIdentityTransfers

	for _, transactionsPerTick := range txs {

		var tickTransactions []*protobuff.TransactionData

		for _, transaction := range transactionsPerTick.Transactions {
			transactionInfo, err := getTransactionInfo(ctx, s.store, transaction.TxId, transaction.TickNumber)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "getting transaction info: %v", err)
			}

			if req.ScOnly == true && transaction.GetInputType() == 0 {
				continue
			}

			transactionData := &protobuff.TransactionData{
				Transaction: transaction,
				Timestamp:   transactionInfo.timestamp,
				MoneyFlew:   transactionInfo.moneyFlew,
			}

			tickTransactions = append(tickTransactions, transactionData)
		}

		transfers := &protobuff.PerTickIdentityTransfers{
			Transactions: tickTransactions,
			Identity:     transactionsPerTick.Identity,
			TickNumber:   transactionsPerTick.TickNumber,
		}

		totalTransactions = append(totalTransactions, transfers)
	}

	if req.Desc == true {

		slices.SortFunc(totalTransactions, func(a, b *protobuff.PerTickIdentityTransfers) int {
			return -cmp.Compare(a.TickNumber, b.TickNumber)
		})

	}

    if req.Count > 0 && len(totalTransactions) > int(req.Count) {
        totalTransactions = totalTransactions[:req.Count]
    }

	return &protobuff.GetIdentityTransfersInTickRangeResponseV2{
		Transactions: totalTransactions,
	}, nil

}
