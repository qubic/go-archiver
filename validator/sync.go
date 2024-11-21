package validator

import (
	"context"
	"github.com/pingcap/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/validator/chain"
	"github.com/qubic/go-archiver/validator/quorum"
	"github.com/qubic/go-archiver/validator/tick"
	"github.com/qubic/go-archiver/validator/tx"
	"github.com/qubic/go-node-connector/types"
	"log"
	"time"
)

type ValidatedTick struct {
	AlignedVotes         *protobuff.QuorumTickDataStored
	TickData             *protobuff.TickData
	ValidTransactions    []*protobuff.Transaction
	ApprovedTransactions *protobuff.TickTransactionsStatus
	ChainHash            [32]byte
	StoreHash            [32]byte
}

type ValidatedTicks []ValidatedTick

type SyncValidator struct {
	tickNumber          uint32
	epoch               uint32
	initialIntervalTick uint32

	computors         types.Computors
	quorumData        *protobuff.QuorumTickData
	tickData          *protobuff.TickData
	transactions      []*protobuff.Transaction
	transactionStatus []*protobuff.TransactionStatus
	previousChainHash [32]byte
	previousStoreHash [32]byte

	pebbleStore        *store.PebbleStore
	processTickTimeout time.Duration
}

func NewSyncValidator(initialIntervalTick uint32, computors types.Computors, syncTickData *protobuff.SyncTickData,
	processTickTimeout time.Duration, pebbleStore *store.PebbleStore, previousChainHash, previousStoreHash [32]byte) *SyncValidator {
	return &SyncValidator{
		tickNumber:          syncTickData.QuorumData.QuorumTickStructure.TickNumber,
		epoch:               syncTickData.QuorumData.QuorumTickStructure.Epoch,
		initialIntervalTick: initialIntervalTick,

		computors:         computors,
		quorumData:        syncTickData.QuorumData,
		tickData:          syncTickData.TickData,
		transactions:      syncTickData.Transactions,
		transactionStatus: syncTickData.TransactionsStatus,
		previousChainHash: previousChainHash,
		previousStoreHash: previousStoreHash,

		pebbleStore:        pebbleStore,
		processTickTimeout: processTickTimeout,
	}
}

func (sv *SyncValidator) Validate() (ValidatedTick, error) {

	ctx, cancel := context.WithTimeout(context.Background(), sv.processTickTimeout)
	defer cancel()

	quorumVotes, err := quorum.ProtoToQubic(sv.quorumData)
	if err != nil {
		return ValidatedTick{}, errors.Wrap(err, "converting quorum data to qubic format")
	}

	alignedVotes, err := quorum.Validate(ctx, GoSchnorrqVerify, quorumVotes, sv.computors)
	if err != nil {
		return ValidatedTick{}, errors.Wrap(err, "validating quorum")
	}

	log.Printf("Quorum validated. Aligned %d. Misaligned %d.\n", len(alignedVotes), len(quorumVotes)-len(alignedVotes))

	tickData, err := tick.ProtoToQubic(sv.tickData)
	if err != nil {
		return ValidatedTick{}, errors.Wrapf(err, "converting tick data to qubic format")
	}

	err = tick.Validate(ctx, GoSchnorrqVerify, tickData, alignedVotes[0], sv.computors)
	if err != nil {
		return ValidatedTick{}, errors.Wrap(err, "validating tick data")
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

	transactions, err := tx.ProtoToQubic(sv.transactions)
	if err != nil {
		return ValidatedTick{}, errors.Wrap(err, "converting transactions to qubic format")
	}

	log.Printf("Validating %d transactions\n", len(transactions))

	validTransactions, err := tx.Validate(ctx, GoSchnorrqVerify, transactions, tickData)
	if err != nil {
		return ValidatedTick{}, errors.Wrap(err, "validating transactions")

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
		return ValidatedTick{}, errors.Wrap(err, "converting transactions to proto format")
	}

	approvedTransactions := &protobuff.TickTransactionsStatus{
		Transactions: sv.transactionStatus,
	}

	chainHash, err := chain.ComputeCurrentTickDigest(nil, alignedVotes[0], sv.previousChainHash)
	if err != nil {
		return ValidatedTick{}, errors.Wrap(err, "calculating chain digest")
	}
	storeHash, err := chain.ComputeCurrentTickStoreDigest(nil, validTransactions, approvedTransactions, sv.previousStoreHash)
	if err != nil {
		return ValidatedTick{}, errors.Wrap(err, "calculating store digest")
	}

	return ValidatedTick{
		AlignedVotes:         quorum.QubicToProtoStored(alignedVotes),
		TickData:             sv.tickData,
		ValidTransactions:    transactionsProto,
		ApprovedTransactions: approvedTransactions,
		ChainHash:            chainHash,
		StoreHash:            storeHash,
	}, nil
}
