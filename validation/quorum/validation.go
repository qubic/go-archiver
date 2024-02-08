package quorum

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/hex"
	"github.com/cloudflare/circl/xof/k12"
	"github.com/pkg/errors"
	"github.com/qubic/go-node-connector/types"
	"log"
	"os/exec"
)

func Validate(ctx context.Context, quorumTickData types.ResponseQuorumTickData, computors types.Computors) error {
	if len(quorumTickData.QuorumData) < types.MinimumQuorumVotes {
		return errors.New("not enough quorum votes")
	}

	log.Printf("Proceed to validate total quorum votes: %d\n", len(quorumTickData.QuorumData))
	if err := compareVotes(ctx, quorumTickData); err != nil {
		return errors.Wrap(err, "quorum votes are not the same between quorum computors")
	}

	err := quorumTickSigVerify(ctx, quorumTickData, computors)
	if err != nil {
		return errors.Wrap(err, "quorum tick signature verification failed")
	}

	return nil
}

func compareVotes(ctx context.Context, data types.ResponseQuorumTickData) error {
	firstTickData := data.QuorumData[0]

	for i := 1; i < len(data.QuorumData); i++ {
		if data.QuorumData[i].Epoch != firstTickData.Epoch ||
					data.QuorumData[i].Tick != firstTickData.Tick ||
					data.QuorumData[i].Millisecond != firstTickData.Millisecond ||
					data.QuorumData[i].Second != firstTickData.Second ||
					data.QuorumData[i].Minute != firstTickData.Minute ||
					data.QuorumData[i].Hour != firstTickData.Hour ||
					data.QuorumData[i].Day != firstTickData.Day ||
					data.QuorumData[i].Month != firstTickData.Month ||
					data.QuorumData[i].Year != firstTickData.Year ||
					data.QuorumData[i].PreviousResourceTestingDigest != firstTickData.PreviousResourceTestingDigest ||
					data.QuorumData[i].PreviousComputerDigest != firstTickData.PreviousComputerDigest ||
					data.QuorumData[i].PreviousSpectrumDigest != firstTickData.PreviousSpectrumDigest ||
					data.QuorumData[i].PreviousUniverseDigest != firstTickData.PreviousUniverseDigest ||
					data.QuorumData[i].TxDigest != firstTickData.TxDigest {
			return errors.New("quorum votes are not the same between quorum computors")
		}
	}

	return nil
}

func quorumTickSigVerify(ctx context.Context, data types.ResponseQuorumTickData, computors types.Computors) error {
	var successVotes = 0
	failedIndexes := make([]uint16, 0, 0)
	failedIdentites := make([]string, 0, 0)
	log.Printf("Proceed to validate total quorum votes: %d\n", len(data.QuorumData))
	for _, quorumTickData := range data.QuorumData {
		digest, err := getDigestFromTickData(quorumTickData)
		if err != nil {
			return errors.Wrap(err, "getting digest from tick data")
		}
		computorPubKey := computors.PubKeys[quorumTickData.ComputorIndex]
		if err := verify(ctx, computorPubKey, digest, quorumTickData.Signature); err != nil {
			//return errors.Wrapf(err, "quorum tick signature verification failed for computor index: %d", quorumTickData.ComputorIndex)
			log.Printf("Quorum tick signature verification failed for computor index: %d\n", quorumTickData.ComputorIndex)
			failedIndexes = append(failedIndexes, quorumTickData.ComputorIndex)
			var badComputor types.Identity
			badComputor, err = badComputor.FromPubKey(computorPubKey, false)
			if err != nil {
				return errors.Wrap(err, "getting bad computor")
			}
			failedIdentites = append(failedIdentites, string(badComputor))
			continue
		}
		successVotes +=1
		log.Printf("Validated vote for computor index: %d. Vote number %d\n", quorumTickData.ComputorIndex, successVotes)
	}

	log.Printf("Validated total quorum votes: %d\n", successVotes)
	log.Printf("Unvalidated total quorum votes: %d. List: %v, %v\n", len(failedIndexes), failedIndexes, failedIdentites)
	return nil
}

func getDigestFromTickData(data types.QuorumTickData) ([32]byte, error) {
	// xor computor index with 8
	data.ComputorIndex ^= 3

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

func verify(ctx context.Context, pubkey [32]byte, digest [32]byte, sig [64]byte) error {
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
