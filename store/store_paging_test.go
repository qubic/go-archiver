package store

import (
	"context"
	"flag"
	"github.com/cockroachdb/pebble"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"log"
	"os"
	"path/filepath"
	"testing"

	pb "github.com/qubic/go-archiver/protobuff" // Update the import path to your generated protobuf package
)

var (
	store *PebbleStore
)

func TestPebbleStore_TransfersPaging_GivenAllFitsPage_ThenReturnAll(t *testing.T) {
	ctx := context.Background()
	idOne := "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB"
	forTickOne := tickOne(t, idOne)
	forTickTwo := tickTwo(t, idOne)

	got, _, err := store.GetTransactionsForEntityPaged(ctx, idOne, 12, 12, Pageable{0, 2}, Sortable{}, Filterable{}) // only first tick in range
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Len(t, got[0].Transactions, 2)
	diff := cmp.Diff([]*pb.TransferTransactionsPerTick{forTickOne}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "comparing first TransferTransactionsPerTick for idOne, for first tick")

	got, _, err = store.GetTransactionsForEntityPaged(ctx, idOne, 10, 20, Pageable{0, 2}, Sortable{}, Filterable{}) // both ticks available
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Len(t, got[0].Transactions, 2)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{forTickOne}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "comparing first TransferTransactionsPerTick for idOne, for first tick")

	got, _, err = store.GetTransactionsForEntityPaged(ctx, idOne, 15, 15, Pageable{0, 3}, Sortable{}, Filterable{}) // only second tick in range
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Len(t, got[0].Transactions, 3)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{forTickTwo}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "comparing second TransferTransactionsPerTick for idOne, for second tick")

}

