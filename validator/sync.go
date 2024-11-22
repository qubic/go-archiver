package validator

import (
	"cmp"
	"fmt"
	"github.com/pingcap/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/validator/chain"
	"github.com/qubic/go-archiver/validator/quorum"
	"github.com/qubic/go-archiver/validator/tick"
	"github.com/qubic/go-archiver/validator/tx"
	"github.com/qubic/go-node-connector/types"
	"log"
	"runtime"
	"slices"
	"sync"
	"time"
)

type ValidatedTick struct {
	AlignedVotes         *protobuff.QuorumTickDataStored
	TickData             *protobuff.TickData
	ValidTransactions    []*protobuff.Transaction
	ApprovedTransactions *protobuff.TickTransactionsStatus
	ChainHash            [32]byte
	StoreHash            [32]byte

	firstVote              types.QuorumTickVote
	validTransactionsQubic []types.Transaction
}

type ValidatedTicks []ValidatedTick

type SyncValidator struct {
	initialIntervalTick uint32

	computors types.Computors

	ticks []*protobuff.SyncTickData

	chainHash *[32]byte
	storeHash *[32]byte

	pebbleStore        *store.PebbleStore
	processTickTimeout time.Duration
}

func NewSyncValidator(initialIntervalTick uint32, computors types.Computors, ticks []*protobuff.SyncTickData, processTickTimeout time.Duration, pebbleStore *store.PebbleStore, chainHash, storeHash *[32]byte) *SyncValidator {

	return &SyncValidator{
		initialIntervalTick: initialIntervalTick,
		computors:           computors,
		ticks:               ticks,

		chainHash: chainHash,
		storeHash: storeHash,

		pebbleStore:        pebbleStore,
		processTickTimeout: processTickTimeout,
	}
}

