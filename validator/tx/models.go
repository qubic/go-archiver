package tx

import (
	"encoding/hex"
	"fmt"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/v2/types"
)

func qubicToProto(txs types.Transactions) ([]*protobuff.Transaction, error) {
	protoTxs := make([]*protobuff.Transaction, len(txs))
	for i, tx := range txs {
		txProto, err := txToProto(tx)
		if err != nil {
			return nil, fmt.Errorf("converting tx to proto: %w", err)
		}
		protoTxs[i] = txProto
	}

	return protoTxs, nil
}

func txToProto(tx types.Transaction) (*protobuff.Transaction, error) {
	digest, err := tx.Digest()
	if err != nil {
		return nil, fmt.Errorf("getting tx digest: %w", err)
	}
	var txID types.Identity
	txID, err = txID.FromPubKey(digest, true)
	if err != nil {
		return nil, fmt.Errorf("getting tx id: %w", err)
	}

	var sourceID types.Identity
	sourceID, err = sourceID.FromPubKey(tx.SourcePublicKey, false)
	if err != nil {
		return nil, fmt.Errorf("getting source id: %w", err)
	}

	var destID types.Identity
	destID, err = destID.FromPubKey(tx.DestinationPublicKey, false)
	if err != nil {
		return nil, fmt.Errorf("getting dest id: %w", err)
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
