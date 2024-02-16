package quorum

import (
	"context"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-node-connector/types"
	"log"
)

func Validate(ctx context.Context, quorumVotes types.QuorumVotes, computors types.Computors) error {
	if len(quorumVotes) < types.MinimumQuorumVotes {
		return errors.New("not enough quorum votes")
	}

	log.Printf("Proceed to validate total quorum votes: %d\n", len(quorumVotes))
	if err := compareVotes(ctx, quorumVotes); err != nil {
		return errors.Wrap(err, "quorum votes are not the same between quorum computors")
	}

	log.Printf("Proceed to validate total quorum sigs: %d\n", len(quorumVotes))
	err := quorumTickSigVerify(ctx, quorumVotes, computors)
	if err != nil {
		return errors.Wrap(err, "quorum tick signature verification failed")
	}

	return nil
}

func compareVotes(ctx context.Context, quorumVotes types.QuorumVotes) error {
	firstTickData := quorumVotes[0]

	for i := 1; i < len(quorumVotes); i++ {
		if quorumVotes[i].Epoch != firstTickData.Epoch ||
					quorumVotes[i].Tick != firstTickData.Tick ||
					quorumVotes[i].Millisecond != firstTickData.Millisecond ||
					quorumVotes[i].Second != firstTickData.Second ||
					quorumVotes[i].Minute != firstTickData.Minute ||
					quorumVotes[i].Hour != firstTickData.Hour ||
					quorumVotes[i].Day != firstTickData.Day ||
					quorumVotes[i].Month != firstTickData.Month ||
					quorumVotes[i].Year != firstTickData.Year ||
					quorumVotes[i].PreviousResourceTestingDigest != firstTickData.PreviousResourceTestingDigest ||
					quorumVotes[i].PreviousComputerDigest != firstTickData.PreviousComputerDigest ||
					quorumVotes[i].PreviousSpectrumDigest != firstTickData.PreviousSpectrumDigest ||
					quorumVotes[i].PreviousUniverseDigest != firstTickData.PreviousUniverseDigest ||
					quorumVotes[i].TxDigest != firstTickData.TxDigest {
			return errors.New("quorum votes are not the same between quorum computors")
		}
	}

	return nil
}

func quorumTickSigVerify(ctx context.Context, quorumVotes types.QuorumVotes, computors types.Computors) error {
	var successVotes = 0
	failedIndexes := make([]uint16, 0, 0)
	failedIdentites := make([]string, 0, 0)
	//log.Printf("Proceed to validate total quorum votes: %d\n", len(quorumVotes))
	for _, quorumTickData := range quorumVotes {
		digest, err := getDigestFromQuorumTickData(quorumTickData)
		if err != nil {
			return errors.Wrap(err, "getting digest from tick data")
		}
		computorPubKey := computors.PubKeys[quorumTickData.ComputorIndex]
		if err := utils.FourQSigVerify(ctx, computorPubKey, digest, quorumTickData.Signature); err != nil {
			//return errors.Wrapf(err, "quorum tick signature verification failed for computor index: %d", quorumTickData.ComputorIndex)
			//log.Printf("Quorum tick signature verification failed for computor index: %d. Err: %s\n", quorumTickData.ComputorIndex, err.Error())
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
		//log.Printf("Validated vote for computor index: %d. Vote number %d\n", quorumTickData.ComputorIndex, successVotes)
	}

	//log.Printf("Validated total quorum votes: %d\n", successVotes)
	//log.Printf("Unvalidated total quorum votes: %d. List: %v, %v\n", len(failedIndexes), failedIndexes, failedIdentites)
	return nil
}

func getDigestFromQuorumTickData(data types.QuorumTickVote) ([32]byte, error) {
	// xor computor index with 8
	data.ComputorIndex ^= 3

	sData, err := utils.BinarySerialize(data)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "serializing data")
	}

	tickData := sData[:len(sData)-64]
	digest, err := utils.K12Hash(tickData)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "hashing tick data")
	}

	return digest, nil
}

func Store(ctx context.Context, store *store.PebbleStore, quorumVotes types.QuorumVotes) error {
	protoModel := qubicToProto(quorumVotes)

	err := store.SetQuorumTickData(ctx, uint64(protoModel.QuorumTickStructure.Epoch), protoModel)
	if err != nil {
		return errors.Wrap(err, "set quorum votes")
	}

	return nil
}


