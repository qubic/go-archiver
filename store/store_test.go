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

	tcs := []struct {
		name       string
		td         *pb.TickData
		exp        *pb.TickData
		tickNumber uint32
	}{
		{
			name:       "TestTickData_1",
			tickNumber: 101,
			td: &pb.TickData{
				ComputorIndex: 1,
				Epoch:         1,
				TickNumber:    101,
				Timestamp:     1596240001,
				SignatureHex:  "signature1",
			},
			exp: &pb.TickData{
				ComputorIndex: 1,
				Epoch:         1,
				TickNumber:    101,
				Timestamp:     1596240001,
				SignatureHex:  "signature1",
			},
		},
		{
			name:       "TestTickData_2",
			tickNumber: 102,
			td: &pb.TickData{
				ComputorIndex: 2,
				Epoch:         2,
				TickNumber:    102,
				Timestamp:     1596240002,
				SignatureHex:  "signature2",
			},
			exp: &pb.TickData{
				ComputorIndex: 2,
				Epoch:         2,
				TickNumber:    102,
				Timestamp:     1596240002,
				SignatureHex:  "signature2",
			},
		},
		{
			name:       "TestTickData_3",
			tickNumber: 103,
			td: &pb.TickData{
				ComputorIndex: 3,
				Epoch:         3,
				TickNumber:    103,
				Timestamp:     1596240003,
				SignatureHex:  "signature3",
			},
			exp: &pb.TickData{
				ComputorIndex: 3,
				Epoch:         3,
				TickNumber:    103,
				Timestamp:     1596240003,
				SignatureHex:  "signature3",
			},
		},
		{
			name:       "TestTickData_nil",
			tickNumber: 104,
			td:         nil,
			exp:        &pb.TickData{},
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			err := store.SetTickData(ctx, tc.tickNumber, tc.td)
			require.NoError(t, err)

			retrievedTickData, err := store.GetTickData(ctx, tc.tickNumber)
			require.NoError(t, err)

			if diff := cmp.Diff(tc.exp, retrievedTickData, cmpopts.IgnoreUnexported(pb.TickData{})); diff != "" {
				t.Fatalf("Unexpected result: %v", diff)
			}
		})
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
	quorumData := &pb.QuorumTickDataStored{
		QuorumTickStructure: &pb.QuorumTickStructure{
			Epoch:                 1,
			TickNumber:            101,
			Timestamp:             1596240001,
			PrevSpectrumDigestHex: "prevSpectrumDigest",
			PrevUniverseDigestHex: "prevUniverseDigest",
			PrevComputerDigestHex: "prevComputerDigest",
			TxDigestHex:           "txDigest",
		},
		QuorumDiffPerComputor: map[uint32]*pb.QuorumDiffStored{
			0: {
				ExpectedNextTickTxDigestHex: "expectedNextTickTxDigest",
				SignatureHex:                "signature",
			},
			1: {
				ExpectedNextTickTxDigestHex: "expectedNextTickTxDigest",
				SignatureHex:                "signature",
			},
		},
	}

	// Set QuorumTickData
	err = store.SetQuorumTickData(ctx, quorumData.QuorumTickStructure.TickNumber, quorumData)
	require.NoError(t, err)

	// Get QuorumTickData
	retrievedData, err := store.GetQuorumTickData(ctx, quorumData.QuorumTickStructure.TickNumber)
	require.NoError(t, err)

	if diff := cmp.Diff(quorumData, retrievedData, cmpopts.IgnoreUnexported(pb.QuorumTickDataStored{}, pb.QuorumTickStructure{}, pb.QuorumDiffStored{})); diff != "" {
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

	transactions := []*pb.Transaction{
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
	}

	tickData := pb.TickData{
		ComputorIndex:  1,
		Epoch:          1,
		TickNumber:     12795005,
		Timestamp:      1596240001,
		SignatureHex:   "signature1",
		TransactionIds: []string{"ff01", "cd01"},
	}
	err = store.SetTickData(ctx, tickData.TickNumber, &tickData)
	require.NoError(t, err, "Failed to store TickData")

	// Sample Transactions for testing
	tickNumber := uint32(12795005)

	// Assuming SetTransactions stores transactions for a tick
	err = store.SetTransactions(ctx, transactions)
	require.NoError(t, err)

	// GetTickTransactions retrieves stored transactions for a tick
	retrievedTransactions, err := store.GetTickTransactions(ctx, tickNumber)
	require.NoError(t, err)
	require.NotNil(t, retrievedTransactions)

	// Validate the retrieved transactions
	require.Len(t, retrievedTransactions, len(transactions))
	for i, tx := range transactions {
		retrievedTx := retrievedTransactions[i]
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
	transactions := []*pb.Transaction{targetTransaction}

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

	//value := make([]byte, 8)
	//binary.LittleEndian.PutUint64(value, 15)
	//key := lastProcessedTickKey()
	//
	//err = db.Set(key, value, pebble.Sync)
	//if err != nil {
	//	t.Fatal("setting last processed tick")
	//}
	//
	//key = lastProcessedTickKeyPerEpoch(1)
	//value = make([]byte, 8)
	//binary.LittleEndian.PutUint64(value, 15)
	//
	//err = db.Set(key, value, pebble.Sync)
	//if err != nil {
	//	t.Fatalf("setting last processed tick")
	//}
	//
	//key = lastProcessedTickKeyPerEpoch(2)
	//value = make([]byte, 8)
	//binary.LittleEndian.PutUint64(value, 16)
	//
	//err = db.Set(key, value, pebble.Sync)
	//if err != nil {
	//	t.Fatalf("setting last processed tick")
	//}
	//
	//got, err := store.GetLastProcessedTick(ctx)
	//if err != nil {
	//	t.Fatal("getting last processed tick")
	//}

	// Set last processed ticks per epoch
	err = store.SetLastProcessedTick(ctx, &pb.ProcessedTick{TickNumber: 16, Epoch: 1})
	require.NoError(t, err)

	// Get last processed tick per epoch
	got, err := store.GetLastProcessedTick(ctx)
	require.NoError(t, err)
	require.True(t, proto.Equal(&pb.ProcessedTick{TickNumber: 16, Epoch: 1}, got))

	lastProcessedTicksPerEpoch, err := store.GetLastProcessedTicksPerEpoch(ctx)
	require.NoError(t, err)
	require.Equal(t, map[uint32]uint32{1: 16}, lastProcessedTicksPerEpoch)

	// Set last processed ticks per epoch
	err = store.SetLastProcessedTick(ctx, &pb.ProcessedTick{TickNumber: 17, Epoch: 1})
	require.NoError(t, err)

	// Get last processed tick per epoch
	got, err = store.GetLastProcessedTick(ctx)
	require.NoError(t, err)
	require.True(t, proto.Equal(&pb.ProcessedTick{TickNumber: 17, Epoch: 1}, got))

	lastProcessedTicksPerEpoch, err = store.GetLastProcessedTicksPerEpoch(ctx)
	require.NoError(t, err)
	require.Equal(t, map[uint32]uint32{1: 17}, lastProcessedTicksPerEpoch)

	// Set last processed ticks per epoch
	err = store.SetLastProcessedTick(ctx, &pb.ProcessedTick{TickNumber: 18, Epoch: 2})
	require.NoError(t, err)

	// Get last processed tick per epoch
	got, err = store.GetLastProcessedTick(ctx)
	require.NoError(t, err)
	require.True(t, proto.Equal(&pb.ProcessedTick{TickNumber: 18, Epoch: 2}, got))

	lastProcessedTicksPerEpoch, err = store.GetLastProcessedTicksPerEpoch(ctx)
	require.NoError(t, err)
	require.Equal(t, map[uint32]uint32{1: 17, 2: 18}, lastProcessedTicksPerEpoch)
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
	}

	forTickTwo := pb.TransferTransactionsPerTick{
		TickNumber: 15,
		Identity:   idTwo,
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
	}

	err = store.PutTransferTransactionsPerTick(ctx, idOne, 12, &forTickOne)
	require.NoError(t, err)

	err = store.PutTransferTransactionsPerTick(ctx, idOne, 13, &forTickOne)
	require.NoError(t, err)

	got, err := store.GetTransferTransactions(ctx, idOne, 12, 12)
	require.NoError(t, err)
	diff := cmp.Diff([]*pb.TransferTransactionsPerTick{&forTickOne}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "comparing first TransferTransactionsPerTick for idOne, forTickOne")

	got, err = store.GetTransferTransactions(ctx, idOne, 13, 13)
	require.NoError(t, err)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{&forTickOne}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "comparing second TransferTransactionsPerTick for idOne, forTickOne")

	err = store.PutTransferTransactionsPerTick(ctx, idTwo, 15, &forTickTwo)
	require.NoError(t, err)
	got, err = store.GetTransferTransactions(ctx, idTwo, 15, 15)
	require.NoError(t, err)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{&forTickTwo}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "comparing TransferTransactionsPerTick for idTwo, forTickTwo")

	perIdentityTx, err := store.GetTransferTransactions(ctx, idOne, 12, 13)
	require.NoError(t, err)
	require.Equal(t, 2, len(perIdentityTx))

	expected := []*pb.TransferTransactionsPerTick{&forTickOne, &forTickOne}
	require.NoError(t, err)
	diff = cmp.Diff(expected, perIdentityTx, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "comparing perIdentityTx")

	// not existing identity means no transfers
	perIdentityTx, err = store.GetTransferTransactions(ctx, idThree, 1, 20)
	require.NoError(t, err)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{}, perIdentityTx, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "comparison of perIdentityTx for idThree")

	// not existing tick means no transfers
	perTickTx, err := store.GetTransferTransactions(ctx, idOne, 14, 14)
	require.NoError(t, err)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{}, perTickTx, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "comparison of perTickTx for idOne and tick 14")
}
func TestPebbleStore_ChainHash(t *testing.T) {
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

	// Sample ChainHash for testing
	tickNumber := uint32(12795005)
	qChainHash := []byte("qChainHash")

	// Set ChainHash
	err = store.PutChainDigest(ctx, tickNumber, qChainHash)
	require.NoError(t, err)

	// Get ChainHash
	retrievedChainHash, err := store.GetChainDigest(ctx, tickNumber)
	require.NoError(t, err)
	require.Equal(t, qChainHash, retrievedChainHash)

	// Test retrieval of non-existent ChainHash
	_, err = store.GetChainDigest(ctx, 999) // Assuming 999 is a tick number that wasn't stored
	require.Error(t, err)
	require.Equal(t, ErrNotFound, err)
}

