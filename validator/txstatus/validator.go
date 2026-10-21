package txstatus

import (
	"bytes"
	"context"
	"fmt"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-node-connector/v2/types"
	"sort"
)

func Validate(ctx context.Context, tickTxStatus types.TransactionStatus, tickTxs types.Transactions) (*protobuff.TickTransactionsStatus, error) {
	//if tickTxStatus.TxCount != uint32(len(tickTxs)) {
	//	return nil, fmt.Errorf("Mismatched tx length. Tick tx status count: %d - len(tickTx): %d", tickTxStatus.TxCount, len(tickTxs))
	//}

	//tickTxDigests, err := getTickTxDigests(tickTxs)
	//if err != nil {
	//	return nil, fmt.Errorf("getting tick tx digests: %w", err)
	//}

	//if !equalDigests(tickTxDigests, tickTxStatus.TransactionDigests) {
	//	return nil, errors.New("digests not equal")
	//}

	proto, err := qubicToProto(tickTxs, tickTxStatus)
	if err != nil {
		return nil, fmt.Errorf("qubic to proto: %w", err)
	}

	return proto, nil
}

func getTickTxDigests(tickTxs types.Transactions) ([][32]byte, error) {
	tickTxDigests := make([][32]byte, 0, len(tickTxs))
	for index, tx := range tickTxs {
		digest, err := tx.Digest()
		if err != nil {
			return nil, fmt.Errorf("creating digest for tx on index %d: %w", index, err)
		}

		tickTxDigests = append(tickTxDigests, digest)
	}

	return tickTxDigests, nil
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

func Store(ctx context.Context, store *store.PebbleStore, tickNumber uint32, approvedTxs *protobuff.TickTransactionsStatus) error {
	err := store.SetTickTransactionsStatus(ctx, uint64(tickNumber), approvedTxs)
	if err != nil {
		return fmt.Errorf("setting tts: %w", err)
	}

	return nil
}
