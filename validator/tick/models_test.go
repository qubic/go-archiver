package tick

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
	"testing"
)

func TestQubicToProto(t *testing.T) {
	txOne := types.Transaction{
		SourcePublicKey:      [32]byte{0x01, 0xd1, 0xf1, 0x00, 0xc2},
		DestinationPublicKey: [32]byte{0x00, 0xff, 0xf1, 0x00, 0xff},
		Amount:               100,
		Tick:                 20,
		InputType:            0,
		InputSize:            100,
		Input:                []byte{0x01, 0x02, 0x03, 0x04, 0x05},
		Signature:            [64]byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x07, 0x8, 0x9, 0x10, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x07, 0x8, 0x9, 0x10}, //01020304050607080910010203040506070809100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
	}

	digestOne, err := txOne.Digest()
	if err != nil {
		t.Fatalf("txOne.Digest() unexpected error: %v", err)
	}

	var idOne types.Identity
	idOne, err = idOne.FromPubKey(digestOne, true)
	if err != nil {
		t.Fatalf("idOne.FromPubKey() unexpected error: %v", err)
	}

	txTwo := types.Transaction{
		SourcePublicKey:      [32]byte{0x01, 0xd1, 0xf1, 0x00, 0xc2},
		DestinationPublicKey: [32]byte{0x00, 0xff, 0xf1, 0x00, 0xff},
		Amount:               15,
		Tick:                 20,
		InputType:            0,
		InputSize:            120,
		Input:                []byte{0x01, 0x02, 0xda, 0x04, 0x05},
		Signature:            [64]byte{0x1, 0x2, 0x3, 0x4, 0xff, 0x6, 0x07, 0x8, 0x9, 0x10, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x07, 0x8, 0x9, 0x10}, //01020304ff0607080910010203040506070809100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
	}

	digestTwo, err := txTwo.Digest()
	if err != nil {
		t.Fatalf("txTwo.Digest() unexpected error: %v", err)
	}

	var idTwo types.Identity
	idTwo, err = idTwo.FromPubKey(digestTwo, true)
	if err != nil {
		t.Fatalf("idTwo.FromPubKey() unexpected error: %v", err)
	}

	timeLock := [32]byte{6, 4, 7, 4, 2}
	transactionDigests := [1024][32]byte{digestOne, digestTwo}
	contractFees := [1024]int64{1, 2, 3, 4, 5, 6, 7, 8, 9}
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
		Timelock:           timeLock,
		TransactionDigests: transactionDigests,
		ContractFees:       contractFees,
		Signature:          signature,
	}

	expectedProtoTickData := protobuff.TickData{
		ComputorIndex:  15,
		Epoch:          20,
		TickNumber:     100,
		Timestamp:      1685937782001,
		TimeLock:       timeLock[:],
		TransactionIds: []string{idOne.String(), idTwo.String()},
		ContractFees:   []int64{1, 2, 3, 4, 5, 6, 7, 8, 9},
		SignatureHex:   fillStringTo(128, "0102030405060708091001020304050607080910"),
	}

	got, err := QubicToProto(qubicTickData)
	if err != nil {
		t.Fatalf("qubicToProto() unexpected error: %v", err)
	}
	if diff := cmp.Diff(got, &expectedProtoTickData, cmpopts.IgnoreUnexported(protobuff.TickData{})); diff != "" {
		t.Fatalf("qubicToProto() mismatch (-got +want):\n%s", diff)
	}

	got, err = QubicToProto(types.TickData{})
	if err != nil {
		t.Fatalf("qubicToProto() unexpected error: %v", err)
	}
	if got != nil {
		t.Fatalf("should have get a nil proto model for an empty tick data")
	}
}

func fillStringTo(nrChars int, value string) string {
	for len(value) < nrChars {
		value = value + "0"
	}
	return value
}
