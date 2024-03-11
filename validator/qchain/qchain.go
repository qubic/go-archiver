package qchain

import (
	"context"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-node-connector/types"
)

func ComputeAndStore(ctx context.Context, store *store.PebbleStore, initialEpochTick, tickNumber uint64, quorumVote types.QuorumTickVote) error {
	prevDigest, err := getPrevQChainDigest(ctx, store, initialEpochTick, tickNumber)
	if err != nil {
		return errors.Wrap(err, "getting prev qChain digest")
	}

	currentDigest, err := computeCurrentTickDigest(ctx, quorumVote, prevDigest)
	if err != nil {
		return errors.Wrap(err, "computing current tick digest")
	}

	err = store.PutQChainDigest(ctx, tickNumber, currentDigest[:])
	if err != nil {
		return errors.Wrapf(err, "storing qChain digest for tick: %d\n", tickNumber)
	}

	return nil
}

func getPrevQChainDigest(ctx context.Context, store *store.PebbleStore, initialEpochTick, tickNumber uint64) ([32]byte, error) {
	// if this is the first tick, there is no previous qChain digest, so we are using an empty one
	if tickNumber == initialEpochTick {
		return [32]byte{}, nil
	}

	previousTickQChainDigestStored, err := store.GetQChainDigest(ctx, tickNumber-1)
	if err != nil {
		//returning nil error in order to not fail until epoch change
		return [32]byte{}, nil
		//return [32]byte{}, errors.Wrapf(err, "getting qChain digest for last tick: %d\n", tickNumber-1)
	}

	var previousTickQChainDigest [32]byte
	copy(previousTickQChainDigest[:], previousTickQChainDigestStored)

	return previousTickQChainDigest, nil
}

func computeCurrentTickDigest(ctx context.Context, vote types.QuorumTickVote, previousTickQChainDigest [32]byte) ([32]byte, error) {
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
		PreviousTickQChainDigest:      previousTickQChainDigest,
	}

	digest, err := qChain.Digest()
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "computing qChain digest")
	}
	return digest, nil
}
