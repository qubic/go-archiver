package rpc

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	qubic "github.com/qubic/go-node-connector"
	"github.com/qubic/go-node-connector/types"
	"github.com/silenceper/pool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"log"
	"net"
	"net/http"
)

var _ protobuff.ArchiveServiceServer = &Server{}

type Server struct {
	protobuff.UnimplementedArchiveServiceServer
	listenAddrGRPC       string
	listenAddrHTTP       string
	store                *store.PebbleStore
	pool                 pool.Pool
	nrPeersToBroadcastTx int
}

func NewServer(listenAddrGRPC, listenAddrHTTP string, store *store.PebbleStore, p pool.Pool, nrPeersToBroadcastTx int) *Server {
	return &Server{
		listenAddrGRPC:       listenAddrGRPC,
		listenAddrHTTP:       listenAddrHTTP,
		store:                store,
		pool:                 p,
		nrPeersToBroadcastTx: nrPeersToBroadcastTx,
	}
}

func (s *Server) GetTickData(ctx context.Context, req *protobuff.GetTickDataRequest) (*protobuff.GetTickDataResponse, error) {
	tickData, err := s.store.GetTickData(ctx, uint64(req.TickNumber))
	if err != nil {
		if errors.Cause(err) == store.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "tick data not found")
		}
		return nil, status.Errorf(codes.Internal, "getting tick data: %v", err)
	}

	return &protobuff.GetTickDataResponse{TickData: tickData}, nil
}
func (s *Server) GetTickTransactions(ctx context.Context, req *protobuff.GetTickTransactionsRequest) (*protobuff.GetTickTransactionsResponse, error) {
	txs, err := s.store.GetTickTransactions(ctx, uint64(req.TickNumber))
	if err != nil {
		if errors.Cause(err) == store.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "tick transactions for specified tick not found")
		}
		return nil, status.Errorf(codes.Internal, "getting tick transactions: %v", err)
	}

	return &protobuff.GetTickTransactionsResponse{Transactions: txs.Transactions}, nil
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
	qtd, err := s.store.GetQuorumTickData(ctx, uint64(req.TickNumber))
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
func (s *Server) GetIdentityInfo(ctx context.Context, req *protobuff.GetIdentityInfoRequest) (*protobuff.GetIdentityInfoResponse, error) {
	qcv, err := s.pool.Get()
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting qubic pooled client connection: %v", err)
	}
	defer s.pool.Put(qcv)

	client := qcv.(*qubic.Client)
	addr, err := client.GetIdentity(ctx, req.Identity)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting identity info: %v", err)
	}

	siblings := make([]string, 0)
	for _, sibling := range addr.Siblings {
		if sibling == [32]byte{} {
			continue
		}
		siblings = append(siblings, hex.EncodeToString(sibling[:]))
	}

	var addrID types.Identity
	addrID, err = addrID.FromPubKey(addr.AddressData.PublicKey, false)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting address id: %v", err)
	}
	return &protobuff.GetIdentityInfoResponse{IdentityInfo: &protobuff.IdentityInfo{
		Id:                         addrID.String(),
		TickNumber:                 addr.Tick,
		Balance:                    addr.AddressData.IncomingAmount - addr.AddressData.OutgoingAmount,
		IncomingAmount:             addr.AddressData.IncomingAmount,
		OutgoingAmount:             addr.AddressData.OutgoingAmount,
		NrIncomingTransfers:        addr.AddressData.NumberOfIncomingTransfers,
		NrOutgoingTransfers:        addr.AddressData.NumberOfOutgoingTransfers,
		LatestIncomingTransferTick: addr.AddressData.LatestIncomingTransferTick,
		LatestOutgoingTransferTick: addr.AddressData.LatestOutgoingTransferTick,
		SiblingsHex:                siblings,
	}}, nil
}

func (s *Server) GetLastProcessedTick(ctx context.Context, req *protobuff.GetLastProcessedTickRequest) (*protobuff.GetLastProcessedTickResponse, error) {
	tick, err := s.store.GetLastProcessedTick(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting last processed tick: %v", err)
	}

	lastProcessedTicksPerEpoch, err := s.store.GetLastProcessedTicksPerEpoch(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting last processed tick: %v", err)
	}

	return &protobuff.GetLastProcessedTickResponse{LastProcessedTick: uint32(tick), LastProcessedTicksPerEpoch: lastProcessedTicksPerEpoch}, nil
}

func (s *Server) SendRawTransaction(ctx context.Context, req *protobuff.SendRawTransactionRequest) (*protobuff.SendRawTransactionResponse, error) {
	nrSuccess := 0
	for i := 0; i < s.nrPeersToBroadcastTx; i++ {
		func() {
			qcv, err := s.pool.Get()
			if err != nil {
				return
			}
			defer s.pool.Put(qcv)

			client := qcv.(*qubic.Client)
			err = client.SendRawTransaction(ctx, []byte(req.SignedTx))
			if err != nil {
				return
			}
			nrSuccess++
		}()
	}

	if nrSuccess == 0 {
		return nil, status.Errorf(codes.Internal, "broadcasting tx failed for all peers")
	}

	return &protobuff.SendRawTransactionResponse{Message: fmt.Sprintf("Transaction broadcasted to %d peers", nrSuccess)}, nil
}

func (s *Server) GetSkippedTicks(ctx context.Context, req *protobuff.GetSkippedTicksRequest) (*protobuff.GetSkippedTicksResponse, error) {
	ticks, err := s.store.GetSkippedTicksInterval(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting skipped ticks: %v", err)
	}

	return &protobuff.GetSkippedTicksResponse{SkippedTicks: ticks.SkippedTicks}, nil
}

func (s *Server) GetTransferTransactions(ctx context.Context, req *protobuff.GetTransferTransactionsRequest) (*protobuff.GetTransferTransactionsResponse, error) {
	txs, err := s.store.GetTransferTransactions(ctx, req.Identity)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "getting transfer transactions: %v", err)
	}

	return &protobuff.GetTransferTransactionsResponse{TransferTransactions: txs}, nil
}
func (s *Server) GetTransferTransactionsPerTick(ctx context.Context, req *protobuff.GetTransferTransactionsPerTickRequest) (*protobuff.GetTransferTransactionsPerTickResponse, error) {
	txs, err := s.store.GetTransferTransactionsPerTick(ctx, req.Identity, uint64(req.TickNumber))
	if err != nil {
		if errors.Cause(err) == store.ErrNotFound {
			return nil, status.Errorf(codes.NotFound, "transfer transactions for specified identity, tick pair not found")
		}
		return nil, status.Errorf(codes.Internal, "getting transfer transactions: %v", err)
	}

	return &protobuff.GetTransferTransactionsPerTickResponse{TransferTransactionsPerTick: txs}, nil
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
