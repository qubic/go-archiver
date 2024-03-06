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
			SourcePublicKey:      [32]byte{230, 252, 58, 173, 75, 89, 77, 130, 191, 49, 3, 161, 16, 22, 216, 13, 232, 131, 222, 135, 59, 206, 196, 142, 144, 57, 98, 134, 80, 59, 38, 19},    //QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB
			DestinationPublicKey: [32]byte{202, 170, 77, 59, 174, 172, 46, 236, 91, 33, 251, 190, 210, 221, 128, 54, 108, 203, 61, 60, 6, 180, 238, 166, 114, 128, 99, 30, 106, 188, 66, 81}, //IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB
			Amount:               100,
			Tick:                 20,
			InputType:            0,
			InputSize:            100,
			Input:                []byte{0x01, 0x02, 0x03, 0x04, 0x05},
			Signature:            [64]byte{0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x07, 0x8, 0x9, 0x10, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x07, 0x8, 0x9, 0x10}, //01020304050607080910010203040506070809100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
		},
		types.Transaction{
			SourcePublicKey:      [32]byte{202, 170, 77, 59, 174, 172, 46, 236, 91, 33, 251, 190, 210, 221, 128, 54, 108, 203, 61, 60, 6, 180, 238, 166, 114, 128, 99, 30, 106, 188, 66, 81}, //IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB
			DestinationPublicKey: [32]byte{230, 252, 58, 173, 75, 89, 77, 130, 191, 49, 3, 161, 16, 22, 216, 13, 232, 131, 222, 135, 59, 206, 196, 142, 144, 57, 98, 134, 80, 59, 38, 19},    //QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB
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

	var idOne types.Identity
	idOne, err = idOne.FromPubKey(digestOne, true)
	if err != nil {
		t.Fatalf("idOne.FromPubKey() unexpected error: %v", err)
	}

	digestTwo, err := qubicTransactions[1].Digest()
	if err != nil {
		t.Fatalf("qubicTransactions[1].Digest() unexpected error: %v", err)
	}

	var idTwo types.Identity
	idTwo, err = idTwo.FromPubKey(digestTwo, true)
	if err != nil {
		t.Fatalf("idTwo.FromPubKey() unexpected error: %v", err)
	}

	expectedProtoTxs := &protobuff.Transactions{
		Transactions: []*protobuff.Transaction{
			&protobuff.Transaction{
				SourceId:     "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
				DestId:       "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
				Amount:       100,
				TickNumber:   20,
				InputType:    0,
				InputSize:    100,
				InputHex:     hex.EncodeToString(qubicTransactions[0].Input[:]),
				SignatureHex: "01020304050607080910010203040506070809100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				TxId:         idOne.String(),
			},
			&protobuff.Transaction{
				SourceId:     "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
				DestId:       "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
				Amount:       15,
				TickNumber:   20,
				InputType:    0,
				InputSize:    120,
				InputHex:     hex.EncodeToString(qubicTransactions[1].Input[:]),
				SignatureHex: "01020304ff0607080910010203040506070809100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
				TxId:         idTwo.String(),
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
