package validator

import (
	"cmp"
	"github.com/pingcap/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/validator/chain"
	"github.com/qubic/go-archiver/validator/quorum"
	"github.com/qubic/go-archiver/validator/tick"
	"github.com/qubic/go-archiver/validator/tx"
	"github.com/qubic/go-node-connector/types"
	"log"
	"slices"
	"sync"
	"time"
)

type ValidatedTick struct {
	AlignedVotes           *protobuff.QuorumTickDataStored
	TickData               *protobuff.TickData
	ValidTransactions      []*protobuff.Transaction
	ApprovedTransactions   *protobuff.TickTransactionsStatus
	ChainHash              [32]byte
	StoreHash              [32]byte
	firstVote              types.QuorumTickVote
	validTransactionsQubic []types.Transaction
}

type SyncValidator struct {
	initialIntervalTick  uint32
	computors            types.Computors
	ticks                []*protobuff.SyncTickData
	lastSynchronizedTick *protobuff.SyncLastSynchronizedTick
	pebbleStore          *store.PebbleStore
	processTickTimeout   time.Duration
}

func NewSyncValidator(initialIntervalTick uint32, computors types.Computors, ticks []*protobuff.SyncTickData, pebbleStore *store.PebbleStore, lastSynchronizedTick *protobuff.SyncLastSynchronizedTick) *SyncValidator {

	return &SyncValidator{
		initialIntervalTick:  initialIntervalTick,
		computors:            computors,
		ticks:                ticks,
		lastSynchronizedTick: lastSynchronizedTick,
		pebbleStore:          pebbleStore,
	}
}

func (sv *SyncValidator) Validate(routineCount int) ([]*ValidatedTick, error) {

	var waitGroup sync.WaitGroup
	validatedTicksChannel := make(chan []*ValidatedTick, routineCount)
	errChannel := make(chan error, routineCount)

	batchSize := len(sv.ticks) / routineCount
	startTime := time.Now()

	for index := range routineCount {
		waitGroup.Add(1)

		start := batchSize * index
		end := start + batchSize
		if end > (len(sv.ticks)) || index == (routineCount-1) {
			end = len(sv.ticks)
		}
		tickRange := sv.ticks[start:end]

		go func(validatedTicksChannel chan<- []*ValidatedTick, errChanel chan<- error) {
			defer waitGroup.Done()

			log.Printf("[Routine %d] Validating tick range %d - %d\n", index, start, end)

			var validatedTicks []*ValidatedTick

			for _, syncTickData := range tickRange {

				validatedTick, err := sv.validateTick(syncTickData)
				if err != nil {
					validatedTicksChannel <- nil
					errChannel <- errors.Wrapf(err, "validating tick %d", syncTickData.QuorumData.QuorumTickStructure.TickNumber)
					return
				}
				validatedTicks = append(validatedTicks, validatedTick)
			}

			validatedTicksChannel <- validatedTicks
			errChannel <- nil

		}(validatedTicksChannel, errChannel)
	}

	waitGroup.Wait()

	firstTick := sv.ticks[0].QuorumData.QuorumTickStructure.TickNumber
	lastTick := sv.ticks[len(sv.ticks)-1].QuorumData.QuorumTickStructure.TickNumber

	log.Printf("Done validating rick range [%d - %d]. Took: %f\n", firstTick, lastTick, time.Since(startTime).Seconds())

	var totalValidatedTicks []*ValidatedTick

	for _ = range routineCount {
		validatedTicks := <-validatedTicksChannel
		err := <-errChannel
		if err != nil {
			return nil, errors.Wrap(err, "processing ticks concurrently")
		}
		totalValidatedTicks = append(totalValidatedTicks, validatedTicks...)
	}

	slices.SortFunc(totalValidatedTicks, func(a, b *ValidatedTick) int {
		// Compare against tick number from quorum data, as tick data may be nil
		return cmp.Compare(a.AlignedVotes.QuorumTickStructure.TickNumber, b.AlignedVotes.QuorumTickStructure.TickNumber)
	})

	log.Printf("Computing chain and store digests...\n")

	err := sv.computeDigests(totalValidatedTicks)
	if err != nil {
		return nil, errors.Wrap(err, "computing digests for validated ticks")
	}

	return totalValidatedTicks, nil
}

