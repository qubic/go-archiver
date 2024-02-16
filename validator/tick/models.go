package tick

import (
	"encoding/hex"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
	"time"
)

func qubicToProto(tickData types.TickData) *protobuff.TickData {
	date := time.Date(2000+int(tickData.Year), time.Month(tickData.Month), int(tickData.Day), int(tickData.Hour), int(tickData.Minute), int(tickData.Second), 0, time.UTC)
	timestamp := date.UnixMilli() + int64(tickData.Millisecond)
	return &protobuff.TickData{
		ComputorIndex:         uint32(tickData.ComputorIndex),
		Epoch:                 uint32(tickData.Epoch),
		TickNumber:            tickData.Tick,
		Timestamp:             uint64(timestamp),
		VarStruct:             tickData.UnionData[:],
		TimeLock:              tickData.Timelock[:],
		TransactionDigestsHex: digestsToProto(tickData.TransactionDigests),
		ContractFees:          contractFeesToProto(tickData.ContractFees),
		SignatureHex:          hex.EncodeToString(tickData.Signature[:]),
	}
}

func digestsToProto(digests [types.NumberOfTransactionsPerTick][32]byte) []string {
	digestsHex := make([]string, 0)
	for _, digest := range digests {
		if digest == [32]byte{} {
			continue
		}
		digestsHex = append(digestsHex, hex.EncodeToString(digest[:]))
	}
	return digestsHex
}

func contractFeesToProto(contractFees [1024]int64) []int64 {
	protoContractFees := make([]int64, len(contractFees))
	for i, fee := range contractFees {
		protoContractFees[i] = fee
	}
	return protoContractFees
}
