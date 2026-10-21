package computors

import (
	"encoding/hex"
	"fmt"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/v2/types"
)

func qubicToProto(computors types.Computors) (*protobuff.Computors, error) {
	identities, err := pubKeysToIdentities(computors.PubKeys)
	if err != nil {
		return nil, fmt.Errorf("converting pubKeys to identities: %w", err)

	}
	return &protobuff.Computors{
		Epoch:        uint32(computors.Epoch),
		Identities:   identities,
		SignatureHex: hex.EncodeToString(computors.Signature[:]),
	}, nil
}

func pubKeysToIdentities(pubKeys [types.NumberOfComputors][32]byte) ([]string, error) {
	identities := make([]string, 0)
	for _, identity := range pubKeys {
		if identity == [32]byte{} {
			continue
		}
		var id types.Identity
		id, err := id.FromPubKey(identity, false)
		if err != nil {
			return nil, fmt.Errorf("getting identity from pubKey hex %s: %w", hex.EncodeToString(identity[:]), err)
		}
		identities = append(identities, id.String())
	}
	return identities, nil
}

func protoToQubic(computors *protobuff.Computors) (types.Computors, error) {
	pubKeys, err := identitiesToPubKeys(computors.Identities)
	if err != nil {
		return types.Computors{}, fmt.Errorf("converting proto identities to qubic model: %w", err)
	}

	var sig [64]byte
	sigBytes, err := hex.DecodeString(computors.SignatureHex)
	if err != nil {
		return types.Computors{}, fmt.Errorf("decoding signature: %w", err)
	}
	copy(sig[:], sigBytes)

	return types.Computors{
		Epoch:     uint16(int(computors.Epoch)),
		PubKeys:   pubKeys,
		Signature: sig,
	}, nil
}

func identitiesToPubKeys(identities []string) ([types.NumberOfComputors][32]byte, error) {
	var pubKeys [types.NumberOfComputors][32]byte
	for i, identity := range identities {
		if len(identity) == 0 {
			continue
		}
		id := types.Identity(identity)
		pubKeyBytes, err := id.ToPubKey(false)
		if err != nil {
			return [types.NumberOfComputors][32]byte{}, fmt.Errorf("converting identity to pubkey: %s: %w", identity, err)
		}
		pubKeys[i] = pubKeyBytes
	}

	return pubKeys, nil
}
