package validator

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/cockroachdb/pebble"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/validator/chain"
	"github.com/qubic/go-archiver/validator/computors"
	"github.com/qubic/go-archiver/validator/quorum"
	"github.com/qubic/go-archiver/validator/tick"
	"github.com/qubic/go-archiver/validator/tx"
	"github.com/qubic/go-archiver/validator/txstatus"
	qubic "github.com/qubic/go-node-connector/v2"
	"github.com/qubic/go-node-connector/v2/types"
	"github.com/qubic/go-schnorrq"
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
		return fmt.Errorf("getting quorum tick data: %w", err)
	}

	if len(quorumVotes) == 0 {
		return errors.New("no quorum votes fetched")
	}

	//getting computors from storage, otherwise get it from a node
	epoch := quorumVotes[0].Epoch
	var comps types.Computors
	comps, err = computors.Get(ctx, v.store, uint32(epoch))
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("getting computors from store: %w", err)
		}

		comps, err = v.qu.GetComputors(ctx)
		if err != nil {
			return fmt.Errorf("getting computors from qubic: %w", err)
		}
	}

	err = computors.Validate(ctx, GoSchnorrqVerify, comps, v.arbitratorPubKey)
	if err != nil {
		return fmt.Errorf("validating comps: %w", err)
	}
	err = computors.Store(ctx, v.store, epoch, comps)
	if err != nil {
		return fmt.Errorf("storing computors: %w", err)
	}

	targetTickVoteSignature, err := v.store.GetTargetTickVoteSignature(uint32(epoch))
	if err != nil {
		if !errors.Is(err, store.ErrNotFound) {
			return fmt.Errorf("getting target tick vote signature: %w", err)
		}

		systemInfo, err := v.qu.GetSystemInfo(ctx)
		if err != nil {
			return fmt.Errorf("fetching system info: %w", err)
		}
		err = v.store.SetTargetTickVoteSignature(uint32(systemInfo.Epoch), systemInfo.TargetTickVoteSignature)
		if err != nil {
			return fmt.Errorf("storing target tick vote signature: %w", err)
		}
		targetTickVoteSignature = systemInfo.TargetTickVoteSignature
	}

	alignedVotes, err := quorum.Validate(ctx, GoSchnorrqVerify, quorumVotes, comps, targetTickVoteSignature)
	if err != nil {
		return fmt.Errorf("validating quorum: %w", err)
	}

	log.Printf("Quorum validated. Aligned %d. Misaligned %d.\n", len(alignedVotes), len(quorumVotes)-len(alignedVotes))

	var tickData types.TickData
	var validTxs = make([]types.Transaction, 0)
	approvedTxs := &protobuff.TickTransactionsStatus{}

	if quorumVotes[0].TxDigest != [32]byte{} {
		td, err := v.qu.GetTickData(ctx, tickNumber)
		if err != nil {
			return fmt.Errorf("getting tick data: %w", err)
		}

		tickData = td
		log.Println("Got tick data")

		err = tick.Validate(ctx, GoSchnorrqVerify, tickData, alignedVotes[0], comps)
		if err != nil {
			return fmt.Errorf("validating tick data: %w", err)
		}

		log.Println("Tick data validated")

		transactions, err := v.qu.GetTickTransactions(ctx, tickNumber)
		if err != nil {
			return fmt.Errorf("getting tick transactions: %w", err)
		}

		log.Printf("Validating %d transactions\n", len(transactions))

		validTxs, err = tx.Validate(ctx, GoSchnorrqVerify, transactions, tickData)
		if err != nil {
			return fmt.Errorf("validating transactions: %w", err)
		}

		log.Printf("Validated %d transactions\n", len(validTxs))

		var tickTxStatus types.TransactionStatus

		if disableStatusAddon {
			tickTxStatus = types.TransactionStatus{
				CurrentTickOfNode:  tickNumber,
				Tick:               tickNumber,
				TxCount:            uint32(len(validTxs)),
				MoneyFlew:          [(types.NumberOfTransactionsPerTick + 7) / 8]byte{},
				TransactionDigests: nil,
			}
		} else {
			tickTxStatus, err = v.qu.GetTxStatus(ctx, tickNumber)
			if err != nil {
				return fmt.Errorf("getting tx status: %w", err)
			}
		}

		approvedTxs, err = txstatus.Validate(ctx, tickTxStatus, validTxs)
		if err != nil {
			return fmt.Errorf("validating tx status: %w", err)
		}
	}

	// proceed to storing tick information
	err = quorum.Store(ctx, v.store, tickNumber, alignedVotes)
	if err != nil {
		return fmt.Errorf("storing quorum votes: %w", err)
	}

	log.Printf("Stored %d quorum votes\n", len(alignedVotes))

	err = tick.Store(ctx, v.store, tickNumber, tickData)
	if err != nil {
		return fmt.Errorf("storing tick data: %w", err)
	}

	log.Printf("Stored tick data\n")

	err = tx.Store(ctx, v.store, tickNumber, validTxs)
	if err != nil {
		return fmt.Errorf("storing transactions: %w", err)
	}

	log.Printf("Stored %d transactions\n", len(validTxs))

	err = txstatus.Store(ctx, v.store, tickNumber, approvedTxs)
	if err != nil {
		return fmt.Errorf("storing tx status: %w", err)
	}

	err = chain.ComputeAndSave(ctx, v.store, initialEpochTick, tickNumber, alignedVotes[0])
	if err != nil {
		return fmt.Errorf("computing and saving chain digest: %w", err)
	}

	err = chain.ComputeStoreAndSave(ctx, v.store, initialEpochTick, tickNumber, validTxs, approvedTxs)
	if err != nil {
		return fmt.Errorf("computing and saving store digest: %w", err)
	}

	isEmpty, err := tick.CheckIfTickIsEmpty(tickData)
	if err != nil {
		return fmt.Errorf("checking if tick is empty: %w", err)
	}

	if isEmpty {
		emptyTicks, err := v.store.GetEmptyTicksForEpoch(uint32(epoch))
		if err != nil {
			if !errors.Is(err, pebble.ErrNotFound) {
				return fmt.Errorf("getting empty ticks for current epoch: %w", err)
			}
		}

		if emptyTicks == 0 {
			fmt.Printf("Initializing empty ticks for epoch: %d\n", epoch)
			err := v.store.SetEmptyTickListPerEpoch(uint32(epoch), make([]uint32, 0))
			if err != nil {
				return fmt.Errorf("initializing empty tick list for epoch %d: %w", epoch, err)
			}
		}

		emptyTicks += 1

		err = v.store.SetEmptyTicksForEpoch(uint32(epoch), emptyTicks)
		if err != nil {
			return fmt.Errorf("setting current ticks for current epoch: %w", err)
		}
		fmt.Printf("Empty ticks for epoch %d: %d\n", epoch, emptyTicks)

		err = v.store.AppendEmptyTickToEmptyTickListPerEpoch(uint32(epoch), tickNumber)
		if err != nil {
			return fmt.Errorf("appending tick to empty tick list: %w", err)
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
		return nil, fmt.Errorf("creating new request: %w", err)
	}

	httpRes, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending request: %w", err)
	}
	defer httpRes.Body.Close()

	if httpRes.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("got non 200 status code: %d", httpRes.StatusCode)
	}

	var res responseStatusStruct

	err = json.NewDecoder(httpRes.Body).Decode(&res)
	if err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	return res, nil
}

func (v *Validator) GetTxStatus(ctx context.Context, tickNumber uint64) (*protobuff.TickTransactionsStatus, error) {
	qliServicesTransactions, err := v.queryQliServicesForTransactions(ctx, tickNumber)
	if err != nil {
		return nil, fmt.Errorf("querying qli services for transactions: %w", err)
	}

	transactions := make([]*protobuff.TransactionStatus, 0, len(qliServicesTransactions))
	for _, transaction := range qliServicesTransactions {
		var id types.Identity

		decoded, err := base64.StdEncoding.DecodeString(transaction.Digest)
		if err != nil {
			return nil, fmt.Errorf("base64 decoding digest: %w", err)
		}
		var pubKey [32]byte
		copy(pubKey[:], decoded[:32])
		id, err = id.FromPubKey(pubKey, true)
		if err != nil {
			return nil, fmt.Errorf("creating identity from public key: %w", err)
		}
		t := protobuff.TransactionStatus{
			TxId:      id.String(),
			MoneyFlew: transaction.MoneyFlew,
		}
		transactions = append(transactions, &t)
	}

	return &protobuff.TickTransactionsStatus{Transactions: transactions}, nil
}
