package utils

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"github.com/cloudflare/circl/xof/k12"
	"github.com/pkg/errors"
	"os/exec"
)

var debug = false

type SigVerifierFunc = func(ctx context.Context, pubkey [32]byte, digest [32]byte, sig [64]byte) error

func FourQSigVerify(ctx context.Context, pubkey [32]byte, digest [32]byte, sig [64]byte) error {
	pubKeyHex := hex.EncodeToString(pubkey[:])
	digestHex := hex.EncodeToString(digest[:])
	sigHex := hex.EncodeToString(sig[:])

	if debug {
		//log.Println("Skipping fourq_verify. Debug mode enabled.")
		return nil
	}

	cmd := exec.Command("./fourq_verify", pubKeyHex, digestHex, sigHex)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "running fourq_verify cmd: %s", cmd.String())
	}

	return nil
}

func K12Hash(data []byte) ([32]byte, error) {
	h := k12.NewDraft10([]byte{}) // Using K12 for hashing, equivalent to KangarooTwelve(temp, 96, h, 64).
	_, err := h.Write(data)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "k12 hashing")
	}

	var out [32]byte
	_, err = h.Read(out[:])
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "reading k12 digest")
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
		return nil, errors.Wrap(err, "writing data to buff")
	}

	return buff.Bytes(), nil
}