func (sv *SyncValidator) Validate() (ValidatedTicks, error) {

	/*ctx, cancel := context.WithTimeout(context.Background(), sv.processTickTimeout)
	defer cancel()*/

	var validatedTicks ValidatedTicks
	counter := 0
	mutex := sync.RWMutex{}

	routineCount := runtime.NumCPU()
	batchSize := len(sv.ticks) / routineCount
	errChannel := make(chan error, routineCount)
	var waitGroup sync.WaitGroup
	startTime := time.Now()

	for index := range routineCount {
		waitGroup.Add(1)

		start := batchSize * index
		end := start + batchSize
		if end > (len(sv.ticks)) || index == (routineCount-1) {
			end = len(sv.ticks)
		}

		tickRange := sv.ticks[start:end]

		go func(errChanel chan<- error) {
			defer waitGroup.Done()
			log.Printf("[Routine %d]  Validating tick range %d - %d\n", index, start, end)

			for _, tickInfo := range tickRange {

				log.Printf("[Routine %d]  Validating tick %d \n", index, tickInfo.QuorumData.QuorumTickStructure.TickNumber)

				quorumVotes, err := quorum.ProtoToQubic(tickInfo.QuorumData)
				if err != nil {
					errChannel <- errors.Wrap(err, "converting quorum data to qubic format")
					return
				}

				alignedVotes, err := quorum.Validate(nil, GoSchnorrqVerify, quorumVotes, sv.computors)
				if err != nil {
					errChannel <- errors.Wrap(err, "validating quorum")
					return
				}

				log.Printf("Quorum validated. Aligned %d. Misaligned %d.\n", len(alignedVotes), len(quorumVotes)-len(alignedVotes))

				tickData, err := tick.ProtoToQubic(tickInfo.TickData)
				if err != nil {
					errChannel <- errors.Wrapf(err, "converting tick data to qubic format")
					return
				}

				err = tick.Validate(nil, GoSchnorrqVerify, tickData, alignedVotes[0], sv.computors)
				if err != nil {
					errChannel <- errors.Wrap(err, "validating tick data")
					return
				}

				/*if len(sv.tickData.VarStruct) != 0 {

					fullTickData, err := tick.ProtoToQubicFull(sv.tickData)
					if err != nil {
						return ValidatedTick{}, errors.Wrap(err, "converting tick data to qubic format")
					}

					err = fullTickData.Validate(ctx, GoSchnorrqVerify, alignedVotes[0], sv.computors)
					if err != nil {
						return ValidatedTick{}, errors.Wrap(err, "validating tick data")
					}
				} else {
					err := tick.Validate(ctx, GoSchnorrqVerify, tickData, alignedVotes[0], sv.computors)
					if err != nil {
						return ValidatedTick{}, errors.Wrap(err, "validating tick data")
					}
				}*/

				log.Println("Tick data validated")

				transactions, err := tx.ProtoToQubic(tickInfo.Transactions)
				if err != nil {
					errChannel <- errors.Wrap(err, "converting transactions to qubic format")
					return
				}

				log.Printf("Validating %d transactions\n", len(transactions))

				validTransactions, err := tx.Validate(nil, GoSchnorrqVerify, transactions, tickData)
				if err != nil {
					errChannel <- errors.Wrap(err, "validating transactions")
					return
				}
				log.Printf("Validated %d transactions\n", len(validTransactions))

				/*isEmpty, err := tick.CheckIfTickIsEmpty(tickData)
				if isEmpty {
					err = handleEmptyTick(sv.pebbleStore, sv.tickNumber, sv.epoch)
					if err != nil {
						return ValidatedTick{}, errors.Wrap(err, "handling empty tick")
					}
				}*/

				transactionsProto, err := tx.QubicToProto(validTransactions)
				if err != nil {
					errChannel <- errors.Wrap(err, "converting transactions to proto format")
					return
				}

				approvedTransactions := &protobuff.TickTransactionsStatus{
					Transactions: tickInfo.TransactionsStatus,
				}

				mutex.Lock()

				validatedTick := ValidatedTick{
					AlignedVotes:         quorum.QubicToProtoStored(alignedVotes),
					TickData:             tickInfo.TickData,
					ValidTransactions:    transactionsProto,
					ApprovedTransactions: approvedTransactions,

					firstVote:              alignedVotes[0],
					validTransactionsQubic: validTransactions,
				}

				validatedTicks = append(validatedTicks, validatedTick)
				counter += 1

				mutex.Unlock()
			}

			errChannel <- nil

		}(errChannel)
	}

	waitGroup.Wait()
	log.Printf("Done processing %d ticks. Took: %v\n", counter, time.Since(startTime))

	for _ = range routineCount {
		err := <-errChannel
		if err != nil {
			return nil, errors.Wrap(err, "processing ticks concurrently")
		}
	}

	slices.SortFunc(validatedTicks, func(a, b ValidatedTick) int {
		return cmp.Compare(a.AlignedVotes.QuorumTickStructure.TickNumber, b.AlignedVotes.QuorumTickStructure.TickNumber)
	})

	log.Printf("Computing chain and store digests...\n")

	for _, validatedTick := range validatedTicks {

		fmt.Printf("Computing for tick %d\r", validatedTick.AlignedVotes.QuorumTickStructure.TickNumber)

		chainHash, err := chain.ComputeCurrentTickDigest(nil, validatedTick.firstVote, *sv.chainHash)
		if err != nil {
			return nil, errors.Wrapf(err, "calculating chain digest for tick %d", validatedTick.AlignedVotes.QuorumTickStructure.TickNumber)

		}
		storeHash, err := chain.ComputeCurrentTickStoreDigest(nil, validatedTick.validTransactionsQubic, validatedTick.ApprovedTransactions, *sv.storeHash)
		if err != nil {
			return nil, errors.Wrapf(err, "calculating store digest for tich %d", validatedTick.AlignedVotes.QuorumTickStructure.TickNumber)
		}

		*sv.chainHash = chainHash
		*sv.storeHash = storeHash

		validatedTick.ChainHash = chainHash
		validatedTick.StoreHash = storeHash

	}

	return validatedTicks, nil
}
