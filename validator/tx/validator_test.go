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

	expectedFirstTickFirstID := &protobuff.TransferTransactionsPerTick{
		TickNumber: 1,
		Identity:   "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
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

	expectedFirstTickSecondID := &protobuff.TransferTransactionsPerTick{
		TickNumber: 1,
		Identity:   "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
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

	got, err := s.GetTransferTransactions(ctx, "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB", 1, 1)
	require.NoError(t, err)
	diff := cmp.Diff(got, []*protobuff.TransferTransactionsPerTick{expectedFirstTickFirstID}, cmpopts.IgnoreFields(protobuff.Transaction{}, "TxId"), cmpopts.IgnoreUnexported(protobuff.TransferTransactionsPerTick{}, protobuff.Transactions{}, protobuff.Transaction{}))
	require.Empty(t, diff)

	got, err = s.GetTransferTransactions(ctx, "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB", 1, 1)
	require.NoError(t, err)
	diff = cmp.Diff(got, []*protobuff.TransferTransactionsPerTick{expectedFirstTickSecondID}, cmpopts.IgnoreFields(protobuff.Transaction{}, "TxId"), cmpopts.IgnoreUnexported(protobuff.TransferTransactionsPerTick{}, protobuff.Transactions{}, protobuff.Transaction{}))
	require.Empty(t, diff)

	err = Store(ctx, s, 2, secondTick)
	require.NoError(t, err)

	expectedSecondTickFirstID := &protobuff.TransferTransactionsPerTick{
		TickNumber: 2,
		Identity:   "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB",
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

	expectedSecondTickSecondID := &protobuff.TransferTransactionsPerTick{
		TickNumber: 2,
		Identity:   "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB",
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

	got, err = s.GetTransferTransactions(ctx, "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB", 2, 2)
	require.NoError(t, err)
	diff = cmp.Diff(got, []*protobuff.TransferTransactionsPerTick{expectedSecondTickFirstID}, cmpopts.IgnoreFields(protobuff.Transaction{}, "TxId"), cmpopts.IgnoreUnexported(protobuff.TransferTransactionsPerTick{}, protobuff.Transactions{}, protobuff.Transaction{}))
	require.Empty(t, diff)

	got, err = s.GetTransferTransactions(ctx, "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB", 2, 2)
	require.NoError(t, err)
	diff = cmp.Diff(got, []*protobuff.TransferTransactionsPerTick{expectedSecondTickSecondID}, cmpopts.IgnoreFields(protobuff.Transaction{}, "TxId"), cmpopts.IgnoreUnexported(protobuff.TransferTransactionsPerTick{}, protobuff.Transactions{}, protobuff.Transaction{}))
	require.Empty(t, diff)

	expectedCombined := []*protobuff.TransferTransactionsPerTick{expectedFirstTickFirstID, expectedSecondTickFirstID}
	gotCombined, err := s.GetTransferTransactions(ctx, "QJRRSSKMJRDKUDTYVNYGAMQPULKAMILQQYOWBEXUDEUWQUMNGDHQYLOAJMEB", 1, 2)
	require.NoError(t, err)
	diff = cmp.Diff(gotCombined, expectedCombined, cmpopts.IgnoreFields(protobuff.Transaction{}, "TxId"), cmpopts.IgnoreUnexported(protobuff.TransferTransactionsPerTick{}, protobuff.Transactions{}, protobuff.Transaction{}))
	require.Empty(t, diff)

	expectedCombined = []*protobuff.TransferTransactionsPerTick{expectedFirstTickSecondID, expectedSecondTickSecondID}
	gotCombined, err = s.GetTransferTransactions(ctx, "IXTSDANOXIVIWGNDCNZVWSAVAEPBGLGSQTLSVHHBWEGKSEKPRQGWIJJCTUZB", 1, 2)
	require.NoError(t, err)
	diff = cmp.Diff(gotCombined, expectedCombined, cmpopts.IgnoreFields(protobuff.Transaction{}, "TxId"), cmpopts.IgnoreUnexported(protobuff.TransferTransactionsPerTick{}, protobuff.Transactions{}, protobuff.Transaction{}))
	require.Empty(t, diff)
}

func identityToPubkeyNoError(id string) [32]byte {
	identity := types.Identity(id)
	pubKey, _ := identity.ToPubKey(false)

	return pubKey
}
