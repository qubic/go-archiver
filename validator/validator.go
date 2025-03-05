package validator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/validator/chain"
	"github.com/qubic/go-archiver/validator/computors"
	"github.com/qubic/go-archiver/validator/quorum"
	"github.com/qubic/go-archiver/validator/tick"
	"github.com/qubic/go-archiver/validator/tx"
	"github.com/qubic/go-archiver/validator/txstatus"
	qubic "github.com/qubic/go-node-connector"
	"github.com/qubic/go-node-connector/types"
	"github.com/qubic/go-schnorrq"
	"log"
	"net/http"
	"strconv"
	"time"
)

type Validator struct {
	qu               *qubic.Client
	store            *store.PebbleStore
	arbitratorPubKey [32]byte
}

func New(qu *qubic.Client, store *store.PebbleStore, arbitratorPubKey [32]byte) *Validator {
	return &Validator{
		qu:               qu,
		store:            store,
		arbitratorPubKey: arbitratorPubKey,
	}
}

func GoSchnorrqVerify(_ context.Context, pubkey [32]byte, digest [32]byte, sig [64]byte) error {
	return schnorrq.Verify(pubkey, digest, sig)
}

func (v *Validator) ValidateTick(ctx context.Context, initialEpochTick, tickNumber uint32, disableStatusAddon bool) error {
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

	err = computors.Validate(ctx, GoSchnorrqVerify, comps, v.arbitratorPubKey)
	if err != nil {
		return errors.Wrap(err, "validating comps")
	}
	err = computors.Store(ctx, v.store, epoch, comps)
	if err != nil {
		return errors.Wrap(err, "storing computors")
	}

	alignedVotes, err := quorum.Validate(ctx, GoSchnorrqVerify, quorumVotes, comps)
	if err != nil {
		return errors.Wrap(err, "validating quorum")
	}

	// if the quorum votes have an empty tick data, it means that POTENTIALLY there is no tick data, it doesn't for
	// validation, but we may need to fetch it in the future ?!
	//if quorumVotes[0].TxDigest == [32]byte{} {
	//	return nil
	//}

	log.Printf("Quorum validated. Aligned %d. Misaligned %d.\n", len(alignedVotes), len(quorumVotes)-len(alignedVotes))

	tickData, err := v.qu.GetTickData(ctx, tickNumber)
	if err != nil {
		return errors.Wrap(err, "getting tick data")
	}
	log.Println("Got tick data")

	err = tick.Validate(ctx, GoSchnorrqVerify, tickData, alignedVotes[0], comps)
	if err != nil {
		return errors.Wrap(err, "validating tick data")
	}

	log.Println("Tick data validated")

	transactions, err := v.qu.GetTickTransactions(ctx, tickNumber)
	if err != nil {
		return errors.Wrap(err, "getting tick transactions")
	}

	log.Printf("Validating %d transactions\n", len(transactions))

	validTxs, err := tx.Validate(ctx, GoSchnorrqVerify, transactions, tickData)
	if err != nil {
		return errors.Wrap(err, "validating transactions")
	}

	log.Printf("Validated %d transactions\n", len(validTxs))

	var tickTxStatus types.TransactionStatus

	if disableStatusAddon {
		tickTxStatus = types.TransactionStatus{
			CurrentTickOfNode:  tickNumber,
			Tick:               tickNumber,
			TxCount:            uint32(len(validTxs)),
			MoneyFlew:          [128]byte{},
			TransactionDigests: nil,
		}
	} else {
		tickTxStatus, err = v.qu.GetTxStatus(ctx, tickNumber)
		if err != nil {
			return errors.Wrap(err, "getting tx status")
		}
	}

	approvedTxs, err := txstatus.Validate(ctx, tickTxStatus, validTxs)
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

	err = txstatus.Store(ctx, v.store, tickNumber, approvedTxs)
	if err != nil {
		return errors.Wrap(err, "storing tx status")
	}

	err = chain.ComputeAndSave(ctx, v.store, initialEpochTick, tickNumber, alignedVotes[0])
	if err != nil {
		return errors.Wrap(err, "computing and saving chain digest")
	}

	err = chain.ComputeStoreAndSave(ctx, v.store, initialEpochTick, tickNumber, validTxs, approvedTxs)
	if err != nil {
		return errors.Wrap(err, "computing and saving store digest")
	}

	isEmpty, err := tick.CheckIfTickIsEmpty(tickData)
	if err != nil {
		return errors.Wrap(err, "checking if tick is empty")
	}

	if isEmpty {
		emptyTicks, err := v.store.GetEmptyTicksForEpoch(uint32(epoch))
		if err != nil {
			if !errors.Is(err, pebble.ErrNotFound) {
				return errors.Wrap(err, "getting empty ticks for current epoch")
			}
		}

		if emptyTicks == 0 {
			fmt.Printf("Initializing empty ticks for epoch: %d\n", epoch)
			err := v.store.SetEmptyTickListPerEpoch(uint32(epoch), make([]uint32, 0))
			if err != nil {
				return errors.Wrapf(err, "initializing empty tick list for epoch %d", epoch)
			}
		}

		emptyTicks += 1

		err = v.store.SetEmptyTicksForEpoch(uint32(epoch), emptyTicks)
		if err != nil {
			return errors.Wrap(err, "setting current ticks for current epoch")
		}
		fmt.Printf("Empty ticks for epoch %d: %d\n", epoch, emptyTicks)

		err = v.store.AppendEmptyTickToEmptyTickListPerEpoch(uint32(epoch), tickNumber)
		if err != nil {
			return errors.Wrap(err, "appending tick to empty tick list")
		}
	}
	return nil
}

type responseStatusStruct []struct {
	Digest    string `json:"digest"`
	MoneyFlew bool   `json:"moneyFlew"`
}

func (v *Validator) queryQliServicesForTransactions(ctx context.Context, tickNumber uint64) (responseStatusStruct, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.qubic.li/Public/TickTransaction/"+strconv.Itoa(int(tickNumber)), nil)
	if err != nil {
		return nil, errors.Wrap(err, "creating new request")
	}

	httpRes, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "sending request")
	}
	defer httpRes.Body.Close()

	if httpRes.StatusCode != http.StatusOK {
		return nil, errors.Errorf("got non 200 status code: %d", httpRes.StatusCode)
	}

	var res responseStatusStruct

	err = json.NewDecoder(httpRes.Body).Decode(&res)
	if err != nil {
		return nil, errors.Wrap(err, "decoding response")
	}

	return res, nil
}

func (v *Validator) GetTxStatus(ctx context.Context, tickNumber uint64) (*protobuff.TickTransactionsStatus, error) {
	qliServicesTransactions, err := v.queryQliServicesForTransactions(ctx, tickNumber)
	if err != nil {
		return nil, errors.Wrap(err, "querying qli services for transactions")
	}

	transactions := make([]*protobuff.TransactionStatus, 0, len(qliServicesTransactions))
	for _, transaction := range qliServicesTransactions {
		var id types.Identity

		decoded, err := base64.StdEncoding.DecodeString(transaction.Digest)
		if err != nil {
			return nil, errors.Wrap(err, "base64 decoding digest")
		}
		var pubKey [32]byte
		copy(pubKey[:], decoded[:32])
		id, err = id.FromPubKey(pubKey, true)
		if err != nil {
			return nil, errors.Wrap(err, "creating identity from public key")
		}
		t := protobuff.TransactionStatus{
			TxId:      id.String(),
			MoneyFlew: transaction.MoneyFlew,
		}
		transactions = append(transactions, &t)
	}

	return &protobuff.TickTransactionsStatus{Transactions: transactions}, nil
}
