package tick

import (
	"encoding/hex"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
	"testing"
)

func TestQubicToProto(t *testing.T) {
	unionData := [256]byte{1, 2, 3, 4, 5}
	timeLock := [32]byte{6, 4, 7, 4, 2}
	digestOne := [32]byte{0xff, 0x12, 0x32, 0x11, 0x00}
	digestTwo := [32]byte{0xda, 0x3a, 0x0a, 0x00, 0xff}
	transactionDigests := [1024][32]byte{digestOne, digestTwo}
	contractFees := [1024]int64{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	signature := [64]byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x07, 0x8, 0x9, 0x10, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x07, 0x8, 0x9, 0x10}
	qubicTickData := types.TickData{
		ComputorIndex:      15,
		Epoch:              20,
		Tick:               100,
		Millisecond:        1,
		Second:             2,
		Minute:             3,
		Hour:               4,
		Day:                5,
		Month:              6,
		Year:               23,
		UnionData:          unionData,
		Timelock:           timeLock,
		TransactionDigests: transactionDigests,
		ContractFees:       contractFees,
		Signature:          signature,
	}

	expectedProtoTickData := &protobuff.TickData{
		ComputorIndex:         15,
		Epoch:                 20,
		TickNumber:            100,
		Timestamp:             1685937782001,
		VarStruct:             unionData[:],
		TimeLock:              timeLock[:],
		TransactionDigestsHex: []string{hex.EncodeToString(digestOne[:]), hex.EncodeToString(digestTwo[:])},
		ContractFees:          contractFees[:],
		SignatureHex:          fillStringTo(128, "0102030405060708091001020304050607080910"),
	}

	got := qubicToProto(qubicTickData)
	if diff := cmp.Diff(got, expectedProtoTickData, cmpopts.IgnoreUnexported(protobuff.TickData{})); diff != "" {
		t.Fatalf("qubicToProto() mismatch (-got +want):\n%s", diff)
	}
}

func fillStringTo(nrChars int, value string) string {
	for len(value) < nrChars {
		value = value + "0"
	}
	return value
}


