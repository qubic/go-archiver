package rpc

import (
	"cmp"
	"context"
	"encoding/hex"
	"github.com/cockroachdb/pebble"
	"github.com/qubic/go-archiver/validator/tick"
	"log"
	"slices"

	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-node-connector/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

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

	wasSkipped, nextAvailableTick := tick.WasSkippedByArchive(req.TickNumber, processedTickIntervalsPerEpoch)
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

	wasSkipped, nextAvailableTick := tick.WasSkippedByArchive(req.TickNumber, processedTickIntervalsPerEpoch)
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

	wasSkipped, nextAvailableTick := tick.WasSkippedByArchive(req.TickNumber, processedTickIntervalsPerEpoch)
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

const maxPageSize uint32 = 100
const defaultPageSize uint32 = 100

func (s *Server) GetIdentityTransfersInTickRangeV2(ctx context.Context, req *protobuff.GetTransferTransactionsPerTickRequestV2) (*protobuff.GetIdentityTransfersInTickRangeResponseV2, error) {

	var pageSize uint32
	if req.GetPageSize() > maxPageSize { // max size
		return nil, status.Errorf(codes.InvalidArgument, "Invalid page size (maximum is %d).", maxPageSize)
	} else if req.GetPageSize() == 0 {
		pageSize = defaultPageSize // default
	} else {
		pageSize = req.GetPageSize()
	}
	pageNumber := max(0, int(req.Page)-1) // API index starts with '1', implementation index starts with '0'.
	txs, totalCount, err := s.store.GetTransactionsForEntityPaged(ctx, req.Identity,
		uint64(req.GetStartTick()), uint64(req.GetEndTick()),
		store.Pageable{Page: uint32(pageNumber), Size: pageSize},
		store.Sortable{Descending: req.Desc},
		store.Filterable{ScOnly: req.ScOnly})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting transfer transactions: %s", err.Error())
	}

	var totalTransactions []*protobuff.PerTickIdentityTransfers

	for _, transactionsPerTick := range txs {

		var tickTransactions []*protobuff.TransactionData

		for _, transaction := range transactionsPerTick.Transactions {
			transactionInfo, err := getTransactionInfo(ctx, s.store, transaction.TxId, transaction.TickNumber)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "Got Err: %s when getting transaction info for tx id: %s", err.Error(), transaction.TxId)
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

	pagination, err := getPaginationInformation(totalCount, pageNumber+1, int(pageSize))
	if err != nil {
		log.Printf("Error creating pagination info: %s", err.Error())
		return nil, status.Error(codes.Internal, "creating pagination info")
	}

	return &protobuff.GetIdentityTransfersInTickRangeResponseV2{
		Pagination:   pagination,
		Transactions: totalTransactions,
	}, nil

}

// ATTENTION: first page has pageNumber == 1 as API starts with index 1
func getPaginationInformation(totalRecords, pageNumber, pageSize int) (*protobuff.Pagination, error) {

	if pageNumber < 1 {
		return nil, errors.Errorf("invalid page number [%d]", pageNumber)
	}

	if pageSize < 1 {
		return nil, errors.Errorf("invalid page size [%d]", pageSize)
	}

	if totalRecords < 0 {
		return nil, errors.Errorf("invalid number of total records [%d]", totalRecords)
	}

	totalPages := totalRecords / pageSize // rounds down
	if totalRecords%pageSize != 0 {
		totalPages += 1
	}

	// next page starts at index 1. -1 if no next page.
	nextPage := pageNumber + 1
	if nextPage > totalPages {
		nextPage = -1
	}

	// previous page starts at index 1. -1 if no previous page
	previousPage := pageNumber - 1
	if previousPage == 0 {
		previousPage = -1
	}

	pagination := protobuff.Pagination{
		TotalRecords: int32(totalRecords),
		CurrentPage:  int32(min(totalRecords, pageNumber)), // 0 if there are no records
		TotalPages:   int32(totalPages),                    // 0 if there are no records
		PageSize:     int32(pageSize),
		NextPage:     int32(nextPage),                      // -1 if there is none
		PreviousPage: int32(min(totalPages, previousPage)), // -1 if there is none, do not exceed total pages
	}
	return &pagination, nil
}

func (s *Server) GetEmptyTickListV2(ctx context.Context, req *protobuff.GetEmptyTickListRequestV2) (*protobuff.GetEmptyTickListResponseV2, error) {

	if req.PageSize <= 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "page size must be at least 1")
	}

	emptyTicks, err := s.store.GetEmptyTickListPerEpoch(req.Epoch)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "no empty ticks found for epoch %d", req.Epoch)
		}
		return nil, status.Errorf(codes.Internal, "getting empty tick list: %v", err)
	}

	pageCount := len(emptyTicks) / int(req.PageSize)
	if len(emptyTicks)%int(req.PageSize) != 0 {
		pageCount += 1
	}
	currentPage := int(req.Page)
	if currentPage <= 0 {
		currentPage = 1
	}
	if currentPage > pageCount {
		return nil, status.Errorf(codes.NotFound, "cannot find specified page. last page: %d", pageCount)
	}

	startingIndex := (currentPage - 1) * int(req.PageSize)
	endingIndex := startingIndex + int(req.PageSize)
	if endingIndex > len(emptyTicks) {
		endingIndex = len(emptyTicks)
	}

	selectedTicks := emptyTicks[startingIndex:endingIndex]

	nextPage := currentPage + 1
	if currentPage == pageCount {
		nextPage = -1
	}

	previousPage := currentPage - 1
	if currentPage == 1 {
		previousPage = -1
	}

	return &protobuff.GetEmptyTickListResponseV2{
		EmptyTicks: selectedTicks,
		Pagination: &protobuff.Pagination{
			TotalRecords: int32(len(emptyTicks)),
			CurrentPage:  int32(currentPage),
			TotalPages:   int32(pageCount),
			PageSize:     req.PageSize,
			NextPage:     int32(nextPage),
			PreviousPage: int32(previousPage),
		},
	}, nil

}

