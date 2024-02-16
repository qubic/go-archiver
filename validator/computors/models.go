package computors

import (
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
)

func qubicToProto(computors types.Computors) *protobuff.Computors {
	return &protobuff.Computors {
		Epoch: uint32(computors.Epoch),
		Identities: identitiesToProto(computors.PubKeys),
		SignatureHex: hex.EncodeToString(computors.Signature[:]),
	}
}

func identitiesToProto(digests [types.NumberOfComputors][32]byte) []string {
	identitiesHex := make([]string, 0)
	for _, identity := range digests {
		if identity == [32]byte{} {
			continue
		}
		identitiesHex = append(identitiesHex, hex.EncodeToString(identity[:]))
	}
	return identitiesHex
}

func protoToQubic(computors *protobuff.Computors) (types.Computors, error) {
	pubKeys, err := protoToIdentities(computors.Identities)
	if err != nil {
		return types.Computors{}, errors.Wrap(err, "converting proto identities to qubic model")
	}

	var sig [64]byte
	sigBytes, err := hex.DecodeString(computors.SignatureHex)
	if err != nil {
		return types.Computors{}, errors.Wrap(err, "decoding signature")
	}
	copy(sig[:], sigBytes)

	return types.Computors {
		Epoch: uint16(int(computors.Epoch)),
		PubKeys:   pubKeys,
		Signature: sig,
	}, nil
}

func protoToIdentities(identities []string) ([types.NumberOfComputors][32]byte, error) {
	var digests [types.NumberOfComputors][32]byte
	for i, identity := range identities {
		if len(identity) == 0 {
			continue
		}
		pubKeyBytes, err := hex.DecodeString(identity)
		if err != nil {
			return [types.NumberOfComputors][32]byte{}, errors.Wrapf(err, "decoding identity: %s", identity)
		}
		copy(digests[i][:], pubKeyBytes)
	}

	return digests, nil
}