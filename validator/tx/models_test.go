package tx

import (
	"encoding/hex"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
	"testing"
)

func TestQubicToProto(t *testing.T) {
	qubicTransactions := types.Transactions{
		types.Transaction{
			SourcePublicKey:      [32]byte{0x01, 0xd1, 0xf1, 0x00, 0xc2},
			DestinationPublicKey: [32]byte{0x00, 0xff, 0xf1, 0x00, 0xff},
			Amount:               100,
			Tick:                 20,
			InputType:            0,
			InputSize:            100,
			Input:                []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			Signature:            [64]byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x07, 0x8, 0x9, 0x10, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x07, 0x8, 0x9, 0x10}, //01020304050607080910010203040506070809100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
		},
		types.Transaction{
			SourcePublicKey:      [32]byte{0x01, 0xd1, 0xf1, 0x00, 0xc2},
			DestinationPublicKey: [32]byte{0x00, 0xff, 0xf1, 0x00, 0xff},
			Amount:               15,
			Tick:                 20,
			InputType:            0,
			InputSize:            120,
			Input:                []byte{0x01, 0x02, 0xda, 0x04, 0x05},
			Signature:            [64]byte{0x1, 0x2, 0x3, 0x4, 0xff, 0x6, 0x07, 0x8, 0x9, 0x10, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x07, 0x8, 0x9, 0x10}, //01020304ff0607080910010203040506070809100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
		},
	}

	digestOne, err := qubicTransactions[0].Digest()
	if err != nil {
		t.Fatalf("qubicTransactions[0].Digest() unexpected error: %v", err)
	}

	digestTwo, err := qubicTransactions[1].Digest()
	if err != nil {
		t.Fatalf("qubicTransactions[1].Digest() unexpected error: %v", err)
	}

	expectedProtoTxs := &protobuff.Transactions{
		Transactions: []*protobuff.Transaction{
			&protobuff.Transaction{
				SourcePubkeyHex: hex.EncodeToString(qubicTransactions[0].SourcePublicKey[:]),
				DestPubkeyHex:   hex.EncodeToString(qubicTransactions[0].DestinationPublicKey[:]),
				Amount:          100,
				TickNumber:      20,
				InputType:       0,
				InputSize:       100,
				InputHex:        hex.EncodeToString(qubicTransactions[0].Input[:]),
				SignatureHex:    "01020304050607080910010203040506070809100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				DigestHex: hex.EncodeToString(digestOne[:]),
			},
			&protobuff.Transaction{
				SourcePubkeyHex: hex.EncodeToString(qubicTransactions[1].SourcePublicKey[:]),
				DestPubkeyHex:   hex.EncodeToString(qubicTransactions[1].DestinationPublicKey[:]),
				Amount:          15,
				TickNumber:      20,
				InputType:       0,
				InputSize:       120,
				InputHex:        hex.EncodeToString(qubicTransactions[1].Input[:]),
				SignatureHex:    "01020304ff0607080910010203040506070809100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				DigestHex: hex.EncodeToString(digestTwo[:]),
			},
		},
	}

	got, err := qubicToProto(qubicTransactions)
	if err != nil {
		t.Fatalf("qubicToProto() unexpected error: %v", err)
	}
	if diff := cmp.Diff(got, expectedProtoTxs, cmpopts.IgnoreUnexported(protobuff.Transactions{}, protobuff.Transaction{})); diff != "" {
		t.Fatalf("qubicToProto() mismatch (-got +want):\n%s", diff)
	}
}
