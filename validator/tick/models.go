package tick

import (
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
	"time"
)

func qubicToProto(tickData types.TickData) (*protobuff.TickData, error) {
	date := time.Date(2000+int(tickData.Year), time.Month(tickData.Month), int(tickData.Day), int(tickData.Hour), int(tickData.Minute), int(tickData.Second), 0, time.UTC)
	timestamp := date.UnixMilli() + int64(tickData.Millisecond)
	transactionIds, err := digestsToIdentities(tickData.TransactionDigests)
	if err != nil {
		return nil, errors.Wrap(err, "getting transaction ids from digests")
	}
	return &protobuff.TickData{
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
	protoContractFees := make([]int64, len(contractFees))
	for i, fee := range contractFees {
		if fee == 0 {
			continue
		}
		protoContractFees[i] = fee
	}
	return protoContractFees
}
