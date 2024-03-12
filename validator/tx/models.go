package tx

import (
	"encoding/hex"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
)

func qubicToProto(txs types.Transactions) ([]*protobuff.Transaction, error) {
	protoTxs := make([]*protobuff.Transaction, len(txs))
	for i, tx := range txs {
		txProto, err := txToProto(tx)
		if err != nil {
			return nil, errors.Wrapf(err, "converting tx to proto")
		}
		protoTxs[i] = txProto
	}

	return protoTxs, nil
}

func txToProto(tx types.Transaction) (*protobuff.Transaction, error) {
	digest, err := tx.Digest()
	if err != nil {
		return nil, errors.Wrap(err, "getting tx digest")
	}
	var txID types.Identity
	txID, err = txID.FromPubKey(digest, true)
	if err != nil {
		return nil, errors.Wrap(err, "getting tx id")
	}

	var sourceID types.Identity
	sourceID, err = sourceID.FromPubKey(tx.SourcePublicKey, false)
	if err != nil {
		return nil, errors.Wrap(err, "getting source id")
	}

	var destID types.Identity
	destID, err = destID.FromPubKey(tx.DestinationPublicKey, false)
	if err != nil {
		return nil, errors.Wrap(err, "getting dest id")
	}

	return &protobuff.Transaction{
		SourceId:     sourceID.String(),
		DestId:       destID.String(),
		Amount:       tx.Amount,
		TickNumber:   tx.Tick,
		InputType:    uint32(tx.InputType),
		InputSize:    uint32(tx.InputSize),
		InputHex:     hex.EncodeToString(tx.Input[:]),
		SignatureHex: hex.EncodeToString(tx.Signature[:]),
		TxId:         txID.String(),
	}, nil
}
