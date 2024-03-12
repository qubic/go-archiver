package store

import (
	"context"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"os"
	"path/filepath"
	"testing"

	"github.com/cockroachdb/pebble"
	"go.uber.org/zap"

	pb "github.com/qubic/go-archiver/protobuff" // Update the import path to your generated protobuf package
)

func TestPebbleStore_TickData(t *testing.T) {
	ctx := context.Background()
	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	require.NoError(t, err)
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	require.NoError(t, err)
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
		require.NoError(t, err, "Failed to store TickData")
	}

	// Retrieve and verify each TickData record
	for _, tdOriginal := range tickDatas {
		retrievedTickData, err := store.GetTickData(ctx, uint64(tdOriginal.TickNumber))
		require.NoError(t, err, "Failed to retrieve TickData")
		ok := proto.Equal(tdOriginal, retrievedTickData)
		require.Equal(t, true, ok, "Retrieved TickData does not match original")
	}

	// Test error handling for non-existent TickData
	_, err = store.GetTickData(ctx, 999) // Assuming 999 is a tick number that wasn't stored
	require.Error(t, err, "Expected an error for non-existent TickData")
	require.Equal(t, ErrNotFound, err, "Expected ErrNotFound for non-existent TickData")
}

func TestPebbleStore_QuorumTickData(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	require.NoError(t, err)
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	require.NoError(t, err)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	store := NewPebbleStore(db, logger)

	// Sample QuorumTickData for testing
	quorumData := &pb.QuorumTickData{
		QuorumTickStructure: &pb.QuorumTickStructure{
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
	require.NoError(t, err)

	// Get QuorumTickData
	retrievedData, err := store.GetQuorumTickData(ctx, uint64(quorumData.QuorumTickStructure.TickNumber))
	require.NoError(t, err)

	if diff := cmp.Diff(quorumData, retrievedData, cmpopts.IgnoreUnexported(pb.QuorumTickData{}, pb.QuorumTickStructure{}, pb.QuorumDiff{})); diff != "" {
		t.Fatalf("Unexpected result: %v", diff)
	}

	// Test retrieval of non-existent QuorumTickData
	_, err = store.GetQuorumTickData(ctx, 999) // Assuming 999 is a tick number that wasn't stored
	require.Error(t, err)
	require.Equal(t, ErrNotFound, err)
}

func TestPebbleStore_Computors(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	require.NoError(t, err)
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	require.NoError(t, err)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	store := NewPebbleStore(db, logger)

	// Sample Computors for testing
	epoch := uint32(1) // Convert epoch to uint32 as per method requirements
	computors := &pb.Computors{
		Epoch:        1,
		Identities:   []string{"identity1", "identity2"},
		SignatureHex: "signature",
	}

	// Set Computors
	err = store.SetComputors(ctx, epoch, computors)
	require.NoError(t, err)

	// Get Computors
	retrievedComputors, err := store.GetComputors(ctx, epoch)
	require.NoError(t, err)

	// Validate retrieved data
	require.NotNil(t, retrievedComputors)
	require.Equal(t, computors.Epoch, retrievedComputors.Epoch)
	require.ElementsMatch(t, computors.Identities, retrievedComputors.Identities)
	require.Equal(t, computors.SignatureHex, retrievedComputors.SignatureHex)

	// Test retrieval of non-existent Computors
	_, err = store.GetComputors(ctx, 999) // Assuming 999 is an epoch number that wasn't stored
	require.Error(t, err)
	require.Equal(t, ErrNotFound, err)
}

func TestPebbleStore_TickTransactions(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	require.NoError(t, err)
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	require.NoError(t, err)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	store := NewPebbleStore(db, logger)

	transactions := &pb.Transactions{
		Transactions: []*pb.Transaction{
			{
				SourceId:     "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
				DestId:       "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
				Amount:       100,
				TickNumber:   101,
				InputType:    1,
				InputSize:    256,
				InputHex:     "input1",
				SignatureHex: "signature1",
				TxId:         "ff01",
			},
			{
				SourceId:     "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
				DestId:       "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
				Amount:       100,
				TickNumber:   101,
				InputType:    1,
				InputSize:    256,
				InputHex:     "input1",
				SignatureHex: "signature2",
				TxId:         "cd01",
			},
			// Add more transactions as needed
		},
	}

	tickData := pb.TickData{
		ComputorIndex:  1,
		Epoch:          1,
		TickNumber:     12795005,
		Timestamp:      1596240001,
		SignatureHex:   "signature1",
		TransactionIds: []string{"ff01", "cd01"},
	}
	err = store.SetTickData(ctx, uint64(tickData.TickNumber), &tickData)
	require.NoError(t, err, "Failed to store TickData")

	// Sample Transactions for testing
	tickNumber := uint64(12795005)

	// Assuming SetTransactions stores transactions for a tick
	err = store.SetTransactions(ctx, transactions)
	require.NoError(t, err)

	// GetTickTransactions retrieves stored transactions for a tick
	retrievedTransactions, err := store.GetTickTransactions(ctx, tickNumber)
	require.NoError(t, err)
	require.NotNil(t, retrievedTransactions)

	// Validate the retrieved transactions
	require.Len(t, retrievedTransactions.Transactions, len(transactions.Transactions))
	for i, tx := range transactions.Transactions {
		retrievedTx := retrievedTransactions.Transactions[i]
		require.Equal(t, tx.SourceId, retrievedTx.SourceId)
		// Continue with other fields...
	}

	// Test retrieval for a non-existent tick
	_, err = store.GetTickTransactions(ctx, 999) // Assuming 999 is a tick number that wasn't stored
	require.Error(t, err)
	require.Equal(t, ErrNotFound, err)
}

func TestPebbleStore_GetTransaction(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	require.NoError(t, err)
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	require.NoError(t, err)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	store := NewPebbleStore(db, logger)

	// Insert transactions for a tick
	targetTransaction := &pb.Transaction{
		SourceId:     "source_target",
		DestId:       "dest_target",
		Amount:       500,
		InputType:    2,
		InputSize:    512,
		InputHex:     "input_target",
		SignatureHex: "signature_target",
		TxId:         "cd01",
	}
	transactions := &pb.Transactions{
		Transactions: []*pb.Transaction{
			targetTransaction,
			// Additional transactions as needed
		},
	}

	// Use SetTransactions to store the transactions
	err = store.SetTransactions(ctx, transactions)
	require.NoError(t, err)

	retrievedTransaction, err := store.GetTransaction(ctx, targetTransaction.TxId)
	require.NoError(t, err)
	require.NotNil(t, retrievedTransaction)

	// Validate the retrieved transaction
	require.Equal(t, targetTransaction.SourceId, retrievedTransaction.SourceId)
	require.Equal(t, targetTransaction.DestId, retrievedTransaction.DestId)
	require.Equal(t, targetTransaction.Amount, retrievedTransaction.Amount)
	require.Equal(t, targetTransaction.InputType, retrievedTransaction.InputType)
	require.Equal(t, targetTransaction.InputSize, retrievedTransaction.InputSize)
	require.Equal(t, targetTransaction.InputHex, retrievedTransaction.InputHex)
	require.Equal(t, targetTransaction.SignatureHex, retrievedTransaction.SignatureHex)

	// Optionally, test retrieval of a non-existent transaction
	_, err = store.GetTransaction(ctx, "00")
	require.Error(t, err)
	require.Equal(t, ErrNotFound, err)
}

func TestSetAndGetLastProcessedTicksPerEpoch(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	require.NoError(t, err)
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	require.NoError(t, err)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	store := NewPebbleStore(db, logger)

	// Set last processed ticks per epoch
	err = store.SetLastProcessedTick(ctx, 16, 1)
	require.NoError(t, err)

	// Get last processed tick per epoch
	lastProcessedTick, err := store.GetLastProcessedTick(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(16), lastProcessedTick)

	lastProcessedTicksPerEpoch, err := store.GetLastProcessedTicksPerEpoch(ctx)
	require.NoError(t, err)
	require.Equal(t, map[uint32]uint64{1: 16}, lastProcessedTicksPerEpoch)

	// Set last processed ticks per epoch
	err = store.SetLastProcessedTick(ctx, 17, 1)
	require.NoError(t, err)

	// Get last processed tick per epoch
	lastProcessedTick, err = store.GetLastProcessedTick(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(17), lastProcessedTick)

	lastProcessedTicksPerEpoch, err = store.GetLastProcessedTicksPerEpoch(ctx)
	require.NoError(t, err)
	require.Equal(t, map[uint32]uint64{1: 17}, lastProcessedTicksPerEpoch)

	// Set last processed ticks per epoch
	err = store.SetLastProcessedTick(ctx, 18, 2)
	require.NoError(t, err)

	// Get last processed tick per epoch
	lastProcessedTick, err = store.GetLastProcessedTick(ctx)
	require.NoError(t, err)
	require.Equal(t, uint64(18), lastProcessedTick)

	lastProcessedTicksPerEpoch, err = store.GetLastProcessedTicksPerEpoch(ctx)
	require.NoError(t, err)
	require.Equal(t, map[uint32]uint64{1: 17, 2: 18}, lastProcessedTicksPerEpoch)
}

func TestGetSetSkippedTicks(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	require.NoError(t, err)
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	require.NoError(t, err)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	store := NewPebbleStore(db, logger)

	skippedTicksIntervalOne := pb.SkippedTicksInterval{
		StartTick: 15,
		EndTick:   20,
	}
	// Set skipped ticks
	err = store.SetSkippedTicksInterval(ctx, &skippedTicksIntervalOne)
	require.NoError(t, err)

	expected := pb.SkippedTicksIntervalList{SkippedTicks: []*pb.SkippedTicksInterval{&skippedTicksIntervalOne}}
	// Get skipped ticks
	got, err := store.GetSkippedTicksInterval(ctx)
	require.NoError(t, err)
	if diff := cmp.Diff(&expected, got, cmpopts.IgnoreUnexported(pb.SkippedTicksInterval{}, pb.SkippedTicksIntervalList{})); diff != "" {
		t.Fatalf("Unexpected result: %v", diff)
	}

	skippedTicksIntervalTwo := pb.SkippedTicksInterval{
		StartTick: 25,
		EndTick:   30,
	}

	err = store.SetSkippedTicksInterval(ctx, &skippedTicksIntervalTwo)
	require.NoError(t, err)

	expected.SkippedTicks = append(expected.SkippedTicks, &skippedTicksIntervalTwo)
	got, err = store.GetSkippedTicksInterval(ctx)
	require.NoError(t, err)
	if diff := cmp.Diff(&expected, got, cmpopts.IgnoreUnexported(pb.SkippedTicksInterval{}, pb.SkippedTicksIntervalList{})); diff != "" {
		t.Fatalf("Unexpected result: %v", diff)
	}
}

func TestPebbleStore_TransferTransactions(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	require.NoError(t, err)
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	require.NoError(t, err)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	store := NewPebbleStore(db, logger)

	idOne := "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB"
	idTwo := "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB"
	idThree := "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMXB"

	// Sample Transactions for testing
	forTickOne := pb.TransferTransactionsPerTick{
		TickNumber: 12,
		Identity:   idOne,
		Transactions: &pb.Transactions{
			Transactions: []*pb.Transaction{
				{
					SourceId:     "aaaaa",
					DestId:       "bbbbb",
					Amount:       15,
					TickNumber:   12,
					InputType:    87,
					InputSize:    122,
					InputHex:     "dddd",
					SignatureHex: "ffff",
					TxId:         "eeee",
				},
				{
					SourceId:     "bbbbb",
					DestId:       "aaaaa",
					Amount:       25,
					TickNumber:   12,
					InputType:    65,
					InputSize:    24,
					InputHex:     "ffff",
					SignatureHex: "dddd",
					TxId:         "cccc",
				},
			},
		},
	}

	forTickTwo := pb.TransferTransactionsPerTick{
		TickNumber: 15,
		Identity:   idTwo,
		Transactions: &pb.Transactions{
			Transactions: []*pb.Transaction{
				{
					SourceId:     "aaaaa",
					DestId:       "bbbbb",
					Amount:       15,
					TickNumber:   15,
					InputType:    87,
					InputSize:    122,
					InputHex:     "dddd",
					SignatureHex: "ffff",
					TxId:         "eeee",
				},
				{
					SourceId:     "bbbbb",
					DestId:       "aaaaa",
					Amount:       25,
					TickNumber:   15,
					InputType:    65,
					InputSize:    24,
					InputHex:     "ffff",
					SignatureHex: "dddd",
					TxId:         "cccc",
				},
			},
		},
	}

	err = store.PutTransferTransactionsPerTick(ctx, idOne, 12, &forTickOne)
	require.NoError(t, err)

	err = store.PutTransferTransactionsPerTick(ctx, idOne, 13, &forTickOne)
	require.NoError(t, err)

	got, err := store.GetTransferTransactions(ctx, idOne, 12, 12)
	require.NoError(t, err)
	diff := cmp.Diff([]*pb.TransferTransactionsPerTick{&forTickOne}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}, pb.Transactions{}))
	require.Equal(t, "", diff, "comparing first TransferTransactionsPerTick for idOne, forTickOne")

	got, err = store.GetTransferTransactions(ctx, idOne, 13, 13)
	require.NoError(t, err)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{&forTickOne}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}, pb.Transactions{}))
	require.Equal(t, "", diff, "comparing second TransferTransactionsPerTick for idOne, forTickOne")

	err = store.PutTransferTransactionsPerTick(ctx, idTwo, 15, &forTickTwo)
	require.NoError(t, err)
	got, err = store.GetTransferTransactions(ctx, idTwo, 15, 15)
	require.NoError(t, err)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{&forTickTwo}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}, pb.Transactions{}))
	require.Equal(t, "", diff, "comparing TransferTransactionsPerTick for idTwo, forTickTwo")
	
	perIdentityTx, err := store.GetTransferTransactions(ctx, idOne, 12, 13)
	require.NoError(t, err)
	require.Equal(t, 2, len(perIdentityTx))

	expected := []*pb.TransferTransactionsPerTick{&forTickOne, &forTickOne}
	require.NoError(t, err)
	diff = cmp.Diff(expected, perIdentityTx, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}, pb.Transactions{}))
	require.Equal(t, "", diff, "comparing perIdentityTx")

	// not existing identity means no transfers
	perIdentityTx, err = store.GetTransferTransactions(ctx, idThree, 1, 20)
	require.NoError(t, err)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{}, perIdentityTx, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}, pb.Transactions{}))
	require.Equal(t, "", diff, "comparison of perIdentityTx for idThree")

	// not existing tick means no transfers
	perTickTx, err := store.GetTransferTransactions(ctx, idOne, 14, 14)
	require.NoError(t, err)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{}, perTickTx, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}, pb.Transactions{}))
	require.Equal(t, "", diff, "comparison of perTickTx for idOne and tick 14")
}

func TestPebbleStore_QChainHash(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	require.NoError(t, err)
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	require.NoError(t, err)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	store := NewPebbleStore(db, logger)

	// Sample QChainHash for testing
	tickNumber := uint64(12795005)
	qChainHash := []byte("qChainHash")

	// Set QChainHash
	err = store.PutQChainDigest(ctx, tickNumber, qChainHash)
	require.NoError(t, err)

	// Get QChainHash
	retrievedQChainHash, err := store.GetQChainDigest(ctx, tickNumber)
	require.NoError(t, err)
	require.Equal(t, qChainHash, retrievedQChainHash)

	// Test retrieval of non-existent QChainHash
	_, err = store.GetQChainDigest(ctx, 999) // Assuming 999 is a tick number that wasn't stored
	require.Error(t, err)
	require.Equal(t, ErrNotFound, err)
}
