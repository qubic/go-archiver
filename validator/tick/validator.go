package tick

import (
	"context"
	"encoding/hex"
	"fmt"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-node-connector/types"
)

func Validate(ctx context.Context, sigVerifierFunc utils.SigVerifierFunc, data types.TickData, quorumTickVote types.QuorumTickVote, comps types.Computors) error {
	if data.Epoch == 0xffff {
		data.Epoch = 0
	}

	//empty tick with empty quorum tx digest means other verification is not needed
	if (data.IsEmpty()) && quorumTickVote.TxDigest == [32]byte{} {
		return nil
	}

	if data.Epoch == 0 {
		data.Epoch = quorumTickVote.Epoch
	}

	computorPubKey := comps.PubKeys[data.ComputorIndex]

	digest, err := getDigestFromTickData(data)
	if err != nil {
		return errors.Wrap(err, "getting partial tick data digest")
	}

	// verify tick signature
	err = sigVerifierFunc(ctx, computorPubKey, digest, data.Signature)
	if err != nil {
		return errors.Wrap(err, "verifying tick signature")
	}

	fullDigest, err := getFullDigestFromTickData(data)
	if err != nil {
		return errors.Wrap(err, "getting full tick data digest")
	}

	if fullDigest != quorumTickVote.TxDigest {
		return fmt.Errorf("quorum tx digest mismatch. full digest: %s. quorum tx digest: %s", hex.EncodeToString(fullDigest[:]), hex.EncodeToString(quorumTickVote.TxDigest[:]))
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

func Store(ctx context.Context, store *store.PebbleStore, tickNumber uint32, tickData types.TickData) error {
	protoTickData, err := qubicToProto(tickData)
	if err != nil {
		return errors.Wrap(err, "converting qubic tick data to proto")
	}

	err = store.SetTickData(ctx, tickNumber, protoTickData)
	if err != nil {
		return errors.Wrap(err, "set tick data")
	}
	return nil
}