func TestPebbleStore_LastProcessedTickIntervals(t *testing.T) {
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

	firstEpochInitialTick := pb.ProcessedTick{TickNumber: 100, Epoch: 1}
	expected := &pb.ProcessedTickIntervalsPerEpoch{
		Epoch: firstEpochInitialTick.Epoch,
		Intervals: []*pb.ProcessedTickInterval{
			{
				InitialProcessedTick: firstEpochInitialTick.TickNumber,
				LastProcessedTick:    firstEpochInitialTick.TickNumber,
			},
		},
	}
	err = store.AppendProcessedTickInterval(ctx, firstEpochInitialTick.Epoch, expected.Intervals[0])
	require.NoError(t, err)

	got, err := store.getProcessedTickIntervalsPerEpoch(ctx, firstEpochInitialTick.Epoch)
	require.NoError(t, err)
	require.True(t, proto.Equal(got, expected))

	err = store.SetLastProcessedTick(ctx, &firstEpochInitialTick)
	require.NoError(t, err)

	got, err = store.getProcessedTickIntervalsPerEpoch(ctx, firstEpochInitialTick.Epoch)
	require.NoError(t, err)
	require.True(t, proto.Equal(got, expected))

	firstEpochSecondTick := pb.ProcessedTick{TickNumber: 101, Epoch: 1}
	expected = &pb.ProcessedTickIntervalsPerEpoch{
		Epoch: firstEpochInitialTick.Epoch,
		Intervals: []*pb.ProcessedTickInterval{
			{
				InitialProcessedTick: firstEpochInitialTick.TickNumber,
				LastProcessedTick:    firstEpochSecondTick.TickNumber,
			},
		},
	}
	err = store.SetLastProcessedTick(ctx, &firstEpochSecondTick)
	require.NoError(t, err, "setting last processed tick")

	got, err = store.getProcessedTickIntervalsPerEpoch(ctx, firstEpochSecondTick.Epoch)
	require.NoError(t, err)
	require.True(t, proto.Equal(got, expected))

	firstEpochFourthTick := pb.ProcessedTick{TickNumber: 103, Epoch: 1}
	expected = &pb.ProcessedTickIntervalsPerEpoch{
		Epoch: firstEpochInitialTick.Epoch,
		Intervals: []*pb.ProcessedTickInterval{
			{
				InitialProcessedTick: firstEpochInitialTick.TickNumber,
				LastProcessedTick:    firstEpochSecondTick.TickNumber,
			},
			{
				InitialProcessedTick: firstEpochFourthTick.TickNumber,
				LastProcessedTick:    firstEpochFourthTick.TickNumber,
			},
		},
	}

	//skip ticks inside same epoch
	err = store.AppendProcessedTickInterval(ctx, firstEpochInitialTick.Epoch, expected.Intervals[1])
	require.NoError(t, err)

	got, err = store.getProcessedTickIntervalsPerEpoch(ctx, firstEpochSecondTick.Epoch)
	require.NoError(t, err)
	require.True(t, proto.Equal(got, expected))

	err = store.SetLastProcessedTick(ctx, &firstEpochFourthTick)
	require.NoError(t, err, "setting last processed tick")

	got, err = store.getProcessedTickIntervalsPerEpoch(ctx, firstEpochSecondTick.Epoch)
	require.NoError(t, err)
	require.True(t, proto.Equal(got, expected))

	firstEpochFifthTick := pb.ProcessedTick{TickNumber: 104, Epoch: 1}
	expected = &pb.ProcessedTickIntervalsPerEpoch{
		Epoch: firstEpochInitialTick.Epoch,
		Intervals: []*pb.ProcessedTickInterval{
			{
				InitialProcessedTick: firstEpochInitialTick.TickNumber,
				LastProcessedTick:    firstEpochSecondTick.TickNumber,
			},
			{
				InitialProcessedTick: firstEpochFourthTick.TickNumber,
				LastProcessedTick:    firstEpochFifthTick.TickNumber,
			},
		},
	}
	err = store.SetLastProcessedTick(ctx, &firstEpochFifthTick)
	require.NoError(t, err, "setting last processed tick")

	got, err = store.getProcessedTickIntervalsPerEpoch(ctx, firstEpochSecondTick.Epoch)
	require.NoError(t, err)
	require.True(t, proto.Equal(got, expected))

	//

	secondEpochInitialTick := pb.ProcessedTick{TickNumber: 200, Epoch: 2}
	expected = &pb.ProcessedTickIntervalsPerEpoch{
		Epoch: secondEpochInitialTick.Epoch,
		Intervals: []*pb.ProcessedTickInterval{
			{
				InitialProcessedTick: secondEpochInitialTick.TickNumber,
				LastProcessedTick:    secondEpochInitialTick.TickNumber,
			},
		},
	}
	err = store.AppendProcessedTickInterval(ctx, secondEpochInitialTick.Epoch, expected.Intervals[0])
	require.NoError(t, err)

	got, err = store.getProcessedTickIntervalsPerEpoch(ctx, secondEpochInitialTick.Epoch)
	require.NoError(t, err)
	require.True(t, proto.Equal(got, expected))

	err = store.SetLastProcessedTick(ctx, &secondEpochInitialTick)
	require.NoError(t, err)

	got, err = store.getProcessedTickIntervalsPerEpoch(ctx, secondEpochInitialTick.Epoch)
	require.NoError(t, err)
	require.True(t, proto.Equal(got, expected))

	secondEpochSecondTick := pb.ProcessedTick{TickNumber: 201, Epoch: 2}
	expected = &pb.ProcessedTickIntervalsPerEpoch{
		Epoch: secondEpochInitialTick.Epoch,
		Intervals: []*pb.ProcessedTickInterval{
			{
				InitialProcessedTick: secondEpochInitialTick.TickNumber,
				LastProcessedTick:    secondEpochSecondTick.TickNumber,
			},
		},
	}
	err = store.SetLastProcessedTick(ctx, &secondEpochSecondTick)
	require.NoError(t, err, "setting last processed tick")

	got, err = store.getProcessedTickIntervalsPerEpoch(ctx, secondEpochSecondTick.Epoch)
	require.NoError(t, err)
	require.True(t, proto.Equal(got, expected))

	expectedCombined := []*pb.ProcessedTickIntervalsPerEpoch{
		{
			Epoch: firstEpochInitialTick.Epoch,
			Intervals: []*pb.ProcessedTickInterval{
				{
					InitialProcessedTick: firstEpochInitialTick.TickNumber,
					LastProcessedTick:    firstEpochSecondTick.TickNumber,
				},
				{
					InitialProcessedTick: firstEpochFourthTick.TickNumber,
					LastProcessedTick:    firstEpochFifthTick.TickNumber,
				},
			},
		},
		{
			Epoch: secondEpochInitialTick.Epoch,
			Intervals: []*pb.ProcessedTickInterval{
				{
					InitialProcessedTick: secondEpochInitialTick.TickNumber,
					LastProcessedTick:    secondEpochSecondTick.TickNumber,
				},
			},
		},
	}

	gotCombined, err := store.GetProcessedTickIntervals(ctx)
	require.NoError(t, err)
	diff := cmp.Diff(gotCombined, expectedCombined, cmpopts.IgnoreUnexported(pb.ProcessedTickInterval{}, pb.ProcessedTickIntervalsPerEpoch{}))
	require.True(t, cmp.Equal(diff, ""))
}

