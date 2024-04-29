package txstatus

import (
	"bytes"
	"context"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-node-connector/types"
	"sort"
)

func Validate(ctx context.Context, tickTxStatus []types.IndividualTransactionStatus, tickTxs types.Transactions) (*protobuff.TickTransactionsStatus, error) {
	if len(tickTxStatus) != len(tickTxs) {
		return nil, errors.Errorf("Mismatched tx length. Tick tx status count: %d - len(tickTx): %d", len(tickTxStatus), len(tickTxs))
	}

	tickTxDigests, err := getTickTxDigests(tickTxs)
	if err != nil {
		return nil, errors.Wrap(err, "getting tick tx digests")
	}

	tickTxStatusDigests, err := getTickTxStatusDigests(tickTxStatus)
	if err != nil {
		return nil, errors.Wrap(err, "getting tick tx status digests")
	}

	if !equalDigests(tickTxDigests, tickTxStatusDigests) {
		return nil, errors.New("digests not equal")
	}

	return qubicToProto(tickTxStatus), nil
}

func getTickTxDigests(tickTxs types.Transactions) ([][32]byte, error) {
	tickTxDigests := make([][32]byte, 0, len(tickTxs))
	for index, tx := range tickTxs {
		digest, err := tx.Digest()
		if err != nil {
			return nil, errors.Wrapf(err, "creating digest for tx on index %d", index)
		}

		tickTxDigests = append(tickTxDigests, digest)
	}

	return tickTxDigests, nil
}

func getTickTxStatusDigests(tickTxStatus []types.IndividualTransactionStatus) ([][32]byte, error) {
	tickTxStatusDigests := make([][32]byte, 0, len(tickTxStatus))
	for index, txStatus := range tickTxStatus {
		id := types.Identity(txStatus.TxID)
		pubKey, err := id.ToPubKey(true)
		if err != nil {
			return nil, errors.Wrapf(err, "creating digest for tx on index %d", index)
		}

		tickTxStatusDigests = append(tickTxStatusDigests, pubKey)
	}

	return tickTxStatusDigests, nil
}

func copySlice(slice [][32]byte) [][32]byte {
	newSlice := make([][32]byte, len(slice))
	copy(newSlice, slice)
	return newSlice
}

func equalDigests(tickTxsDigests [][32]byte, tickTxStatusDigests [][32]byte) bool {
	//copy slices to avoid modifying the original slices
	copyTxsDigests := copySlice(tickTxsDigests)
	copyTxStatusDigests := copySlice(tickTxStatusDigests)

	// Sort the slices
	sortByteSlices(copyTxsDigests)
	sortByteSlices(copyTxStatusDigests)

	// Compare the sorted slices element by element
	for i := range copyTxsDigests {
		if !bytes.Equal(copyTxsDigests[i][:], copyTxStatusDigests[i][:]) {
			return false
		}
	}

	return true
}

func sortByteSlices(slice [][32]byte) {
	sort.Slice(slice, func(i, j int) bool {
		return bytes.Compare(slice[i][:], slice[j][:]) == -1
	})
}

func Store(ctx context.Context, store *store.PebbleStore, tickNumber uint32, validTxsStatus *protobuff.TickTransactionsStatus) error {
	err := store.SetTickTransactionsStatus(ctx, uint64(tickNumber), validTxsStatus)
	if err != nil {
		return errors.Wrap(err, "setting tts")
	}

	return nil
}