func TestPebbleStore_TransfersPaging_GivenPartOfTickFitsPage_ThenPage(t *testing.T) {
	ctx := context.Background()
	idOne := "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB"
	forTickTwo := tickTwo(t, idOne)

	got, _, err := store.GetTransactionsForEntityPaged(ctx, idOne, 15, 15, Pageable{0, 2}, Sortable{}, Filterable{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Len(t, got[0].Transactions, 2)
	diff := cmp.Diff([]*pb.TransferTransactionsPerTick{
		{
			TickNumber:   forTickTwo.TickNumber,
			Identity:     forTickTwo.Identity,
			Transactions: forTickTwo.Transactions[:2],
		},
	}[:], got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "comparing second TransferTransactionsPerTick for idOne, for second tick")

	got, _, err = store.GetTransactionsForEntityPaged(ctx, idOne, 15, 15, Pageable{1, 1}, Sortable{}, Filterable{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Len(t, got[0].Transactions, 1)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{
		{
			TickNumber:   forTickTwo.TickNumber,
			Identity:     forTickTwo.Identity,
			Transactions: forTickTwo.Transactions[1:2],
		},
	}[:], got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "comparing second TransferTransactionsPerTick for idOne, for second tick")

	got, _, err = store.GetTransactionsForEntityPaged(ctx, idOne, 15, 15, Pageable{1, 2}, Sortable{}, Filterable{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Len(t, got[0].Transactions, 1)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{
		{
			TickNumber:   forTickTwo.TickNumber,
			Identity:     forTickTwo.Identity,
			Transactions: forTickTwo.Transactions[2:3],
		},
	}[:], got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "comparing second TransferTransactionsPerTick for idOne, for second tick")
}

func TestPebbleStore_TransfersPaging_GivenPageOutOfRange_ThenEmpty(t *testing.T) {
	ctx := context.Background()
	idOne := "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB"
	tickTwo(t, idOne)

	got, _, err := store.GetTransactionsForEntityPaged(ctx, idOne, 15, 15, Pageable{1, 3}, Sortable{}, Filterable{})
	require.NoError(t, err)
	require.Len(t, got, 0) // all items in page 0

	got, _, err = store.GetTransactionsForEntityPaged(ctx, idOne, 15, 15, Pageable{3, 1}, Sortable{}, Filterable{})
	require.NoError(t, err)
	require.Len(t, got, 0) // all items in page 0
}

func TestPebbleStore_TransfersPaging_GivenPageSpansAllData_ThenReturnAll(t *testing.T) {
	ctx := context.Background()
	idOne := "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB"
	forTickOne := tickOne(t, idOne)
	forTickTwo := tickTwo(t, idOne)

	got, _, err := store.GetTransactionsForEntityPaged(ctx, idOne, 10, 20, Pageable{0, 5}, Sortable{}, Filterable{})
	require.NoError(t, err)
	require.Len(t, got, 2)
	diff := cmp.Diff([]*pb.TransferTransactionsPerTick{forTickOne, forTickTwo}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "expecting last two from tick two")

	got, _, err = store.GetTransactionsForEntityPaged(ctx, idOne, 10, 20, Pageable{0, 5000000}, Sortable{}, Filterable{})
	require.NoError(t, err)
	require.Len(t, got, 2)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{forTickOne, forTickTwo}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "expecting all 5 (size = 5).")

}

func TestPebbleStore_TransfersPaging_GivenPageSpansMultipleTicks_ThenCombineInPage(t *testing.T) {
	ctx := context.Background()
	idOne := "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB"
	forTickOne := tickOne(t, idOne)
	forTickTwo := tickTwo(t, idOne)

	got, _, err := store.GetTransactionsForEntityPaged(ctx, idOne, 10, 20, Pageable{0, 3}, Sortable{}, Filterable{})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Len(t, got[0].Transactions, 2)
	require.Len(t, got[1].Transactions, 1)
	diff := cmp.Diff([]*pb.TransferTransactionsPerTick{forTickOne,
		{
			TickNumber:   forTickTwo.TickNumber,
			Identity:     forTickTwo.Identity,
			Transactions: forTickTwo.Transactions[0:1],
		}}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "expecting 2 from tick one, 1 from tick two")

	got, _, err = store.GetTransactionsForEntityPaged(ctx, idOne, 10, 20, Pageable{1, 2}, Sortable{}, Filterable{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Len(t, got[0].Transactions, 2)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{
		{
			TickNumber:   forTickTwo.TickNumber,
			Identity:     forTickTwo.Identity,
			Transactions: forTickTwo.Transactions[0:2],
		}}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "expecting first 2 from tick two")

	got, _, err = store.GetTransactionsForEntityPaged(ctx, idOne, 10, 20, Pageable{2, 2}, Sortable{}, Filterable{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Len(t, got[0].Transactions, 1)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{
		{
			TickNumber:   forTickTwo.TickNumber,
			Identity:     forTickTwo.Identity,
			Transactions: forTickTwo.Transactions[2:3],
		}}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "expecting last one from tick two")

	got, _, err = store.GetTransactionsForEntityPaged(ctx, idOne, 10, 20, Pageable{1, 3}, Sortable{}, Filterable{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Len(t, got[0].Transactions, 2)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{
		{
			TickNumber:   forTickTwo.TickNumber,
			Identity:     forTickTwo.Identity,
			Transactions: forTickTwo.Transactions[1:3],
		}}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "expecting last two from tick two")
}

func TestPebbleStore_TransfersPaging_GivenSortDesc_ThenSortPage(t *testing.T) {
	ctx := context.Background()
	idOne := "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB"
	forTickOne := tickOne(t, idOne)
	forTickTwo := tickTwo(t, idOne)

	got, _, err := store.GetTransactionsForEntityPaged(ctx, idOne, 10, 20, Pageable{0, 2}, Sortable{Descending: true}, Filterable{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Len(t, got[0].Transactions, 2)
	diff := cmp.Diff([]*pb.TransferTransactionsPerTick{{
		TickNumber:   forTickTwo.TickNumber,
		Identity:     forTickTwo.Identity,
		Transactions: forTickTwo.Transactions[0:2], // last one because first two are on page '0' already.
	}}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "expecting 2 from tick two")

	got, _, err = store.GetTransactionsForEntityPaged(ctx, idOne, 10, 20, Pageable{1, 2}, Sortable{Descending: true}, Filterable{})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Len(t, got[0].Transactions, 1)
	require.Len(t, got[1].Transactions, 1)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{{
		TickNumber:   forTickTwo.TickNumber,
		Identity:     forTickTwo.Identity,
		Transactions: forTickTwo.Transactions[2:3], // last one because first two are on page '0' already.
	}, {
		TickNumber:   forTickOne.TickNumber,
		Identity:     forTickOne.Identity,
		Transactions: forTickOne.Transactions[0:1], // first one because we do not reverse sort inside ticks.
	}}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "expecting 1 from tick two then 1 from tick one")

	got, _, err = store.GetTransactionsForEntityPaged(ctx, idOne, 10, 20, Pageable{2, 2}, Sortable{Descending: true}, Filterable{})
	require.NoError(t, err)
	require.Len(t, got, 1)
	require.Len(t, got[0].Transactions, 1)
	diff = cmp.Diff([]*pb.TransferTransactionsPerTick{{
		TickNumber:   forTickOne.TickNumber,
		Identity:     forTickOne.Identity,
		Transactions: forTickOne.Transactions[1:2], // last one is the first one of the first tick
	}}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "expecting 1 from tick one")
}

func TestPebbleStore_TransfersPaging_GivenScOnly_ThenSkipTransfers(t *testing.T) {
	ctx := context.Background()
	idOne := "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB"
	forTickOne := tickOne(t, idOne)
	forTickTwo := tickTwo(t, idOne)

	tickTwoTransactions := make([]*pb.Transaction, 0)
	tickTwoTransactions = append(tickTwoTransactions, forTickTwo.GetTransactions()[0])
	tickTwoTransactions = append(tickTwoTransactions, forTickTwo.GetTransactions()[2])

	got, _, err := store.GetTransactionsForEntityPaged(ctx, idOne, 10, 20, Pageable{0, 5}, Sortable{Descending: false}, Filterable{ScOnly: true})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Len(t, got[0].Transactions, 1)
	require.Len(t, got[1].Transactions, 2)
	diff := cmp.Diff([]*pb.TransferTransactionsPerTick{{
		TickNumber:   forTickOne.TickNumber,
		Identity:     forTickOne.Identity,
		Transactions: forTickOne.Transactions[1:2], // last one of first tick is a smart contract transaction
	}, {
		TickNumber:   forTickTwo.TickNumber,
		Identity:     forTickTwo.Identity,
		Transactions: tickTwoTransactions, // first and last of tick two are smart contract transactions
	}}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "expecting sc transactions at position 2, 3, and 5")
}

func TestPebbleStore_TransfersPaging_GivenDescendingScOnly_ThenSortAndSkipTransfers(t *testing.T) {
	ctx := context.Background()
	idOne := "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB"
	forTickOne := tickOne(t, idOne)
	forTickTwo := tickTwo(t, idOne)

	tickTwoTransactions := make([]*pb.Transaction, 0)
	tickTwoTransactions = append(tickTwoTransactions, forTickTwo.GetTransactions()[0])
	tickTwoTransactions = append(tickTwoTransactions, forTickTwo.GetTransactions()[2])

	got, _, err := store.GetTransactionsForEntityPaged(ctx, idOne, 10, 20, Pageable{0, 5}, Sortable{Descending: true}, Filterable{ScOnly: true})
	require.NoError(t, err)
	require.Len(t, got, 2)
	require.Len(t, got[0].Transactions, 2)
	require.Len(t, got[1].Transactions, 1)
	diff := cmp.Diff([]*pb.TransferTransactionsPerTick{{
		TickNumber:   forTickTwo.TickNumber,
		Identity:     forTickTwo.Identity,
		Transactions: tickTwoTransactions, // first and last of tick two are smart contract transactions
	}, {
		TickNumber:   forTickOne.TickNumber,
		Identity:     forTickOne.Identity,
		Transactions: forTickOne.Transactions[1:2], // last one of first tick is a smart contract transaction
	}}, got, cmpopts.IgnoreUnexported(pb.TransferTransactionsPerTick{}, pb.Transaction{}))
	require.Equal(t, "", diff, "expecting sc transactions at position 5,3, and 2")
}

func tickOne(t *testing.T, idOne string) *pb.TransferTransactionsPerTick {
	// Sample Transactions for testing
	forTickOne := pb.TransferTransactionsPerTick{
		TickNumber: 12,
		Identity:   idOne,
		Transactions: []*pb.Transaction{
			{
				SourceId:     "aaa",
				DestId:       "bbb",
				Amount:       15,
				TickNumber:   12,
				InputType:    0,
				InputSize:    122,
				InputHex:     "ccc",
				SignatureHex: "ddd",
				TxId:         "eee",
			},
			{
				SourceId:     "bbb",
				DestId:       "ccc",
				Amount:       25,
				TickNumber:   12,
				InputType:    666,
				InputSize:    24,
				InputHex:     "ddd",
				SignatureHex: "eee",
				TxId:         "fff",
			},
		},
	}
	err := store.PutTransferTransactionsPerTick(context.Background(), idOne, 12, &forTickOne)
	require.NoError(t, err)
	return &forTickOne
}

func tickTwo(t *testing.T, idOne string) *pb.TransferTransactionsPerTick {
	forTickTwo := pb.TransferTransactionsPerTick{
		TickNumber: 15,
		Identity:   idOne,
		Transactions: []*pb.Transaction{
			{
				SourceId:     "ccc",
				DestId:       "ddd",
				Amount:       15,
				TickNumber:   15,
				InputType:    87,
				InputSize:    122,
				InputHex:     "eee",
				SignatureHex: "fff",
				TxId:         "ggg",
			},
			{
				SourceId:     "ddd",
				DestId:       "eee",
				Amount:       25,
				TickNumber:   15,
				InputType:    0,
				InputSize:    24,
				InputHex:     "fff",
				SignatureHex: "ggg",
				TxId:         "hhh",
			},
			{
				SourceId:     "eee",
				DestId:       "fff",
				Amount:       12,
				TickNumber:   23,
				InputType:    34,
				InputSize:    45,
				InputHex:     "ggg",
				SignatureHex: "hhh",
				TxId:         "iii",
			},
		},
	}
	err := store.PutTransferTransactionsPerTick(context.Background(), idOne, 15, &forTickTwo)
	require.NoError(t, err)
	return &forTickTwo
}

func TestMain(m *testing.M) {

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	if err != nil {
		log.Fatal(err)
	}
	//goland:noinspection ALL
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	if err != nil {
		log.Fatal(err)
	}
	//goland:noinspection ALL
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	store = NewPebbleStore(db, logger)

	// Parse args and run
	flag.Parse()
	exitCode := m.Run()
	// Exit
	os.Exit(exitCode)
}
