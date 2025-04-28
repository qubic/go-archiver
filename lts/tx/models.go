package tx

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"github.com/qubic/go-archiver/protobuff"
)

type Tx struct {
	TxID       string `json:"txID"`
	SourceID   string `json:"sourceID"`
	DestID     string `json:"destID"`
	Amount     int64  `json:"amount"`
	TickNumber uint32 `json:"tickNumber"`
	InputType  uint32 `json:"inputType"`
	InputSize  uint32 `json:"inputSize"`
	Input      string `json:"input"`
	Signature  string `json:"signature"`
	Timestamp  uint64 `json:"timestamp"`
	MoneyFlew  bool   `json:"moneyFlew"`
}

func ArchiveTxToLtsTx(archiveTx *protobuff.TransactionData) (Tx, error) {
	inputBytes, err := hex.DecodeString(archiveTx.Transaction.InputHex)
	if err != nil {
		return Tx{}, fmt.Errorf("decoding input hex: %v", err)
	}
	sigBytes, err := hex.DecodeString(archiveTx.Transaction.SignatureHex)
	if err != nil {
		return Tx{}, fmt.Errorf("decoding signature hex: %v", err)
	}

	return Tx{
		TxID:       archiveTx.Transaction.TxId,
		SourceID:   archiveTx.Transaction.SourceId,
		DestID:     archiveTx.Transaction.DestId,
		Amount:     archiveTx.Transaction.Amount,
		TickNumber: archiveTx.Transaction.TickNumber,
		InputType:  archiveTx.Transaction.InputType,
		InputSize:  archiveTx.Transaction.InputSize,
		Input:      base64.StdEncoding.EncodeToString(inputBytes),
		Signature:  base64.StdEncoding.EncodeToString(sigBytes),
		Timestamp:  archiveTx.Timestamp,
		MoneyFlew:  archiveTx.MoneyFlew,
	}, nil
}
