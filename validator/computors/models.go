package computors

import (
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
)

func qubicToProto(computors types.Computors) (*protobuff.Computors, error) {
	identities, err := pubKeysToIdentities(computors.PubKeys)
	if err != nil {
		return nil, errors.Wrap(err, "converting pubKeys to identities")

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
			return nil, errors.Wrapf(err, "getting identity from pubKey hex %s", hex.EncodeToString(identity[:]))
		}
		identities = append(identities, id.String())
	}
	return identities, nil
}

func ProtoToQubic(computors *protobuff.Computors) (types.Computors, error) {
	pubKeys, err := identitiesToPubKeys(computors.Identities)
	if err != nil {
		return types.Computors{}, errors.Wrap(err, "converting proto identities to qubic model")
	}

	var sig [64]byte
	sigBytes, err := hex.DecodeString(computors.SignatureHex)
	if err != nil {
		return types.Computors{}, errors.Wrap(err, "decoding signature")
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
			return [types.NumberOfComputors][32]byte{}, errors.Wrapf(err, "converting identity to pubkey: %s", identity)
		}
		pubKeys[i] = pubKeyBytes
	}

	return pubKeys, nil
}