func (sv *SyncValidator) validateTick(syncTickData *protobuff.SyncTickData) (*ValidatedTick, error) {

	quorumVotes, err := quorum.ProtoToQubic(syncTickData.QuorumData)
	if err != nil {
		return nil, errors.Wrap(err, "converting quorum data to qubic format")
	}

	alignedVotes, err := quorum.Validate(nil, GoSchnorrqVerify, quorumVotes, sv.computors)
	if err != nil {
		return nil, errors.Wrap(err, "validating quorum")
	}

	tickData, err := tick.ProtoToQubic(syncTickData.TickData)
	if err != nil {
		return nil, errors.Wrapf(err, "converting tick data to qubic format")
	}

	if syncTickData.QuorumData.QuorumTickStructure.Epoch < 124 { // Mitigation for epochs that contain varStruct in the tick data

		fullTickData, err := tick.ProtoToQubicFull(syncTickData.TickData)
		if err != nil {
			return nil, errors.Wrap(err, "converting tick data to qubic format")
		}

		err = fullTickData.Validate(nil, GoSchnorrqVerify, alignedVotes[0], sv.computors)
		if err != nil {
			return nil, errors.Wrap(err, "validating full tick data")
		}
	} else {

		err := tick.Validate(nil, GoSchnorrqVerify, tickData, alignedVotes[0], sv.computors)
		if err != nil {
			return nil, errors.Wrap(err, "validating tick data")
		}
	}

	transactions, err := tx.ProtoToQubic(syncTickData.Transactions)
	if err != nil {
		return nil, errors.Wrap(err, "converting transactions to qubic format")
	}

	validTransactions, err := tx.Validate(nil, GoSchnorrqVerify, transactions, tickData)
	if err != nil {
		return nil, errors.Wrap(err, "validating transactions")
	}

	transactionsProto, err := tx.QubicToProto(validTransactions)
	if err != nil {
		return nil, errors.Wrap(err, "converting transactions to proto format")
	}

	approvedTransactions := &protobuff.TickTransactionsStatus{
		Transactions: syncTickData.TransactionsStatus,
	}

	validatedTick := ValidatedTick{
		AlignedVotes:         quorum.QubicToProtoStored(alignedVotes),
		TickData:             syncTickData.TickData,
		ValidTransactions:    transactionsProto,
		ApprovedTransactions: approvedTransactions,

		firstVote:              alignedVotes[0],
		validTransactionsQubic: validTransactions,
	}

	return &validatedTick, nil
}

func (sv *SyncValidator) computeDigests(validatedTicks []*ValidatedTick) error {

	// Digests of previous validated tick are required to compute the digests of current tick
	var lastChainHash [32]byte
	var lastStoreHash [32]byte
	if sv.initialIntervalTick <= sv.lastSynchronizedTick.TickNumber {
		copy(lastChainHash[:], sv.lastSynchronizedTick.ChainHash)
		copy(lastStoreHash[:], sv.lastSynchronizedTick.StoreHash)
	}

	for _, validatedTick := range validatedTicks {

		if sv.lastSynchronizedTick.TickNumber == validatedTick.AlignedVotes.QuorumTickStructure.TickNumber {
			continue
		}

		chainHash, err := chain.ComputeCurrentTickDigest(nil, validatedTick.firstVote, lastChainHash)
		if err != nil {
			return errors.Wrapf(err, "calculating chain digest for tick %d", validatedTick.AlignedVotes.QuorumTickStructure.TickNumber)

		}
		storeHash, err := chain.ComputeCurrentTickStoreDigest(nil, validatedTick.validTransactionsQubic, validatedTick.ApprovedTransactions, lastStoreHash)
		if err != nil {
			return errors.Wrapf(err, "calculating store digest for tich %d", validatedTick.AlignedVotes.QuorumTickStructure.TickNumber)
		}

		validatedTick.ChainHash = chainHash
		validatedTick.StoreHash = storeHash

		lastChainHash = chainHash
		lastStoreHash = storeHash

	}

	return nil
}
