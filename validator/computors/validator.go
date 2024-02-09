package computors

import (
	"context"

	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-node-connector/types"

)

func Validate(ctx context.Context, computors types.Computors) error {
	arbitratorID := types.Identity(types.ArbitratorIdentity)
	arbitratorPubKey, err := arbitratorID.ToPubKey()
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
