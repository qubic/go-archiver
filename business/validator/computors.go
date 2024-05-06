package validator

import (
	"context"
	"encoding/hex"
	"github.com/pkg/errors"
	pb "github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
)

type SigVerifierFunc = func(ctx context.Context, pubkey [32]byte, digest [32]byte, sig [64]byte) error

func ValidateComputors(ctx context.Context, sigVerifierFunc SigVerifierFunc, comps *pb.Computors) error {
	arbitratorID := types.Identity(types.ArbitratorIdentity)
	arbitratorPubKey, err := arbitratorID.ToPubKey(false)
	if err != nil {
		return errors.Wrap(err, "getting arbitrator pubkey")
	}

	digest, err := hex.DecodeString(comps.DigestHex)
	if err != nil {
		return errors.Wrap(err, "decoding digest")
	}

	sig, err := hex.DecodeString(comps.SignatureHex)
	if err != nil {
		return errors.Wrap(err, "decoding signature")
	}

	err = sigVerifierFunc(ctx, arbitratorPubKey, [32]byte(digest), [64]byte(sig))
	if err != nil {
		return errors.Wrap(err, "verifying computors signature")
	}

	return nil
}
