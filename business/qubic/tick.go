package qubic

import (
	"context"
	"encoding/hex"
	"github.com/pkg/errors"
	pb "github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
	"time"
)

type TickDataFetcher interface {
	GetTickData(ctx context.Context, tickNumber uint32) (types.TickData, error)
}

func GetTickData(ctx context.Context, fetcher TickDataFetcher, tickNumber uint32) (*pb.TickData, error) {
	qubicTd, err := fetcher.GetTickData(ctx, tickNumber)
	if err != nil {
		return nil, errors.Wrap(err, "fetching tick data")
	}

	td, err := tickDataQubicToProto(qubicTd)
	if err != nil {
		return nil, errors.Wrap(err, "converting tick data to proto")
	}

	return td, nil
}

func tickDataQubicToProto(tickData types.TickData) (*pb.TickData, error) {
	if tickData.IsEmpty() {
		return nil, nil
	}

	date := time.Date(2000+int(tickData.Year), time.Month(tickData.Month), int(tickData.Day), int(tickData.Hour), int(tickData.Minute), int(tickData.Second), 0, time.UTC)
	timestamp := date.UnixMilli() + int64(tickData.Millisecond)
	transactionIds, err := digestsToIdentities(tickData.TransactionDigests)
	if err != nil {
		return nil, errors.Wrap(err, "getting transaction ids from digests")
	}
	return &pb.TickData{
		ComputorIndex:  uint32(tickData.ComputorIndex),
		Epoch:          uint32(tickData.Epoch),
		TickNumber:     tickData.Tick,
		Timestamp:      uint64(timestamp),
		VarStruct:      tickData.UnionData[:],
		TimeLock:       tickData.Timelock[:],
		TransactionIds: transactionIds,
		ContractFees:   contractFeesToProto(tickData.ContractFees),
		SignatureHex:   hex.EncodeToString(tickData.Signature[:]),
	}, nil
}

func digestsToIdentities(digests [types.NumberOfTransactionsPerTick][32]byte) ([]string, error) {
	identities := make([]string, 0)
	for _, digest := range digests {
		if digest == [32]byte{} {
			continue
		}
		var id types.Identity
		id, err := id.FromPubKey(digest, true)
		if err != nil {
			return nil, errors.Wrapf(err, "getting identity from digest hex %s", hex.EncodeToString(digest[:]))
		}
		identities = append(identities, id.String())
	}

	return identities, nil
}

func contractFeesToProto(contractFees [1024]int64) []int64 {
	protoContractFees := make([]int64, 0, len(contractFees))
	for _, fee := range contractFees {
		if fee == 0 {
			continue
		}
		protoContractFees = append(protoContractFees, fee)
	}
	return protoContractFees
}
