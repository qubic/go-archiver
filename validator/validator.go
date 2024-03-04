package validator

import (
	"context"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/validator/computors"
	"github.com/qubic/go-archiver/validator/quorum"
	"github.com/qubic/go-archiver/validator/tick"
	"github.com/qubic/go-archiver/validator/tx"
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

func (v *Validator) ValidateTick(ctx context.Context, tickNumber uint64) error {
	quorumVotes, err := v.qu.GetQuorumVotes(ctx, uint32(tickNumber))
	if err != nil {
		return errors.Wrap(err, "getting quorum tick data")
	}

	if len(quorumVotes) == 0 {
		return errors.New("no quorum votes fetched")
	}

	//getting computors from storage, otherwise get it from a node
	epoch := quorumVotes[0].Epoch
	var comps types.Computors
	comps, err = computors.Get(ctx, v.store, uint64(epoch))
	if err != nil {
		if errors.Cause(err) != store.ErrNotFound {
			return errors.Wrap(err, "getting computors from store")
		}

		comps, err = v.qu.GetComputors(ctx)
		if err != nil {
			return errors.Wrap(err, "getting computors from qubic")
		}
	}

	err = computors.Validate(ctx, comps)
	if err != nil {
		return errors.Wrap(err, "validating comps")
	}
	err = computors.Store(ctx, v.store, comps)
	if err != nil {
		return errors.Wrap(err, "storing computors")
	}

	err = quorum.Validate(ctx, quorumVotes, comps)
	if err != nil {
		return errors.Wrap(err, "validating quorum")
	}

	// if the quorum votes have an empty tick data, it means that POTENTIALLY there is no tick data, it doesn't for
	// validation, but we may need to fetch it in the future ?!
	if quorumVotes[0].TxDigest == [32]byte{} {
		return nil
	}

	log.Println("Quorum validated")

	tickData, err := v.qu.GetTickData(ctx, uint32(tickNumber))
	if err != nil {
		return errors.Wrap(err, "getting tick data")
	}
	log.Println("Got tick data")

	err = tick.Validate(ctx, tickData, quorumVotes[0], comps)
	if err != nil {
		return errors.Wrap(err, "validating tick data")
	}

	log.Println("Tick data validated")

	transactions, err := v.qu.GetTickTransactions(ctx, uint32(tickNumber))
	if err != nil {
		return errors.Wrap(err, "getting tick transactions")
	}

	log.Printf("Validating %d transactions\n", len(transactions))

	validTxs, err := tx.Validate(ctx, transactions, tickData)
	if err != nil {
		return errors.Wrap(err, "validating transactions")
	}

	log.Printf("Validated %d transactions\n", len(validTxs))

	// proceed to storing tick information
	err = quorum.Store(ctx, v.store, quorumVotes)
	if err != nil {
		return errors.Wrap(err, "storing quorum votes")
	}

	log.Printf("Stored %d quorum votes\n", len(quorumVotes))

	err = tick.Store(ctx, v.store, tickData)
	if err != nil {
		return errors.Wrap(err, "storing tick data")
	}

	log.Printf("Stored tick data\n")

	err = tx.Store(ctx, v.store, validTxs)
	if err != nil {
		return errors.Wrap(err, "storing transactions")
	}

	log.Printf("Stored %d transactions\n", len(transactions))

	return nil
}
