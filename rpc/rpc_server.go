package rpc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	qubic "github.com/qubic/go-node-connector"
	"github.com/qubic/go-node-connector/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"io"
	"log"
	"net"
	"net/http"
)

var _ protobuff.ArchiveServiceServer = &Server{}

var emptyTd = &protobuff.TickData{}

type TransactionInfo struct {
	timestamp uint64
	moneyFlew bool
}

type Server struct {
	protobuff.UnimplementedArchiveServiceServer
	listenAddrGRPC    string
	listenAddrHTTP    string
	syncThreshold     int
	chainTickFetchUrl string
	store             *store.PebbleStore
	pool              *qubic.Pool
}

func NewServer(listenAddrGRPC, listenAddrHTTP string, syncThreshold int, chainTickUrl string, store *store.PebbleStore, pool *qubic.Pool) *Server {
	return &Server{
		listenAddrGRPC:    listenAddrGRPC,
		listenAddrHTTP:    listenAddrHTTP,
		syncThreshold:     syncThreshold,
		chainTickFetchUrl: chainTickUrl,
		store:             store,
		pool:              pool,
	}
}

func getTransactionInfo(ctx context.Context, pebbleStore *store.PebbleStore, transactionId string, tickNumber uint32) (*TransactionInfo, error) {

	txStatus, err := pebbleStore.GetTransactionStatus(ctx, transactionId)
	if err != nil {
		return nil, errors.Wrap(err, "getting transaction status")
	}

	tickData, err := pebbleStore.GetTickData(ctx, tickNumber)
	if err != nil {
		return nil, errors.Wrap(err, "getting tick data")
	}

	return &TransactionInfo{
		timestamp: tickData.Timestamp,
		moneyFlew: txStatus.MoneyFlew,
	}, nil

}

func (s *Server) GetTickData(ctx context.Context, req *protobuff.GetTickDataRequest) (*protobuff.GetTickDataResponse, error) {
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

	tickData, err := s.store.GetTickData(ctx, req.TickNumber)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "tick data not found")
		}
		return nil, status.Errorf(codes.Internal, "getting tick data: %v", err)
	}

	if tickData == emptyTd {
		tickData = nil
	}

	return &protobuff.GetTickDataResponse{TickData: tickData}, nil
}
func (s *Server) GetTickTransactions(ctx context.Context, req *protobuff.GetTickTransactionsRequest) (*protobuff.GetTickTransactionsResponse, error) {
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

	return &protobuff.GetTickTransactionsResponse{Transactions: txs}, nil
}

func (s *Server) GetTickTransferTransactions(ctx context.Context, req *protobuff.GetTickTransactionsRequest) (*protobuff.GetTickTransactionsResponse, error) {
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

	return &protobuff.GetTickTransactionsResponse{Transactions: txs}, nil
}
func (s *Server) GetTransaction(ctx context.Context, req *protobuff.GetTransactionRequest) (*protobuff.GetTransactionResponse, error) {
	tx, err := s.store.GetTransaction(ctx, req.TxId)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "transaction not found")
		}
		return nil, status.Errorf(codes.Internal, "getting transaction: %v", err)
	}

	return &protobuff.GetTransactionResponse{Transaction: tx}, nil
}
func (s *Server) GetQuorumTickData(ctx context.Context, req *protobuff.GetQuorumTickDataRequest) (*protobuff.GetQuorumTickDataResponse, error) {
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

	qtd, err := s.store.GetQuorumTickData(ctx, req.TickNumber)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "quorum tick data not found")
		}
		return nil, status.Errorf(codes.Internal, "getting quorum tick data: %v", err)
	}

	return &protobuff.GetQuorumTickDataResponse{QuorumTickData: qtd}, nil
}
func (s *Server) GetComputors(ctx context.Context, req *protobuff.GetComputorsRequest) (*protobuff.GetComputorsResponse, error) {
	computors, err := s.store.GetComputors(ctx, req.Epoch)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "computors not found")
		}
		return nil, status.Errorf(codes.Internal, "getting computors: %v", err)
	}

	return &protobuff.GetComputorsResponse{Computors: computors}, nil
}

func (s *Server) GetStatus(ctx context.Context, _ *emptypb.Empty) (*protobuff.GetStatusResponse, error) {
	tick, err := s.store.GetLastProcessedTick(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting last processed tick: %v", err)
	}

	lastProcessedTicksPerEpoch, err := s.store.GetLastProcessedTicksPerEpoch(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting last processed tick: %v", err)
	}

	skippedTicks, err := s.store.GetSkippedTicksInterval(ctx)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return &protobuff.GetStatusResponse{LastProcessedTick: tick, LastProcessedTicksPerEpoch: lastProcessedTicksPerEpoch}, nil
		}

		return nil, status.Errorf(codes.Internal, "getting skipped ticks: %v", err)
	}

	ptie, err := s.store.GetProcessedTickIntervals(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting processed tick intervals")
	}

	return &protobuff.GetStatusResponse{
		LastProcessedTick:              tick,
		LastProcessedTicksPerEpoch:     lastProcessedTicksPerEpoch,
		SkippedTicks:                   skippedTicks.SkippedTicks,
		ProcessedTickIntervalsPerEpoch: ptie,
	}, nil
}

