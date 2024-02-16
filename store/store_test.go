package store

import (
	"context"
	"encoding/hex"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/pebble"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"

	pb "github.com/qubic/go-archiver/protobuff" // Update the import path to your generated protobuf package
)

func TestPebbleStore_TickData(t *testing.T) {
	ctx := context.Background()
	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	assert.NoError(t, err)
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	assert.NoError(t, err)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	store := NewPebbleStore(db, logger)

	// Create multiple TickData records
	tickDatas := []*pb.TickData{
		{
			ComputorIndex: 1,
			Epoch:         1,
			TickNumber:    101,
			Timestamp:     1596240001,
			SignatureHex:  "signature1",
		},
		{
			ComputorIndex: 2,
			Epoch:         2,
			TickNumber:    102,
			Timestamp:     1596240002,
			SignatureHex:  "signature2",
		},
		{
			ComputorIndex: 3,
			Epoch:         3,
			TickNumber:    103,
			Timestamp:     1596240003,
			SignatureHex:  "signature3",
		},
	}

	// Insert TickData records
	for _, td := range tickDatas {
		err = store.SetTickData(ctx, uint64(td.TickNumber), td)
		assert.NoError(t, err, "Failed to store TickData")
	}

	// Retrieve and verify each TickData record
	for _, tdOriginal := range tickDatas {
		retrievedTickData, err := store.GetTickData(ctx, uint64(tdOriginal.TickNumber))
		assert.NoError(t, err, "Failed to retrieve TickData")
		ok := proto.Equal(tdOriginal, retrievedTickData)
		assert.Equal(t, true, ok, "Retrieved TickData does not match original")
	}

	// Test error handling for non-existent TickData
	_, err = store.GetTickData(ctx, 999) // Assuming 999 is a tick number that wasn't stored
	assert.Error(t, err, "Expected an error for non-existent TickData")
	assert.Equal(t, ErrNotFound, err, "Expected ErrNotFound for non-existent TickData")
}

func TestPebbleStore_QuorumTickData(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	assert.NoError(t, err)
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	assert.NoError(t, err)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	store := NewPebbleStore(db, logger)

	// Sample QuorumTickData for testing
	quorumData := &pb.QuorumTickData{
		QuorumTickStructure: &pb.QuorumTickStructure{
			ComputorIndex:         1,
			Epoch:                 1,
			TickNumber:            101,
			Timestamp:             1596240001,
			PrevSpectrumDigestHex: "prevSpectrumDigest",
			PrevUniverseDigestHex: "prevUniverseDigest",
			PrevComputerDigestHex: "prevComputerDigest",
			TxDigestHex:           "txDigest",
		},
		QuorumDiffPerComputor: map[uint32]*pb.QuorumDiff{
			0: {
				SaltedSpectrumDigestHex:     "saltedSpectrumDigest",
				SaltedUniverseDigestHex:     "saltedUniverseDigest",
				SaltedComputerDigestHex:     "saltedComputerDigest",
				ExpectedNextTickTxDigestHex: "expectedNextTickTxDigest",
				SignatureHex:                "signature",
			},
			1: {
				SaltedSpectrumDigestHex:     "saltedSpectrumDigest",
				SaltedUniverseDigestHex:     "saltedUniverseDigest",
				SaltedComputerDigestHex:     "saltedComputerDigest",
				ExpectedNextTickTxDigestHex: "expectedNextTickTxDigest",
				SignatureHex:                "signature",
			},
		},
	}

	// Set QuorumTickData
	err = store.SetQuorumTickData(ctx, uint64(quorumData.QuorumTickStructure.TickNumber), quorumData)
	assert.NoError(t, err)

	// Get QuorumTickData
	retrievedData, err := store.GetQuorumTickData(ctx, uint64(quorumData.QuorumTickStructure.TickNumber))
	assert.NoError(t, err)

	if diff := cmp.Diff(quorumData, retrievedData, cmpopts.IgnoreUnexported(pb.QuorumTickData{}, pb.QuorumTickStructure{}, pb.QuorumDiff{})); diff != "" {
		t.Fatalf("Unexpected result: %v", diff)
	}

	// Test retrieval of non-existent QuorumTickData
	_, err = store.GetQuorumTickData(ctx, 999) // Assuming 999 is a tick number that wasn't stored
	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
}

func TestPebbleStore_Computors(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	assert.NoError(t, err)
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	assert.NoError(t, err)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	store := NewPebbleStore(db, logger)

	// Sample Computors for testing
	epoch := uint64(1) // Convert epoch to uint64 as per method requirements
	computors := &pb.Computors{
		Epoch:        1,
		Identities:   []string{"identity1", "identity2"},
		SignatureHex: "signature",
	}

	// Set Computors
	err = store.SetComputors(ctx, epoch, computors)
	assert.NoError(t, err)

	// Get Computors
	retrievedComputors, err := store.GetComputors(ctx, epoch)
	assert.NoError(t, err)

	// Validate retrieved data
	assert.NotNil(t, retrievedComputors)
	assert.Equal(t, computors.Epoch, retrievedComputors.Epoch)
	assert.ElementsMatch(t, computors.Identities, retrievedComputors.Identities)
	assert.Equal(t, computors.SignatureHex, retrievedComputors.SignatureHex)

	// Test retrieval of non-existent Computors
	_, err = store.GetComputors(ctx, 999) // Assuming 999 is an epoch number that wasn't stored
	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
}

