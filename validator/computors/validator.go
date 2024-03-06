package computors

import (
	"context"
	"github.com/qubic/go-archiver/store"

	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-node-connector/types"
)

func Validate(ctx context.Context, computors types.Computors) error {
	arbitratorID := types.Identity(types.ArbitratorIdentity)
	arbitratorPubKey, err := arbitratorID.ToPubKey(false)
	if err != nil {
		return errors.Wrap(err, "getting arbitrator pubkey")
	}

	digest, err := getDigestFromComputors(computors)
	if err != nil {
		return errors.Wrap(err, "getting digest from computors")
	}

	err = utils.FourQSigVerify(ctx, arbitratorPubKey, digest, computors.Signature)
	if err != nil {
		return errors.Wrap(err, "verifying computors signature")
	}

	return nil
}

func getDigestFromComputors(data types.Computors) ([32]byte, error) {
	sData, err := utils.BinarySerialize(data)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "serializing data")
	}

	// remove signature from computors data
	computorsData := sData[:len(sData)-64]
	digest, err := utils.K12Hash(computorsData)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "hashing computors data")
	}

	return digest, nil
}

func Store(ctx context.Context, store *store.PebbleStore, epoch uint16, computors types.Computors) error {
	protoModel := qubicToProto(computors)

	err := store.SetComputors(ctx, uint32(epoch), protoModel)
	if err != nil {
		return errors.Wrap(err, "set computors")
	}

	return nil
}

func Get(ctx context.Context, store *store.PebbleStore, epoch uint32) (types.Computors, error) {
	protoModel, err := store.GetComputors(ctx, epoch)
	if err != nil {
		return types.Computors{}, errors.Wrap(err, "get computors")
	}

	model, err := protoToQubic(protoModel)
	if err != nil {
		return types.Computors{}, errors.Wrap(err, "proto to qubic")
	}

	return model, nil
}