type response struct {
	ChainTick int `json:"max_tick"`
}

func fetchChainTick(ctx context.Context, url string) (int, error) {

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, errors.Wrap(err, "creating new request")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, errors.Wrap(err, "getting chain tick from node fetcher")
	}
	defer res.Body.Close()

	var resp response
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return 0, errors.Wrap(err, "reading response body")
	}
	err = json.Unmarshal(body, &resp)
	if err != nil {
		return 0, errors.Wrap(err, "unmarshalling response")
	}

	tick := resp.ChainTick

	if tick == 0 {
		return 0, errors.New("response has no chain tick or chain tick is 0")
	}

	return tick, nil

}

func (s *Server) GetHealthCheck(ctx context.Context, _ *emptypb.Empty) (*protobuff.GetHealthCheckResponse, error) {
	//Get last processed tick
	lastProcessedTick, err := s.store.GetLastProcessedTick(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting last processed tick: %v", err)
	}

	//Poll node-fetcher for network tick
	chainTick, err := fetchChainTick(ctx, s.chainTickFetchUrl)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "fetching network tick: %v", err)
	}

	//Calculate difference between node tick and our last processed tick. difference = nodeTick - lastProcessed
	difference := chainTick - int(lastProcessedTick.TickNumber)

	//If the sync difference is bigger than our threshold
	if difference > s.syncThreshold {
		return nil, status.Errorf(codes.Internal, "processor is behind network by %d ticks", difference)
	}

	return &protobuff.GetHealthCheckResponse{Status: true}, nil

}

func (s *Server) GetLatestTick(ctx context.Context, _ *emptypb.Empty) (*protobuff.GetLatestTickResponseV2, error) {
	chainTick, err := fetchChainTick(ctx, s.chainTickFetchUrl)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "fetching chain tick: %v", err)
	}

	return &protobuff.GetLatestTickResponseV2{LatestTick: uint32(chainTick)}, nil
}

func (s *Server) GetTransferTransactionsPerTick(ctx context.Context, req *protobuff.GetTransferTransactionsPerTickRequest) (*protobuff.GetTransferTransactionsPerTickResponse, error) {
	txs, err := s.store.GetTransferTransactions(ctx, req.Identity, uint64(req.GetStartTick()), uint64(req.GetEndTick()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting transfer transactions: %v", err)
	}

	return &protobuff.GetTransferTransactionsPerTickResponse{TransferTransactionsPerTick: txs}, nil
}

func (s *Server) GetTickApprovedTransactions(ctx context.Context, req *protobuff.GetTickApprovedTransactionsRequest) (*protobuff.GetTickApprovedTransactionsResponse, error) {
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

	tts, err := s.store.GetTickTransactionsStatus(ctx, uint64(req.TickNumber))
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "tick transactions status data not found for tick %d", req.TickNumber)
		}
		return nil, status.Errorf(codes.Internal, "getting tick transactions status: %v", err)
	}

	approvedTxs := make([]*protobuff.Transaction, 0, len(tts.Transactions))
	for _, txStatus := range tts.Transactions {
		if txStatus.MoneyFlew == false {
			continue
		}

		tx, err := s.store.GetTransaction(ctx, txStatus.TxId)
		if err != nil {
			return nil, errors.Wrapf(err, "getting tx %s from archiver", txStatus.TxId)
		}

		if tx.InputType == 1 && tx.InputSize == 1000 && tx.DestId == "EAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAVWRF" {
			moneyFlew, err := recomputeSendManyMoneyFlew(tx)
			if err != nil {
				return nil, status.Errorf(codes.Internal, "recomputeSendManyMoneyFlew: %v", err)
			}

			if moneyFlew == false {
				continue
			}
		}

		approvedTxs = append(approvedTxs, &protobuff.Transaction{
			SourceId:     tx.SourceId,
			DestId:       tx.DestId,
			Amount:       tx.Amount,
			TickNumber:   tx.TickNumber,
			InputType:    tx.InputType,
			InputSize:    tx.InputSize,
			InputHex:     tx.InputHex,
			SignatureHex: tx.SignatureHex,
			TxId:         tx.TxId,
		})
	}

	return &protobuff.GetTickApprovedTransactionsResponse{ApprovedTransactions: approvedTxs}, nil
}

func wasTickSkippedByArchive(tick uint32, processedTicksIntervalPerEpoch []*protobuff.ProcessedTickIntervalsPerEpoch) (bool, uint32) {
	if len(processedTicksIntervalPerEpoch) == 0 {
		return false, 0
	}
	for _, epochInterval := range processedTicksIntervalPerEpoch {
		for _, interval := range epochInterval.Intervals {
			if tick < interval.InitialProcessedTick {
				return true, interval.InitialProcessedTick
			}
			if tick >= interval.InitialProcessedTick && tick <= interval.LastProcessedTick {
				return false, 0
			}
		}
	}
	return false, 0
}

