package rpc

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/validator/tick"
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

func (s *Server) GetTickData(ctx context.Context, req *protobuff.GetTickDataRequest) (*protobuff.GetTickDataResponse, error) {
	tickData, err := s.store.GetTickData(req.TickNumber)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "tick data not found")
		}
		return nil, status.Errorf(codes.Internal, "getting tick data: %v", err)
	}

	if tickData == emptyTd {
		tickData = nil
	}

	if tick.CheckIfTickIsEmptyProto(tickData) {
		tickData = nil
	}

	return &protobuff.GetTickDataResponse{TickData: tickData}, nil
}

func (s *Server) GetTickTransactions(ctx context.Context, req *protobuff.GetTickTransactionsRequest) (*protobuff.GetTickTransactionsResponse, error) {

	txs, err := s.store.GetTickTransactions(req.TickNumber)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "tick transactions for specified tick not found")
		}
		return nil, status.Errorf(codes.Internal, "getting tick transactions: %v", err)
	}

	return &protobuff.GetTickTransactionsResponse{Transactions: txs}, nil
}

/*
func (s *Server) GetTickTransferTransactions(ctx context.Context, req *protobuff.GetTickTransactionsRequest) (*protobuff.GetTickTransactionsResponse, error) {

		txs, err := s.store.GetTickTransferTransactions(ctx, req.TickNumber)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				return nil, status.Errorf(codes.NotFound, "tick transfer transactions for specified tick not found")
			}
			return nil, status.Errorf(codes.Internal, "getting tick transactions: %v", err)
		}

		return &protobuff.GetTickTransactionsResponse{Transactions: txs}, nil
	}
*/
func (s *Server) GetTransaction(ctx context.Context, req *protobuff.GetTransactionRequest) (*protobuff.GetTransactionResponse, error) {
	tx, err := s.store.GetTransaction(req.TxId)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "transaction not found")
		}
		return nil, status.Errorf(codes.Internal, "getting transaction: %v", err)
	}

	return &protobuff.GetTransactionResponse{Transaction: tx}, nil
}

func (s *Server) GetComputors(ctx context.Context, req *protobuff.GetComputorsRequest) (*protobuff.GetComputorsResponse, error) {
	computors, err := s.store.GetCurrentEpochStore().GetComputorList()
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return nil, status.Errorf(codes.NotFound, "computors not found")
		}
		return nil, status.Errorf(codes.Internal, "getting computors: %v", err)
	}

	return &protobuff.GetComputorsResponse{Computors: computors}, nil
}

func (s *Server) GetStatus(ctx context.Context, _ *emptypb.Empty) (*protobuff.GetStatusResponse, error) {
	lastProcessedTick, err := s.store.GetLastProcessedTick()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting last processed tick: %v", err)
	}

	lastProcessedTicksPerEpoch, err := s.store.GetLastProcessedTicksPerEpoch()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting last processed tick: %v", err)
	}

	ptie, err := s.store.GetProcessedTickIntervals()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting processed tick intervals")
	}

	return &protobuff.GetStatusResponse{
		LastProcessedTick:              lastProcessedTick,
		LastProcessedTicksPerEpoch:     lastProcessedTicksPerEpoch,
		SkippedTicks:                   nil,
		ProcessedTickIntervalsPerEpoch: ptie,
		EmptyTicksPerEpoch:             nil,
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

	tickNumber := resp.ChainTick

	if tickNumber == 0 {
		return 0, errors.New("response has no chain tick or chain tick is 0")
	}

	return tickNumber, nil

}

func (s *Server) GetHealthCheck(ctx context.Context, _ *emptypb.Empty) (*protobuff.GetHealthCheckResponse, error) {
	//Get last processed tick
	lastProcessedTick, err := s.store.GetLastProcessedTick()
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

func (s *Server) GetLatestTick(ctx context.Context, _ *emptypb.Empty) (*protobuff.GetLatestTickResponse, error) {
	chainTick, err := fetchChainTick(ctx, s.chainTickFetchUrl)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "fetching chain tick: %v", err)
	}

	return &protobuff.GetLatestTickResponse{LatestTick: uint32(chainTick)}, nil
}

func (s *Server) Start() error {

	//TODO
	//tickInBoundsInterception := NewTickWithinBoundsInterceptor(s.store)

	srv := grpc.NewServer(
		grpc.MaxRecvMsgSize(600*1024*1024),
		grpc.MaxSendMsgSize(600*1024*1024),
		//grpc.UnaryInterceptor(tickInBoundsInterception.GetInterceptor),
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
				MarshalOptions: protojson.MarshalOptions{EmitDefaultValues: true, EmitUnpopulated: true},
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
