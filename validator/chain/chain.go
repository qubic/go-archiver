package chain

import (
	"context"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-node-connector/types"
)

func ComputeAndStore(ctx context.Context, store *store.PebbleStore, initialEpochTick, tickNumber uint32, quorumVote types.QuorumTickVote) error {
	prevDigest, err := getPrevChainDigest(ctx, store, initialEpochTick, tickNumber)
	if err != nil {
		return errors.Wrap(err, "getting prev chain digest")
	}

	currentDigest, err := computeCurrentTickDigest(ctx, quorumVote, prevDigest)
	if err != nil {
		return errors.Wrap(err, "computing current tick digest")
	}

	err = store.PutChainDigest(ctx, tickNumber, currentDigest[:])
	if err != nil {
		return errors.Wrapf(err, "storing chain digest for tick: %d\n", tickNumber)
	}

	return nil
}

func getPrevChainDigest(ctx context.Context, store *store.PebbleStore, initialEpochTick, tickNumber uint32) ([32]byte, error) {
	// if this is the first tick, there is no previous chain digest, so we are using an empty one
	if tickNumber == initialEpochTick {
		return [32]byte{}, nil
	}

	previousTickChainDigestStored, err := store.GetChainDigest(ctx, tickNumber-1)
	if err != nil {
		//returning nil error in order to not fail until epoch change
		return [32]byte{}, nil
		//return [32]byte{}, errors.Wrapf(err, "getting chain digest for last tick: %d\n", tickNumber-1)
	}

	var previousTickChainDigest [32]byte
	copy(previousTickChainDigest[:], previousTickChainDigestStored)

	return previousTickChainDigest, nil
}

func computeCurrentTickDigest(ctx context.Context, vote types.QuorumTickVote, previousTickChainDigest [32]byte) ([32]byte, error) {
	chain := Chain{
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
		PreviousTickChainDigest:       previousTickChainDigest,
	}

	digest, err := chain.Digest()
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "computing chain digest")
	}
	return digest, nil
}
