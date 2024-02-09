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
	"log"
)

type Validator struct {
	qu *qubic.Client
	store *store.PebbleStore
}

func NewValidator(qu *qubic.Client, store *store.PebbleStore) *Validator {
	return &Validator{qu: qu, store: store}
}

func (v *Validator) ValidateTick(ctx context.Context, tickNumber uint64) error {
	comps, err := v.qu.GetComputors(ctx)
	if err != nil {
		return errors.Wrap(err, "getting comps")
	}

	err = computors.Validate(ctx, comps.Computors)
	if err != nil {
		return errors.Wrap(err, "validating comps")
	}
	log.Println("Computors validated")

	quorumTickData, err := v.qu.GetQuorumTickData(ctx, uint32(tickNumber))
	if err != nil {
		return errors.Wrap(err, "getting quorum tick data")
	}

	err = quorum.Validate(ctx, quorumTickData, comps.Computors)
	if err != nil {
		return errors.Wrap(err, "validating quorum")
	}

	log.Println("Quorum validated")

	tickData, err := v.qu.GetTickData(ctx, uint32(tickNumber))
	if err != nil {
		return errors.Wrap(err, "getting tick data")
	}

	err = tick.Validate(ctx, tickData, quorumTickData.QuorumData[0], comps.Computors)
	if err != nil {
		return errors.Wrap(err, "validating tick data")
	}

	log.Println("Tick validated")

	transactions, err := v.qu.GetTickTransactions(ctx, uint32(tickNumber))
	if err != nil {
		return errors.Wrap(err, "getting tick transactions")
	}

	err = tx.Validate(ctx, transactions, tickData)
	if err != nil {
		return errors.Wrap(err, "validating transactions")
	}

	return nil
}
