package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"github.com/cloudflare/circl/xof/k12"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/validation/quorum"
	"github.com/qubic/go-node-connector/types"
	"go.uber.org/zap"
	"log"
	"os"
	"os/exec"
	"strconv"

	qubic "github.com/qubic/go-node-connector"
)

var nodePort = "21841"

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Please provide tick number and nodeIP")
	}

	ip := os.Args[1]

	tickNumber, err := strconv.ParseUint(os.Args[2], 10, 64)
	if err != nil {
		log.Fatalf("Parsing tick number. err: %s", err.Error())
	}

	err = run(ip, tickNumber)
	if err != nil {
		log.Fatalf(err.Error())
	}
}

func run(nodeIP string, tickNumber uint64) error {
	client, err := qubic.NewClient(context.Background(), nodeIP, nodePort)
	if err != nil {
		log.Fatalf("creating qubic sdk: err: %s", err.Error())
	}
	// releasing tcp connection related resources
	defer client.Close()

	logger, _ := zap.NewDevelopment()
	verifier := NewVerifier(client, store.NewPebbleStore(nil, logger))
	if err := verifier.Verify(context.Background(), tickNumber); err != nil {
		return errors.Wrap(err, "verifying tick")
	}

	return nil
}


func getDigestFromTickData(data types.TickData) ([32]byte, error) {
	// xor computor index with 8
	data.ComputorIndex ^= 8

	sData, err := serializeBinary(data)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "serializing data")
	}

	tickData := sData[:len(sData)-64]
	digest, err := hash(tickData)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "hashing tick data")
	}

	return digest, nil
}

func verify(pubkey [32]byte, digest [32]byte, sig [64]byte) error {
	pubKeyHex := hex.EncodeToString(pubkey[:])
	digestHex := hex.EncodeToString(digest[:])
	sigHex := hex.EncodeToString(sig[:])

	cmd := exec.Command("./fourq_verify", pubKeyHex, digestHex, sigHex)
	var out bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Run(); err != nil {
		return errors.Wrapf(err, "running fourq_verify cmd: %s", cmd.String())
	}

	return nil
}

func hash(data []byte) ([32]byte, error) {
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

func serializeBinary(data interface{}) ([]byte, error) {
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

type Verifier struct {
	qu *qubic.Client
	store *store.PebbleStore
}

func NewVerifier(qu *qubic.Client, store *store.PebbleStore) *Verifier {
	return &Verifier{qu: qu, store: store}
}

func (v *Verifier) Verify(ctx context.Context, tickNumber uint64) error {
	computors, err := v.qu.GetComputors(ctx)
	if err != nil {
		return errors.Wrap(err, "getting computors")
	}

	arbitratorID := types.Identity(types.ArbitratorIdentity)
	arbitratorPubKey, err := arbitratorID.ToPubKey()
	if err != nil {
		return errors.Wrap(err, "getting arbitrator pubkey")
	}

	err = validateComputors(computors.Computors, arbitratorPubKey)
	if err != nil {
		return errors.Wrap(err, "validating computors")
	}
	log.Println("Computors validated")

	quorumTickData, err := v.qu.GetQuorumTickData(ctx, uint32(tickNumber))
	if err != nil {
		return errors.Wrap(err, "getting quorum tick data")
	}

	err = quorum.Validate(ctx, quorumTickData, computors.Computors)
	if err != nil {
		return errors.Wrap(err, "validating quorum")
	}

	log.Println("Quorum validated")

	//tickData, err := v.qu.GetTickData(ctx, uint32(tickNumber))
	//if err != nil {
	//	return errors.Wrap(err, "getting tick data")
	//}
	//computorPubKey := computors.Computors.PubKeys[tickData.ComputorIndex]
	//
	//digest, err := getDigestFromTickData(tickData)
	//
	//// verify tick signature
	//err = verify(computorPubKey, digest, tickData.Signature)
	//if err != nil {
	//	return errors.Wrap(err, "verifying tick signature")
	//}

	return nil
}

func validateComputors(computors types.Computors, arbitratorPubKey [32]byte) error {
	digest, err := getDigestFromComputors(computors)
	if err != nil {
		return errors.Wrap(err, "getting digest from computors")
	}

	err = verify(arbitratorPubKey, digest, computors.Signature)
	if err != nil {
		return errors.Wrap(err, "verifying computors signature")
	}

	return nil
}

func getDigestFromComputors(data types.Computors) ([32]byte, error) {
	sData, err := serializeBinary(data)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "serializing data")
	}

	// remove signature from computors data
	computorsData := sData[:len(sData)-64]
	digest, err := hash(computorsData)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "hashing computors data")
	}

	return digest, nil
}
