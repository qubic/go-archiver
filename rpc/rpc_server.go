package rpc

import (
	"context"
	"encoding/hex"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/emptypb"
	"log"
	"net"
	"net/http"
)

var _ protobuff.ArchiveServiceServer = &Server{}

var emptyTd = &protobuff.TickData{}

type Server struct {
	protobuff.UnimplementedArchiveServiceServer
	listenAddrGRPC string
	listenAddrHTTP string
	store          *store.PebbleStore
}

func NewServer(listenAddrGRPC, listenAddrHTTP string, store *store.PebbleStore) *Server {
	return &Server{
		listenAddrGRPC: listenAddrGRPC,
		listenAddrHTTP: listenAddrHTTP,
		store:          store,
	}
}

func (s *Server) GetTickData(ctx context.Context, req *protobuff.GetTickDataRequest) (*protobuff.GetTickDataResponse, error) {
	tickData, err := s.store.GetTickData(ctx, req.TickNumber)
	if err != nil {
		if errors.Cause(err) == store.ErrNotFound {
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
	txs, err := s.store.GetTickTransactions(ctx, req.TickNumber)
	if err != nil {
		if errors.Cause(err) == store.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "tick transactions for specified tick not found")
		}
		return nil, status.Errorf(codes.Internal, "getting tick transactions: %v", err)
	}

	return &protobuff.GetTickTransactionsResponse{Transactions: txs}, nil
}

func (s *Server) GetTickTransferTransactions(ctx context.Context, req *protobuff.GetTickTransactionsRequest) (*protobuff.GetTickTransactionsResponse, error) {
	txs, err := s.store.GetTickTransferTransactions(ctx, req.TickNumber)
	if err != nil {
		if errors.Cause(err) == store.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "tick transfer transactions for specified tick not found")
		}
		return nil, status.Errorf(codes.Internal, "getting tick transactions: %v", err)
	}

	return &protobuff.GetTickTransactionsResponse{Transactions: txs}, nil
}
func (s *Server) GetTransaction(ctx context.Context, req *protobuff.GetTransactionRequest) (*protobuff.GetTransactionResponse, error) {
	tx, err := s.store.GetTransaction(ctx, req.TxId)
	if err != nil {
		if errors.Cause(err) == store.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "transaction not found")
		}
		return nil, status.Errorf(codes.Internal, "getting transaction: %v", err)
	}

	return &protobuff.GetTransactionResponse{Transaction: tx}, nil
}
func (s *Server) GetQuorumTickData(ctx context.Context, req *protobuff.GetQuorumTickDataRequest) (*protobuff.GetQuorumTickDataResponse, error) {
	qtd, err := s.store.GetQuorumTickData(ctx, req.TickNumber)
	if err != nil {
		if errors.Cause(err) == store.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "quorum tick data not found")
		}
		return nil, status.Errorf(codes.Internal, "getting quorum tick data: %v", err)
	}

	return &protobuff.GetQuorumTickDataResponse{QuorumTickData: qtd}, nil
}
func (s *Server) GetComputors(ctx context.Context, req *protobuff.GetComputorsRequest) (*protobuff.GetComputorsResponse, error) {
	computors, err := s.store.GetComputors(ctx, req.Epoch)
	if err != nil {
		if errors.Cause(err) == store.ErrNotFound {
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
		if errors.Cause(err) == store.ErrNotFound {
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

func (s *Server) GetLatestTick(ctx context.Context, _ *emptypb.Empty) (*protobuff.GetLatestTickResponse, error) {
	tick, err := s.store.GetLastProcessedTick(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting last processed tick: %v", err)
	}

	return &protobuff.GetLatestTickResponse{LatestTick: tick.TickNumber + 3}, nil
}

func (s *Server) GetTransferTransactionsPerTick(ctx context.Context, req *protobuff.GetTransferTransactionsPerTickRequest) (*protobuff.GetTransferTransactionsPerTickResponse, error) {
	txs, err := s.store.GetTransferTransactions(ctx, req.Identity, uint64(req.GetStartTick()), uint64(req.GetEndTick()))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting transfer transactions: %v", err)
	}

	return &protobuff.GetTransferTransactionsPerTickResponse{TransferTransactionsPerTick: txs}, nil
}

func (s *Server) GetChainHash(ctx context.Context, req *protobuff.GetChainHashRequest) (*protobuff.GetChainHashResponse, error) {
	hash, err := s.store.GetChainDigest(ctx, req.TickNumber)
	if err != nil {
		if errors.Cause(err) == store.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "chain hash for specified tick not found")
		}
		return nil, status.Errorf(codes.Internal, "getting chain hash: %v", err)
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