func (s *Server) GetEpochTickListV2(ctx context.Context, req *protobuff.GetEpochTickListRequestV2) (*protobuff.GetEpochTickListResponseV2, error) {

	if req.PageSize <= 0 {
		return nil, status.Errorf(codes.FailedPrecondition, "page size must be at least 1")
	}

	intervals, err := s.store.GetProcessedTickIntervals(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get tick intervals: %v", err)
	}

	var processedTickIntervalsForEpoch []*protobuff.ProcessedTickInterval

	for _, interval := range intervals {
		if interval.Epoch != req.Epoch {
			continue
		}
		processedTickIntervalsForEpoch = interval.Intervals
	}

	if len(processedTickIntervalsForEpoch) == 0 {
		return nil, status.Errorf(codes.Internal, "no processed tick intervals found for epoch %d", req.Epoch)
	}

	emptyTicksForEpoch, err := s.store.GetEmptyTickListPerEpoch(req.Epoch)
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "no empty ticks found for epoch %d", req.Epoch)
		}
		return nil, status.Errorf(codes.Internal, "getting empty tick list: %v", err)
	}

	tickMap := make(map[uint32]bool)

	for _, interval := range processedTickIntervalsForEpoch {

		for i := interval.InitialProcessedTick; i <= interval.LastProcessedTick; i++ {
			tickMap[i] = false
		}
	}

	for _, tickNumber := range emptyTicksForEpoch {
		_, exists := tickMap[tickNumber]
		if exists {
			tickMap[tickNumber] = true
		}
	}

	var result []*protobuff.TickStatus

	for tickNumber, isEmpty := range tickMap {
		result = append(result, &protobuff.TickStatus{
			TickNumber: tickNumber,
			IsEmpty:    isEmpty,
		})
	}

	slices.SortFunc(result, func(a, b *protobuff.TickStatus) int {
		if req.Desc {
			return -cmp.Compare(a.TickNumber, b.TickNumber)
		}
		return cmp.Compare(a.TickNumber, b.TickNumber)
	})

	pageCount := len(result) / int(req.PageSize)
	if len(result)%int(req.PageSize) != 0 {
		pageCount += 1
	}
	currentPage := int(req.Page)
	if currentPage <= 0 {
		currentPage = 1
	}
	if currentPage > pageCount {
		return nil, status.Errorf(codes.NotFound, "cannot find specified page. last page: %d", pageCount)
	}

	startingIndex := (currentPage - 1) * int(req.PageSize)
	endingIndex := startingIndex + int(req.PageSize)
	if endingIndex > len(result) {
		endingIndex = len(result)
	}

	selectedTicks := result[startingIndex:endingIndex]

	nextPage := currentPage + 1
	if currentPage == pageCount {
		nextPage = -1
	}

	previousPage := currentPage - 1
	if currentPage == 1 {
		previousPage = -1
	}

	return &protobuff.GetEpochTickListResponseV2{
		Pagination: &protobuff.Pagination{
			TotalRecords: int32(len(result)),
			CurrentPage:  int32(currentPage),
			TotalPages:   int32(pageCount),
			PageSize:     req.PageSize,
			NextPage:     int32(nextPage),
			PreviousPage: int32(previousPage),
		},
		Ticks: selectedTicks,
	}, nil

}