func TestPebbleStore_SetAndGetEmptyTickListForEpoch(t *testing.T) {

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	require.NoError(t, err)
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	require.NoError(t, err)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	store := NewPebbleStore(db, logger)

	epoch := uint32(130)

	emptyTickList := []uint32{
		16393119, 16393320, 16393310, 13390319,
		16393419, 13363379, 16493319, 17393419,
		16393619, 13393319, 16798319, 15393719}

	err = store.SetEmptyTickListPerEpoch(epoch, emptyTickList)
	require.NoError(t, err, "setting empty tick list")

	got, err := store.GetEmptyTickListPerEpoch(epoch)
	require.NoError(t, err, "getting empty tick list")

	diff := cmp.Diff(got, emptyTickList)
	require.True(t, cmp.Equal(diff, ""))

}

func TestPebbleStore_AppendToEmptyTickList(t *testing.T) {
	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	require.NoError(t, err)
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	require.NoError(t, err)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	store := NewPebbleStore(db, logger)

	epoch := uint32(130)

	emptyTickList := []uint32{
		16393119, 16393320, 16393310, 13390319,
		16393419, 13363379, 16493319, 17393419,
		16393619, 13393319, 16798319, 15393719}

	toAppend := uint32(10000000)

	err = store.SetEmptyTickListPerEpoch(epoch, emptyTickList)
	require.NoError(t, err, "setting empty tick list")

	err = store.AppendEmptyTickToEmptyTickListPerEpoch(epoch, toAppend)
	require.NoError(t, err, "appending to empty tick list")

	expected := append(emptyTickList, toAppend)

	got, err := store.GetEmptyTickListPerEpoch(epoch)
	require.NoError(t, err, "getting empty tick list")

	diff := cmp.Diff(got, expected)
	require.True(t, cmp.Equal(diff, ""))
}
