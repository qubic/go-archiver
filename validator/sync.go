package validator

import (
	"context"
	"github.com/pingcap/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/validator/quorum"
	"github.com/qubic/go-archiver/validator/tick"
	"github.com/qubic/go-archiver/validator/tx"
	"github.com/qubic/go-node-connector/types"
	"log"
	"time"
)

type ValidatedTick struct {
	FirstVote            types.QuorumTickVote
	AlignedVotes         *protobuff.QuorumTickDataStored
	TickData             types.TickData
	ValidTransactions    []types.Transaction
	ApprovedTransactions *protobuff.TickTransactionsStatus
}

type ValidatedTicks []ValidatedTick

type SyncValidator struct {
	tickNumber        uint32
	epoch             uint32
	computors         types.Computors
	quorumData        *protobuff.QuorumTickData
	tickData          *protobuff.TickData
	transactions      []*protobuff.Transaction
	transactionStatus []*protobuff.TransactionStatus

	pebbleStore        *store.PebbleStore
	processTickTimeout time.Duration
	initialEpochTick   uint32
}

func NewSyncValidator(initialEpochTick uint32, computors types.Computors, syncTickData *protobuff.SyncTickData, processTickTimeout time.Duration, pebbleStore *store.PebbleStore) *SyncValidator {
	return &SyncValidator{
		tickNumber:        syncTickData.QuorumData.QuorumTickStructure.TickNumber,
		epoch:             syncTickData.QuorumData.QuorumTickStructure.Epoch,
		computors:         computors,
		quorumData:        syncTickData.QuorumData,
		tickData:          syncTickData.TickData,
		transactions:      syncTickData.Transactions,
		transactionStatus: syncTickData.TransactionsStatus,

		pebbleStore:        pebbleStore,
		processTickTimeout: processTickTimeout,
		initialEpochTick:   initialEpochTick,
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
		return ValidatedTick{}, errors.Wrap(err, "converting tick data to qubic format")
	}

	err = tick.Validate(ctx, GoSchnorrqVerify, tickData, alignedVotes[0], sv.computors)
	if err != nil {
		return ValidatedTick{}, errors.Wrap(err, "validating tick data")
	}
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

	/*err = quorum.Store(ctx, sv.pebbleStore, sv.tickNumber, alignedVotes)
	if err != nil {
		return nil, errors.Wrap(err, "storing quorum votes")
	}
	log.Printf("Stored %d quorum votes\n", len(alignedVotes))*/

	/*err = tick.Store(ctx, sv.pebbleStore, sv.tickNumber, tickData)
	if err != nil {
		return nil, errors.Wrap(err, "storing tick data")
	}
	log.Printf("Stored tick data\n")*/

	/*err = tx.Store(ctx, sv.pebbleStore, sv.tickNumber, validTransactions)
	if err != nil {
		return nil, errors.Wrap(err, "storing transactions")
	}
	log.Printf("Stored %d transactions\n", len(transactions))*/

	approvedTransactions := &protobuff.TickTransactionsStatus{
		Transactions: sv.transactionStatus,
	}

	/*err = txstatus.Store(ctx, sv.pebbleStore, sv.tickNumber, approvedTransactions)
	if err != nil {
		return nil, errors.Wrap(err, "storing tx status")
	}*/

	/*err = chain.ComputeAndSave(ctx, sv.pebbleStore, sv.initialEpochTick, sv.tickNumber, alignedVotes[0])
	if err != nil {
		return nil, errors.Wrap(err, "computing and saving chain digest")
	}

	err = chain.ComputeStoreAndSave(ctx, sv.pebbleStore, sv.initialEpochTick, sv.tickNumber, validTransactions, approvedTransactions)
	if err != nil {
		return nil, errors.Wrap(err, "computing and saving store digest")
	}

	isEmpty, err := tick.CheckIfTickIsEmpty(tickData)
	if isEmpty {
		err = handleEmptyTick(sv.pebbleStore, sv.tickNumber, sv.epoch)
		if err != nil {
			return nil, errors.Wrap(err, "handling empty tick")
		}
	}*/

	return ValidatedTick{
		FirstVote:            alignedVotes[0],
		AlignedVotes:         quorum.QubicToProtoStored(alignedVotes),
		TickData:             tickData,
		ValidTransactions:    validTransactions,
		ApprovedTransactions: approvedTransactions,
	}, nil
}