func (s *Server) GetTransactionStatus(ctx context.Context, req *protobuff.GetTransactionStatusRequest) (*protobuff.GetTransactionStatusResponse, error) {
	id := types.Identity(req.TxId)
	pubKey, err := id.ToPubKey(true)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tx id format: %v", err)
	}

	var pubkeyFixed [32]byte
	copy(pubkeyFixed[:], pubKey[:32])
	id, err = id.FromPubKey(pubkeyFixed, true)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tx id format: %v", err)
	}

	if id.String() != req.TxId {
		return nil, status.Errorf(codes.InvalidArgument, "invalid tx id format")
	}

	tx, err := s.store.GetTransaction(ctx, req.TxId)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "tx status for specified tx id not found")
		}
		return nil, status.Errorf(codes.Internal, "getting tx status: %v", err)
	}

	lastProcessedTick, err := s.store.GetLastProcessedTick(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting last processed tick: %v", err)
	}

	if tx.TickNumber > lastProcessedTick.TickNumber {
		return nil, status.Errorf(codes.NotFound, "tx status for specified tx id not found")
	}

	if tx.Amount <= 0 {
		return nil, status.Errorf(codes.NotFound, "tx status for specified tx id not found")
	}

	txStatus, err := s.store.GetTransactionStatus(ctx, req.TxId)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return &protobuff.GetTransactionStatusResponse{TransactionStatus: &protobuff.TransactionStatus{TxId: tx.TxId, MoneyFlew: false}}, nil
		}
		return nil, status.Errorf(codes.Internal, "getting tx status: %v", err)
	}

	if tx.InputType == 1 && tx.InputSize == 1000 && tx.DestId == "EAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAVWRF" {
		moneyFlew, err := recomputeSendManyMoneyFlew(tx)
		if err != nil {
			return nil, status.Errorf(codes.Internal, "recomputeSendManyMoneyFlew: %v", err)
		}

		return &protobuff.GetTransactionStatusResponse{TransactionStatus: &protobuff.TransactionStatus{TxId: tx.TxId, MoneyFlew: moneyFlew}}, nil
	}

	return &protobuff.GetTransactionStatusResponse{TransactionStatus: txStatus}, nil
}

func (s *Server) GetChainHash(ctx context.Context, req *protobuff.GetChainHashRequest) (*protobuff.GetChainHashResponse, error) {
	hash, err := s.store.GetChainDigest(ctx, req.TickNumber)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "chain hash for specified tick not found")
		}
		return nil, status.Errorf(codes.Internal, "getting chain hash: %v", err)
	}

	return &protobuff.GetChainHashResponse{HexDigest: hex.EncodeToString(hash[:])}, nil
}

func (s *Server) GetStoreHash(ctx context.Context, req *protobuff.GetChainHashRequest) (*protobuff.GetChainHashResponse, error) {
	hash, err := s.store.GetStoreDigest(ctx, req.TickNumber)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "store hash for specified tick not found")
		}
		return nil, status.Errorf(codes.Internal, "getting store hash: %v", err)
	}

	return &protobuff.GetChainHashResponse{HexDigest: hex.EncodeToString(hash[:])}, nil
}

func (s *Server) Start() error {
	srv := grpc.NewServer(
		grpc.MaxRecvMsgSize(600*1024*1024),
		grpc.MaxSendMsgSize(600*1024*1024),
	)
	protobuff.RegisterArchiveServiceServer(srv, s)
	reflection.Register(srv)

	lis, err := net.Listen("tcp", s.listenAddrGRPC)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	go func() {
		if err := srv.Serve(lis); err != nil {
			panic(err)
		}
	}()

	if s.listenAddrHTTP != "" {
		go func() {
			mux := runtime.NewServeMux(runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
				MarshalOptions: protojson.MarshalOptions{EmitDefaultValues: true, EmitUnpopulated: false},
			}))
			opts := []grpc.DialOption{
				grpc.WithTransportCredentials(insecure.NewCredentials()),
				grpc.WithDefaultCallOptions(
					grpc.MaxCallRecvMsgSize(600*1024*1024),
					grpc.MaxCallSendMsgSize(600*1024*1024),
				),
			}

			if err := protobuff.RegisterArchiveServiceHandlerFromEndpoint(
				context.Background(),
				mux,
				s.listenAddrGRPC,
				opts,
			); err != nil {
				panic(err)
			}

			if err := http.ListenAndServe(s.listenAddrHTTP, mux); err != nil {
				panic(err)
			}
		}()
	}

	return nil
}

func recomputeSendManyMoneyFlew(tx *protobuff.Transaction) (bool, error) {
	decodedInput, err := hex.DecodeString(tx.InputHex)
	if err != nil {
		return false, status.Errorf(codes.Internal, "decoding tx input: %v", err)
	}
	var sendmanypayload types.SendManyTransferPayload
	err = sendmanypayload.UnmarshallBinary(decodedInput)
	if err != nil {
		return false, status.Errorf(codes.Internal, "unmarshalling payload: %v", err)
	}

	if tx.Amount < sendmanypayload.GetTotalAmount() {
		return false, nil
	}

	return true, nil
}
