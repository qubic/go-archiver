package tx

import (
	"context"
	"github.com/cockroachdb/pebble"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-node-connector/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"os"
	"path/filepath"
	"testing"
)

func Test_CreateTransferTransactionsIdentityMap(t *testing.T) {
	txs := protobuff.Transactions{
		Transactions: []*protobuff.Transaction{
			{
				SourceId: "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
				DestId:   "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
			},
			{
				SourceId: "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
				DestId:   "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
			},
			{
				SourceId: "AXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
				DestId:   "BJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
			},
			{
				SourceId: "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
				DestId:   "ZXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
			},
		},
	}

	expected := map[string]*protobuff.Transactions{
		"QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB": {
			Transactions: []*protobuff.Transaction{
				{
					SourceId: "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
					DestId:   "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
				},
				{
					SourceId: "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
					DestId:   "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
				},
				{
					SourceId: "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
					DestId:   "ZXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
				},
			},
		},
		"IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB": {
			Transactions: []*protobuff.Transaction{
				{
					SourceId: "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
					DestId:   "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
				},
				{
					SourceId: "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
					DestId:   "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
				},
			},
		},
		"AXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB": {
			Transactions: []*protobuff.Transaction{
				{
					SourceId: "AXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
					DestId:   "BJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
				},
			},
		},
		"BJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB": {
			Transactions: []*protobuff.Transaction{
				{
					SourceId: "AXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
					DestId:   "BJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
				},
			},
		},
		"ZXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB": {
			Transactions: []*protobuff.Transaction{
				{
					SourceId: "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
					DestId:   "ZXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
				},
			},
		},
	}

	got, err := createTransferTransactionsIdentityMap(context.Background(), &txs)
	require.NoError(t, err)
	diff := cmp.Diff(got, expected, cmpopts.IgnoreUnexported(protobuff.Transactions{}, protobuff.Transaction{}))
	require.Empty(t, diff)
}

func TestStore(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	require.NoError(t, err)
	defer os.RemoveAll(dbDir)

	db, err := pebble.Open(filepath.Join(dbDir, "testdb"), &pebble.Options{})
	require.NoError(t, err)
	defer db.Close()

	logger, _ := zap.NewDevelopment()
	s := store.NewPebbleStore(db, logger)

	firstTick := []types.Transaction{
		{
			SourcePublicKey:      identityToPubkeyNoError("QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB"),
			DestinationPublicKey: identityToPubkeyNoError("IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB"),
			Amount:               15,
		},
		{
			SourcePublicKey:      identityToPubkeyNoError("IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB"),
			DestinationPublicKey: identityToPubkeyNoError("QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB"),
			Amount:               20,
		},
	}

	secondTick := []types.Transaction{
		{
			SourcePublicKey:      identityToPubkeyNoError("QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB"),
			DestinationPublicKey: identityToPubkeyNoError("IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB"),
			Amount:               25,
		},
		{
			SourcePublicKey:      identityToPubkeyNoError("IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB"),
			DestinationPublicKey: identityToPubkeyNoError("QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB"),
			Amount:               30,
		},

		{
			SourcePublicKey:      identityToPubkeyNoError("IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB"),
			DestinationPublicKey: identityToPubkeyNoError("QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB"),
			Amount:               0,
		},
		{
			SourcePublicKey:      identityToPubkeyNoError("QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB"),
			DestinationPublicKey: identityToPubkeyNoError("IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB"),
			Amount:               0,
		},
	}

	err = Store(ctx, s, 1, firstTick)
	require.NoError(t, err)

	expectedFirstTick := &protobuff.TransferTransactionsPerTick{
		TickNumber: 1,
		Transactions: &protobuff.Transactions{Transactions: []*protobuff.Transaction{
			{
				SourceId:     "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
				DestId:       "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
				Amount:       15,
				SignatureHex: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			},
			{
				SourceId:     "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
				DestId:       "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
				Amount:       20,
				SignatureHex: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			},
		}},
	}

	got, err := s.GetTransferTransactionsPerTick(ctx, "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB", 1)
	require.NoError(t, err)
	diff := cmp.Diff(got, expectedFirstTick, cmpopts.IgnoreFields(protobuff.Transaction{}, "TxId"), cmpopts.IgnoreUnexported(protobuff.TransferTransactionsPerTick{}, protobuff.Transactions{}, protobuff.Transaction{}))
	require.Empty(t, diff)

	got, err = s.GetTransferTransactionsPerTick(ctx, "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB", 1)
	require.NoError(t, err)
	diff = cmp.Diff(got, expectedFirstTick, cmpopts.IgnoreFields(protobuff.Transaction{}, "TxId"), cmpopts.IgnoreUnexported(protobuff.TransferTransactionsPerTick{}, protobuff.Transactions{}, protobuff.Transaction{}))
	require.Empty(t, diff)

	err = Store(ctx, s, 2, secondTick)
	require.NoError(t, err)

	expectedSecondTick := &protobuff.TransferTransactionsPerTick{
		TickNumber: 2,
		Transactions: &protobuff.Transactions{Transactions: []*protobuff.Transaction{
			{
				SourceId:     "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
				DestId:       "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
				Amount:       25,
				SignatureHex: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			},
			{
				SourceId:     "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
				DestId:       "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
				Amount:       30,
				SignatureHex: "00000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
			},
		}},
	}

	got, err = s.GetTransferTransactionsPerTick(ctx, "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB", 2)
	require.NoError(t, err)
	diff = cmp.Diff(got, expectedSecondTick, cmpopts.IgnoreFields(protobuff.Transaction{}, "TxId"), cmpopts.IgnoreUnexported(protobuff.TransferTransactionsPerTick{}, protobuff.Transactions{}, protobuff.Transaction{}))
	require.Empty(t, diff)

	got, err = s.GetTransferTransactionsPerTick(ctx, "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB", 2)
	require.NoError(t, err)
	diff = cmp.Diff(got, expectedSecondTick, cmpopts.IgnoreFields(protobuff.Transaction{}, "TxId"), cmpopts.IgnoreUnexported(protobuff.TransferTransactionsPerTick{}, protobuff.Transactions{}, protobuff.Transaction{}))
	require.Empty(t, diff)

	expectedCombined := &protobuff.TransferTransactions{TransferTransactionsPerTick: []*protobuff.TransferTransactionsPerTick{expectedFirstTick, expectedSecondTick}}
	gotCombined, err := s.GetTransferTransactions(ctx, "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB")
	require.NoError(t, err)
	diff = cmp.Diff(gotCombined, expectedCombined, cmpopts.IgnoreFields(protobuff.Transaction{}, "TxId"), cmpopts.IgnoreUnexported(protobuff.TransferTransactions{}, protobuff.TransferTransactionsPerTick{}, protobuff.Transactions{}, protobuff.Transaction{}))
	require.Empty(t, diff)

	gotCombined, err = s.GetTransferTransactions(ctx, "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB")
	require.NoError(t, err)
	diff = cmp.Diff(gotCombined, expectedCombined, cmpopts.IgnoreFields(protobuff.Transaction{}, "TxId"), cmpopts.IgnoreUnexported(protobuff.TransferTransactions{}, protobuff.TransferTransactionsPerTick{}, protobuff.Transactions{}, protobuff.Transaction{}))
	require.Empty(t, diff)
}

func identityToPubkeyNoError(id string) [32]byte {
	identity := types.Identity(id)
	pubKey, _ := identity.ToPubKey(false)

	return pubKey
}
