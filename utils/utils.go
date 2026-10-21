package utils

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/cloudflare/circl/xof/k12"
)

type SigVerifierFunc = func(ctx context.Context, pubkey [32]byte, digest [32]byte, sig [64]byte) error

func K12Hash(data []byte) ([32]byte, error) {
	h := k12.NewDraft10([]byte{}) // Using K12 for hashing, equivalent to KangarooTwelve(temp, 96, h, 64).
	_, err := h.Write(data)
	if err != nil {
		return [32]byte{}, fmt.Errorf("k12 hashing: %w", err)
	}

	var out [32]byte
	_, err = h.Read(out[:])
	if err != nil {
		return [32]byte{}, fmt.Errorf("reading k12 digest: %w", err)
	}

	return out, nil
}

func BinarySerialize(data interface{}) ([]byte, error) {
	if data == nil {
		return nil, nil
	}

	var buff bytes.Buffer
	err := binary.Write(&buff, binary.LittleEndian, data)
	if err != nil {
		return nil, fmt.Errorf("writing data to buff: %w", err)
	}

	return buff.Bytes(), nil
}
