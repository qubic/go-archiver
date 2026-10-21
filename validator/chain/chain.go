package chain

import (
	"context"
	"fmt"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-node-connector/v2/types"
)

func ComputeAndSave(ctx context.Context, store *store.PebbleStore, initialEpochTick, tickNumber uint32, quorumVote types.QuorumTickVote) error {
	prevDigest, err := getPrevChainDigest(ctx, store, initialEpochTick, tickNumber)
	if err != nil {
		return fmt.Errorf("getting prev chain digest: %w", err)
	}

	currentDigest, err := computeCurrentTickDigest(ctx, quorumVote, prevDigest)
	if err != nil {
		return fmt.Errorf("computing current tick digest: %w", err)
	}

	err = store.PutChainDigest(ctx, tickNumber, currentDigest[:])
	if err != nil {
		return fmt.Errorf("storing chain digest for tick: %d\n: %w", tickNumber, err)
	}

	return nil
}

func ComputeStoreAndSave(ctx context.Context, store *store.PebbleStore, initialEpochTick, tickNumber uint32, validTxs []types.Transaction, tickTxsStatus *protobuff.TickTransactionsStatus) error {
	if tickNumber < 13752150 {
		return nil
	}

	prevDigest, err := getPrevStoreDigest(ctx, store, initialEpochTick, tickNumber)
	if err != nil {
		return fmt.Errorf("getting prev chain digest: %w", err)
	}

	currentDigest, err := computeCurrentTickStoreDigest(ctx, validTxs, tickTxsStatus, prevDigest)
	if err != nil {
		return fmt.Errorf("computing current tick digest: %w", err)
	}

	err = store.PutStoreDigest(ctx, tickNumber, currentDigest[:])
	if err != nil {
		return fmt.Errorf("storing store digest for tick: %d\n: %w", tickNumber, err)
	}

	return nil
}

func getPrevStoreDigest(ctx context.Context, store *store.PebbleStore, initialEpochTick, tickNumber uint32) ([32]byte, error) {
	if tickNumber == initialEpochTick {
		return [32]byte{}, nil
	}

	previousTickStoreDigestStored, err := store.GetStoreDigest(ctx, tickNumber-1)
	if err != nil {
		//returning nil error in order to not fail until epoch change
		return [32]byte{}, nil
		//return [32]byte{}, fmt.Errorf("getting chain digest for last tick: %d\n: %w", tickNumber-1, err)
	}

	var previousTickStoreDigest [32]byte
	copy(previousTickStoreDigest[:], previousTickStoreDigestStored)

	return previousTickStoreDigest, nil
}

func getPrevChainDigest(ctx context.Context, store *store.PebbleStore, initialEpochTick, tickNumber uint32) ([32]byte, error) {
	// if this is the first tick, there is no previous chain digest, so we are using an empty one
	if tickNumber == initialEpochTick {
		return [32]byte{}, nil
	}

	previousTickChainDigestStored, err := store.GetChainDigest(ctx, tickNumber-1)
	if err != nil {
		return [32]byte{}, fmt.Errorf("getting chain digest for last tick: %d\n: %w", tickNumber-1, err)
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
		PreviousTransactionBodyDigest: vote.PreviousTransactionBodyDigest,
		PreviousSpectrumDigest:        vote.PreviousSpectrumDigest,
		PreviousUniverseDigest:        vote.PreviousUniverseDigest,
		PreviousComputerDigest:        vote.PreviousComputerDigest,
		TxDigest:                      vote.TxDigest,
		PreviousTickChainDigest:       previousTickChainDigest,
	}

	digest, err := chain.Digest()
	if err != nil {
		return [32]byte{}, fmt.Errorf("computing chain digest: %w", err)
	}
	return digest, nil
}

func computeCurrentTickStoreDigest(ctx context.Context, validTxs []types.Transaction, tickTxsStatus *protobuff.TickTransactionsStatus, previousTickChainDigest [32]byte) ([32]byte, error) {
	s := Store{
		PreviousTickStoreDigest: previousTickChainDigest,
		ValidTxs:                validTxs,
		TickTxsStatus:           tickTxsStatus,
	}

	digest, err := s.Digest()
	if err != nil {
		return [32]byte{}, fmt.Errorf("computing store digest: %w", err)
	}

	return digest, nil
}