func TestPebbleStore_TickTransactions(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	assert.NoError(t, err)
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	assert.NoError(t, err)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	store := NewPebbleStore(db, logger)

	transactions := &pb.Transactions{
		Transactions: []*pb.Transaction{
			{
				SourcePubkeyHex: "source1",
				DestPubkeyHex:   "dest1",
				Amount:          100,
				TickNumber:      101,
				InputType:       1,
				InputSize:       256,
				InputHex:        "input1",
				SignatureHex:    "signature1",
				DigestHex: "ff01",
			},
			{
				SourcePubkeyHex: "source1",
				DestPubkeyHex:   "dest1",
				Amount:          100,
				TickNumber:      101,
				InputType:       1,
				InputSize:       256,
				InputHex:        "input1",
				SignatureHex:    "signature2",
				DigestHex: "cd01",
			},
			// Add more transactions as needed
		},
	}

	tickData := pb.TickData{
		ComputorIndex:         1,
		Epoch:                 1,
		TickNumber:            101,
		Timestamp:             1596240001,
		SignatureHex:          "signature1",
		TransactionDigestsHex: []string{"ff01", "cd01"},
	}
	err = store.SetTickData(ctx, uint64(tickData.TickNumber), &tickData)
	assert.NoError(t, err, "Failed to store TickData")

	// Sample Transactions for testing
	tickNumber := uint64(101)

	// Assuming SetTickTransactions stores transactions for a tick
	err = store.SetTickTransactions(ctx, transactions)
	assert.NoError(t, err)

	// GetTickTransactions retrieves stored transactions for a tick
	retrievedTransactions, err := store.GetTickTransactions(ctx, tickNumber)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedTransactions)

	// Validate the retrieved transactions
	assert.Len(t, retrievedTransactions.Transactions, len(transactions.Transactions))
	for i, tx := range transactions.Transactions {
		retrievedTx := retrievedTransactions.Transactions[i]
		assert.Equal(t, tx.SourcePubkeyHex, retrievedTx.SourcePubkeyHex)
		// Continue with other fields...
	}

	// Test retrieval for a non-existent tick
	_, err = store.GetTickTransactions(ctx, 999) // Assuming 999 is a tick number that wasn't stored
	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
}

func TestPebbleStore_GetTransaction(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	assert.NoError(t, err)
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	assert.NoError(t, err)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	store := NewPebbleStore(db, logger)

	// Insert transactions for a tick
	targetTransaction := &pb.Transaction{
		SourcePubkeyHex: "source_target",
		DestPubkeyHex:   "dest_target",
		Amount:          500,
		InputType:       2,
		InputSize:       512,
		InputHex:        "input_target",
		SignatureHex:    "signature_target",
		DigestHex: "cd01",
	}
	transactions := &pb.Transactions{
		Transactions: []*pb.Transaction{
			targetTransaction,
			// Additional transactions as needed
		},
	}

	// Use SetTickTransactions to store the transactions
	err = store.SetTickTransactions(ctx, transactions)
	assert.NoError(t, err)

	// Attempt to retrieve the target transaction
	digest, err := hex.DecodeString(targetTransaction.DigestHex)
	assert.NoError(t, err)
	retrievedTransaction, err := store.GetTransaction(ctx, digest)
	assert.NoError(t, err)
	assert.NotNil(t, retrievedTransaction)

	// Validate the retrieved transaction
	assert.Equal(t, targetTransaction.SourcePubkeyHex, retrievedTransaction.SourcePubkeyHex)
	assert.Equal(t, targetTransaction.DestPubkeyHex, retrievedTransaction.DestPubkeyHex)
	assert.Equal(t, targetTransaction.Amount, retrievedTransaction.Amount)
	assert.Equal(t, targetTransaction.InputType, retrievedTransaction.InputType)
	assert.Equal(t, targetTransaction.InputSize, retrievedTransaction.InputSize)
	assert.Equal(t, targetTransaction.InputHex, retrievedTransaction.InputHex)
	assert.Equal(t, targetTransaction.SignatureHex, retrievedTransaction.SignatureHex)

	digest[0] = 0x00
	// Optionally, test retrieval of a non-existent transaction
	_, err = store.GetTransaction(ctx, digest)
	assert.Error(t, err)
	assert.Equal(t, ErrNotFound, err)
}
