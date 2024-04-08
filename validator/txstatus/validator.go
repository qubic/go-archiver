package txstatus

import (
	"bytes"
	"context"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-node-connector/types"
	"sort"
)

func Validate(ctx context.Context, tickTxStatus types.TransactionStatus, tickTxs types.Transactions) error {
	if tickTxStatus.TxCount != uint32(len(tickTxs)) {
		return errors.Errorf("Mismatched tx length. Tick tx status count: %d - len(tickTx): %d", tickTxStatus.TxCount, len(tickTxs))
	}

	tickTxDigests, err := getTickTxDigests(tickTxs)
	if err != nil {
		return errors.Wrap(err, "getting tick tx digests")
	}

	if !equalDigests(tickTxDigests, tickTxStatus.TransactionDigests) {
		return errors.New("digests not equal")
	}

	return nil
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

func equalDigests(tickTxsDigests [][32]byte, tickTxStatusDigests [][32]byte) bool {
	// Sort the slices
	sortByteSlices(tickTxsDigests)
	sortByteSlices(tickTxStatusDigests)

	// Compare the sorted slices element by element
	for i := range tickTxsDigests {
		if !bytes.Equal(tickTxsDigests[i][:], tickTxStatusDigests[i][:]) {
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

func Store(ctx context.Context, store *store.PebbleStore, tickNumber uint32, tickTxStatus types.TransactionStatus) error {
	proto, err := qubicToProto(tickTxStatus)
	if err != nil {
		return errors.Wrap(err, "qubic to proto")
	}

	err = store.SetTickTransactionsStatus(ctx, uint64(tickNumber), proto)
	if err != nil {
		return errors.Wrap(err, "setting tts")
	}

	return nil
}
