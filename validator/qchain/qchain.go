package qchain

import (
	"context"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-node-connector/types"
)

func ComputeAndStore(ctx context.Context, store *store.PebbleStore, tickNumber uint64, quorumVote types.QuorumTickVote) error {
	lastQChainDigest, err := store.GetQChainDigest(ctx, tickNumber-1)
	if err != nil {
		return errors.Wrapf(err, "getting qChain digest for last tick: %d\n", tickNumber-1)
	}
	currentDigest, err := computeCurrentTickDigest(ctx, quorumVote, lastQChainDigest)
	if err != nil {
		return errors.Wrap(err, "computing current tick digest")
	}

	err = store.PutQChainDigest(ctx, tickNumber, currentDigest[:])
	if err != nil {
		return errors.Wrapf(err, "storing qChain digest for tick: %d\n", tickNumber)
	}

	return nil
}

func computeCurrentTickDigest(ctx context.Context, vote types.QuorumTickVote, lastQChainDigest []byte) ([32]byte, error) {
	qChain := QChain{
		ComputorIndex:                 vote.ComputorIndex,
		Epoch:                         vote.Epoch,
		Tick:                          vote.Tick,
		Millisecond:                   vote.Millisecond,
		Second:                        vote.Second,
		Minute:                        vote.Minute,
		Hour:                          vote.Hour,
		Day:                           vote.Day,
		Month:                         vote.Month,
		Year:                          vote.Year,
		PreviousResourceTestingDigest: vote.PreviousResourceTestingDigest,
		PreviousSpectrumDigest:        vote.PreviousSpectrumDigest,
		PreviousUniverseDigest:        vote.PreviousUniverseDigest,
		PreviousComputerDigest:        vote.PreviousComputerDigest,
		TxDigest:                      vote.TxDigest,
	}
	copy(qChain.PreviousTickQChainDigest[:], lastQChainDigest[:])

	digest, err := qChain.Digest()
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "computing qChain digest")
	}
	return digest, nil
}
