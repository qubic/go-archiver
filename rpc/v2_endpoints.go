package rpc

import (
	"cmp"
	"context"
	"encoding/hex"
	"github.com/qubic/go-archiver/utils"
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

	return &protobuff.GetIdentityTransfersInTickRangeResponseV2{
		Transactions: totalTransactions,
	}, nil

}

func (s *Server) GetQuorumTickDataV2(ctx context.Context, req *protobuff.GetQuorumTickDataRequest) (*protobuff.GetQuorumTickDataResponse, error) {

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

	if req.TickNumber == lastProcessedTick.TickNumber {

		tickData, err := s.store.GetQuorumTickDataV2(ctx, req.TickNumber)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, status.Errorf(codes.NotFound, "quorum tick data not found")
			}
			return nil, status.Errorf(codes.Internal, "getting quorum tick data: %v", err)
		}

		res := protobuff.GetQuorumTickDataResponse{
			QuorumTickData: &protobuff.QuorumTickData{
				QuorumTickStructure:   tickData.QuorumTickStructure,
				QuorumDiffPerComputor: make(map[uint32]*protobuff.QuorumDiff),
			},
		}

		for id, diff := range tickData.QuorumDiffPerComputor {
			res.QuorumTickData.QuorumDiffPerComputor[id] = &protobuff.QuorumDiff{
				ExpectedNextTickTxDigestHex: diff.ExpectedNextTickTxDigestHex,
				SignatureHex:                diff.SignatureHex,
			}
		}

		return &res, nil
	}

	nextTick := req.TickNumber + 1

	//Check if next tick was skipped
	//TODO: check if this is accurate behaviour (if the tick gets skipped, then the data should be in the next available tick)
	skipped, nextAvailable := wasTickSkippedByArchive(nextAvailableTick, processedTickIntervalsPerEpoch)
	if skipped {
		nextTick = nextAvailable
	}

	//Get quorum data for next tick
	nextTickQuorumData, err := s.store.GetQuorumTickDataV2(ctx, nextTick)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.Internal, "quorum data for next tick was not found")
		}
		return nil, status.Errorf(codes.Internal, "getting tick data: %v", err)
	}

	//Get quorum data for current tick
	currentTickQuorumData, err := s.store.GetQuorumTickDataV2(ctx, req.TickNumber)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.Internal, "quorum data for  tick was not found")
		}
		return nil, status.Errorf(codes.Internal, "getting tick data: %v", err)
	}

	//Get computors
	computors, err := s.store.GetComputors(ctx, currentTickQuorumData.QuorumTickStructure.Epoch)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting computor list")
	}

	//Response object
	res := protobuff.GetQuorumTickDataResponse{
		QuorumTickData: &protobuff.QuorumTickData{
			QuorumTickStructure:   currentTickQuorumData.QuorumTickStructure,
			QuorumDiffPerComputor: make(map[uint32]*protobuff.QuorumDiff),
		},
	}

	//Digests
	spectrumDigest, err := hex.DecodeString(nextTickQuorumData.QuorumTickStructure.PrevSpectrumDigestHex)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "obtaining spectrum digest from next tick quorum data")
	}
	universeDigest, err := hex.DecodeString(nextTickQuorumData.QuorumTickStructure.PrevUniverseDigestHex)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "obtaining universe digest from next tick quorum data")
	}
	computerDigest, err := hex.DecodeString(nextTickQuorumData.QuorumTickStructure.PrevComputerDigestHex)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "obtaining computer digest from next tick quorum data")
	}
	resourceDigest, err := hex.DecodeString(nextTickQuorumData.QuorumTickStructure.PrevResourceTestingDigestHex)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "obtaining resource testing digest from next tick quorum data")
	}

	//Loop over all computors in current tick data
	for id, voteDiff := range currentTickQuorumData.QuorumDiffPerComputor {

		identity := types.Identity(computors.Identities[id])

		computorPublicKey, err := identity.ToPubKey(false)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "obtaining public key for computor id: %d", id)
		}

		var tmp [64]byte
		copy(computorPublicKey[:], tmp[:32]) // Public key as the first part

		//Salted spectrum digest
		copy(spectrumDigest[:], tmp[32:])
		saltedSpectrumDigest, err := utils.K12Hash(tmp[:])
		if err != nil {
			return nil, status.Errorf(codes.Internal, "hashing salted spectrum digest")
		}

		//Salted universe digest
		copy(universeDigest[:], tmp[32:])
		saltedUniverseDigest, err := utils.K12Hash(tmp[:])
		if err != nil {
			return nil, status.Errorf(codes.Internal, "hashing salted universe digest")
		}

		//Salted computer digest
		copy(computerDigest[:], tmp[32:])
		saltedComputerDigest, err := utils.K12Hash(tmp[:])
		if err != nil {
			return nil, status.Errorf(codes.Internal, "hashing salted computer digest")
		}

		//Salted resource digest
		var tmp2 [40]byte
		copy(computorPublicKey[:], tmp2[32:]) // Public key as the first part
		copy(resourceDigest[:], tmp2[:32])
		saltedResourceTestingDigest, err := utils.K12Hash(tmp2[:])
		if err != nil {
			return nil, status.Errorf(codes.Internal, "hashing salted resource testing digest")
		}

		//Add reconstructed object to response
		res.QuorumTickData.QuorumDiffPerComputor[id] = &protobuff.QuorumDiff{
			SaltedResourceTestingDigestHex: hex.EncodeToString(saltedResourceTestingDigest[:]),
			SaltedSpectrumDigestHex:        hex.EncodeToString(saltedSpectrumDigest[:]),
			SaltedUniverseDigestHex:        hex.EncodeToString(saltedUniverseDigest[:]),
			SaltedComputerDigestHex:        hex.EncodeToString(saltedComputerDigest[:]),
			ExpectedNextTickTxDigestHex:    voteDiff.ExpectedNextTickTxDigestHex,
			SignatureHex:                   voteDiff.SignatureHex,
		}
	}

	return &res, nil
}
