package tx

import (
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
)

func qubicToProto(txs types.Transactions) (*protobuff.Transactions, error){
	protoTxs := protobuff.Transactions{
		Transactions: make([]*protobuff.Transaction, len(txs)),
	}
	for i, tx := range txs {
		txProto, err := txToProto(tx)
		if err != nil {
			return nil, errors.Wrapf(err, "converting tx to proto")
		}
		protoTxs.Transactions[i] = txProto
	}

	return &protoTxs, nil
}

func txToProto(tx types.Transaction) (*protobuff.Transaction, error) {
	digest, err := tx.Digest()
	if err != nil {
		return nil, errors.Wrap(err, "getting tx digest")
	}

	return &protobuff.Transaction{
		SourcePubkeyHex: hex.EncodeToString(tx.SourcePublicKey[:]),
		DestPubkeyHex:   hex.EncodeToString(tx.DestinationPublicKey[:]),
		Amount:          tx.Amount,
		TickNumber:      tx.Tick,
		InputType:       uint32(tx.InputType),
		InputSize:       uint32(tx.InputSize),
		InputHex:        hex.EncodeToString(tx.Input[:]),
		SignatureHex:    hex.EncodeToString(tx.Signature[:]),
		DigestHex: hex.EncodeToString(digest[:]),
	}, nil
}
