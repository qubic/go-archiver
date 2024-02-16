package computors

import (
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
	"testing"
)

func TestQubicModelToProtoAndReverse(t *testing.T) {
	computors := types.Computors{
		Epoch: 1,
		PubKeys: [types.NumberOfComputors][32]byte{
			{0xc3},
			{0xd4},
		},
		Signature: [64]byte{0x1, 0x2, 0x3, 0x4, 0xff, 0x6, 0x07, 0x8, 0x9, 0x10, 0x1, 0x2, 0x3, 0x4, 0x5, 0x6, 0x07, 0x8, 0x9, 0x10}, //01020304ff0607080910010203040506070809100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000
	}

	expected := &protobuff.Computors{
		Epoch: 1,
		Identities: []string{fillStringTo(64, "c3"), fillStringTo(64, "d4")},
		SignatureHex: "01020304ff0607080910010203040506070809100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
	}

	got := qubicToProto(computors)

	if diff := cmp.Diff(&got, &expected, cmpopts.IgnoreUnexported(protobuff.Computors{})); diff != "" {
		t.Fatalf("qubicToProto() mismatch (-got +want):\n%s", diff)
	}

	converted, err := protoToQubic(got)
	if err != nil {
		t.Fatalf("protoToQubic() error: %v", err)
	}

	if diff := cmp.Diff(converted, computors); diff != "" {
		t.Fatalf("protoToQubic() mismatch (-got +want):\n%s", diff)
	}
}

func fillStringTo(nrChars int, value string) string {
	for len(value) < nrChars {
		value = value + "0"
	}
	return value
}