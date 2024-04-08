package validator

import (
	"context"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-archiver/validator/chain"
	"github.com/qubic/go-archiver/validator/computors"
	"github.com/qubic/go-archiver/validator/quorum"
	"github.com/qubic/go-archiver/validator/tick"
	"github.com/qubic/go-archiver/validator/tx"
	"github.com/qubic/go-archiver/validator/txstatus"
	qubic "github.com/qubic/go-node-connector"
	"github.com/qubic/go-node-connector/types"
	"log"
)

type Validator struct {
	qu    *qubic.Client
	store *store.PebbleStore
}

func New(qu *qubic.Client, store *store.PebbleStore) *Validator {
	return &Validator{qu: qu, store: store}
}

func (v *Validator) ValidateTick(ctx context.Context, initialEpochTick, tickNumber uint32) error {
	quorumVotes, err := v.qu.GetQuorumVotes(ctx, tickNumber)
	if err != nil {
		return errors.Wrap(err, "getting quorum tick data")
	}

	if len(quorumVotes) == 0 {
		return errors.New("no quorum votes fetched")
	}

	//getting computors from storage, otherwise get it from a node
	epoch := quorumVotes[0].Epoch
	var comps types.Computors
	comps, err = computors.Get(ctx, v.store, uint32(epoch))
	if err != nil {
		if errors.Cause(err) != store.ErrNotFound {
			return errors.Wrap(err, "getting computors from store")
		}

		comps, err = v.qu.GetComputors(ctx)
		if err != nil {
			return errors.Wrap(err, "getting computors from qubic")
		}
	}

	err = computors.Validate(ctx, utils.FourQSigVerify, comps)
	if err != nil {
		return errors.Wrap(err, "validating comps")
	}
	err = computors.Store(ctx, v.store, epoch, comps)
	if err != nil {
		return errors.Wrap(err, "storing computors")
	}

	alignedVotes, err := quorum.Validate(ctx, utils.FourQSigVerify, quorumVotes, comps)
	if err != nil {
		return errors.Wrap(err, "validating quorum")
	}

	// if the quorum votes have an empty tick data, it means that POTENTIALLY there is no tick data, it doesn't for
	// validation, but we may need to fetch it in the future ?!
	//if quorumVotes[0].TxDigest == [32]byte{} {
	//	return nil
	//}

	log.Printf("Quorum validated. Aligned %d. Misaligned %d.\n", len(alignedVotes), len(quorumVotes)-len(alignedVotes))

	tickData, err := v.qu.GetTickData(ctx, uint32(tickNumber))
	if err != nil {
		return errors.Wrap(err, "getting tick data")
	}
	log.Println("Got tick data")

	err = tick.Validate(ctx, utils.FourQSigVerify, tickData, alignedVotes[0], comps)
	if err != nil {
		return errors.Wrap(err, "validating tick data")
	}

	log.Println("Tick data validated")

	transactions, err := v.qu.GetTickTransactions(ctx, tickNumber)
	if err != nil {
		return errors.Wrap(err, "getting tick transactions")
	}

	log.Printf("Validating %d transactions\n", len(transactions))

	validTxs, err := tx.Validate(ctx, utils.FourQSigVerify, transactions, tickData)
	if err != nil {
		return errors.Wrap(err, "validating transactions")
	}

	log.Printf("Validated %d transactions\n", len(validTxs))

	tickTxStatus, err := v.qu.GetTxStatus(ctx, tickNumber)
	if err != nil {
		return errors.Wrap(err, "getting tx status")
	}

	err = txstatus.Validate(ctx, tickTxStatus, validTxs)
	if err != nil {
		return errors.Wrap(err, "validating tx status")
	}

	// proceed to storing tick information
	err = quorum.Store(ctx, v.store, tickNumber, alignedVotes)
	if err != nil {
		return errors.Wrap(err, "storing quorum votes")
	}

	log.Printf("Stored %d quorum votes\n", len(alignedVotes))

	err = tick.Store(ctx, v.store, tickNumber, tickData)
	if err != nil {
		return errors.Wrap(err, "storing tick data")
	}

	log.Printf("Stored tick data\n")

	err = tx.Store(ctx, v.store, tickNumber, validTxs)
	if err != nil {
		return errors.Wrap(err, "storing transactions")
	}

	log.Printf("Stored %d transactions\n", len(transactions))

	err = txstatus.Store(ctx, v.store, tickNumber, tickTxStatus)
	if err != nil {
		return errors.Wrap(err, "storing tx status")
	}

	err = chain.ComputeAndStore(ctx, v.store, initialEpochTick, tickNumber, alignedVotes[0])
	if err != nil {
		return errors.Wrap(err, "computing and storing chain")
	}

	return nil
}
