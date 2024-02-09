package tick

import (
	"context"
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-node-connector/types"
)

func Validate(ctx context.Context, data types.TickData, quorumTickData types.QuorumTickData, comps types.Computors) error {
	computorPubKey := comps.PubKeys[data.ComputorIndex]

	digest, err := getDigestFromTickData(data)
	if err != nil {
		return errors.Wrap(err, "getting partial tick data digest")
	}

	// verify tick signature
	err = utils.FourQSigVerify(ctx, computorPubKey, digest, data.Signature)
	if err != nil {
		return errors.Wrap(err, "verifying tick signature")
	}

	fullDigest, err := getFullDigestFromTickData(data)
	if err != nil {
		return errors.Wrap(err, "getting full tick data digest")
	}

	if fullDigest != quorumTickData.TxDigest {
		return errors.Wrapf(err, "quorum tx digest mismatch. full digest: %s. quorum tx digest: %s", hex.EncodeToString(fullDigest[:]), hex.EncodeToString(quorumTickData.TxDigest[:]))
	}

	return nil
}

func getDigestFromTickData(data types.TickData) ([32]byte, error) {
	// xor computor index with 8
	data.ComputorIndex ^= 8

	sData, err := utils.BinarySerialize(data)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "serializing data")
	}

	tickData := sData[:len(sData)-64]
	digest, err := utils.K12Hash(tickData)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "hashing tick data")
	}

	return digest, nil
}

func getFullDigestFromTickData(data types.TickData) ([32]byte, error) {
	// xor computor index with 8
	data.ComputorIndex ^= 8

	sData, err := utils.BinarySerialize(data)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "serializing data")
	}

	tickData := sData[:]
	digest, err := utils.K12Hash(tickData)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "hashing tick data")
	}

	return digest, nil
}
