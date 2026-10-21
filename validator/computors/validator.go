package computors

import (
	"context"
	"fmt"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-node-connector/v2/types"
)

func Validate(ctx context.Context, sigVerifierFunc utils.SigVerifierFunc, computors types.Computors, arbitratorPubKey [32]byte) error {
	digest, err := getDigestFromComputors(computors)
	if err != nil {
		return fmt.Errorf("getting digest from computors: %w", err)
	}

	err = sigVerifierFunc(ctx, arbitratorPubKey, digest, computors.Signature)
	if err != nil {
		return fmt.Errorf("verifying computors signature: %w", err)
	}

	return nil
}

func getDigestFromComputors(data types.Computors) ([32]byte, error) {
	sData, err := utils.BinarySerialize(data)
	if err != nil {
		return [32]byte{}, fmt.Errorf("serializing data: %w", err)
	}

	// remove signature from computors data
	computorsData := sData[:len(sData)-64]
	digest, err := utils.K12Hash(computorsData)
	if err != nil {
		return [32]byte{}, fmt.Errorf("hashing computors data: %w", err)
	}

	return digest, nil
}

func Store(ctx context.Context, store *store.PebbleStore, epoch uint16, computors types.Computors) error {
	protoModel, err := qubicToProto(computors)
	if err != nil {
		return fmt.Errorf("qubic to proto: %w", err)
	}

	err = store.SetComputors(ctx, uint32(epoch), protoModel)
	if err != nil {
		return fmt.Errorf("set computors: %w", err)
	}

	return nil
}

func Get(ctx context.Context, store *store.PebbleStore, epoch uint32) (types.Computors, error) {
	protoModel, err := store.GetComputors(ctx, epoch)
	if err != nil {
		return types.Computors{}, fmt.Errorf("get computors: %w", err)
	}

	model, err := protoToQubic(protoModel)
	if err != nil {
		return types.Computors{}, fmt.Errorf("proto to qubic: %w", err)
	}

	return model, nil
}
