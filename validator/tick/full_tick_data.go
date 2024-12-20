package tick

import (
	"context"
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-node-connector/types"
)

type FullTickData struct {
	ComputorIndex      uint16
	Epoch              uint16
	Tick               uint32
	Millisecond        uint16
	Second             uint8
	Minute             uint8
	Hour               uint8
	Day                uint8
	Month              uint8
	Year               uint8
	UnionData          [256]byte
	Timelock           [32]byte
	TransactionDigests [types.NumberOfTransactionsPerTick][32]byte `json:",omitempty"`
	ContractFees       [1024]int64                                 `json:",omitempty"`
	Signature          [types.SignatureSize]byte
}

func (ftd *FullTickData) IsEmpty() bool {
	if ftd == nil {
		return true
	}

	return *ftd == FullTickData{}
}

func (ftd *FullTickData) Validate(ctx context.Context, sigVerifierFunc utils.SigVerifierFunc, quorumTickVote types.QuorumTickVote, comps types.Computors) error {
	//empty tick with empty quorum tx digest means other verification is not needed
	if (ftd.IsEmpty()) && quorumTickVote.TxDigest == [32]byte{} {
		return nil
	}

	computorPubKey := comps.PubKeys[ftd.ComputorIndex]

	digest, err := ftd.getDigestFromTickData()
	if err != nil {
		return errors.Wrap(err, "getting partial tick data digest")
	}

	// verify tick signature
	err = sigVerifierFunc(ctx, computorPubKey, digest, ftd.Signature)
	if err != nil {
		return errors.Wrap(err, "verifying tick signature")
	}

	fullDigest, err := ftd.getFullDigestFromTickData()
	if err != nil {
		return errors.Wrap(err, "getting full tick data digest")
	}

	if fullDigest != quorumTickVote.TxDigest {
		return errors.Wrapf(err, "quorum tx digest mismatch. full digest: %s. quorum tx digest: %s", hex.EncodeToString(fullDigest[:]), hex.EncodeToString(quorumTickVote.TxDigest[:]))
	}

	return nil
}

func (ftd *FullTickData) getDigestFromTickData() ([32]byte, error) {
	// xor computor index with 8
	ftd.ComputorIndex ^= 8

	sData, err := utils.BinarySerialize(ftd)
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

func (ftd *FullTickData) getFullDigestFromTickData() ([32]byte, error) {
	// xor computor index with 8
	ftd.ComputorIndex ^= 8

	sData, err := utils.BinarySerialize(ftd)
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
