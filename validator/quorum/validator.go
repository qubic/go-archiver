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
	alignedVotes, err := getAlignedVotes(quorumVotes)
	if err != nil {
		return errors.Wrap(err, "quorum votes are not the same between quorum computors")
	}

	if alignedVotes < types.MinimumQuorumVotes {
		return errors.Errorf("Not enough aligned quorum votes. Aligned votes: %d", alignedVotes)
	}

	log.Printf("Proceed to validate total quorum sigs: %d\n", len(quorumVotes))
	err = quorumTickSigVerify(ctx, quorumVotes, computors)
	if err != nil {
		return errors.Wrap(err, "quorum tick signature verification failed")
	}

	return nil
}

func compareVotes(ctx context.Context, quorumVotes types.QuorumVotes, minimumRequiredVotes int) error {
	alignedVotes, err := getAlignedVotes(quorumVotes)
	if err != nil {
		return errors.Wrap(err, "getting aligned votes")
	}

	if alignedVotes < minimumRequiredVotes {
		return errors.Errorf("Not enough aligned quorum votes. Aligned votes: %d", alignedVotes)
	}

	return nil
}

type vote struct {
	Epoch                         uint16
	Tick                          uint32
	Millisecond                   uint16
	Second                        uint8
	Minute                        uint8
	Hour                          uint8
	Day                           uint8
	Month                         uint8
	Year                          uint8
	PreviousResourceTestingDigest uint64
	PreviousSpectrumDigest        [32]byte
	PreviousUniverseDigest        [32]byte
	PreviousComputerDigest        [32]byte
	TxDigest                      [32]byte
}

func (v *vote) digest() ([32]byte, error) {
	b, err := utils.BinarySerialize(v)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "serializing vote")
	}

	digest, err := utils.K12Hash(b)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "hashing vote")
	}

	return digest, nil
}

func getAlignedVotes(quorumVotes types.QuorumVotes) (int, error) {
	votesHeatMap := make(map[[32]byte]int)
	for _, qv := range quorumVotes {
		v := vote{
			Epoch:                         qv.Epoch,
			Tick:                          qv.Tick,
			Millisecond:                   qv.Millisecond,
			Second:                        qv.Second,
			Minute:                        qv.Minute,
			Hour:                          qv.Hour,
			Day:                           qv.Day,
			Month:                         qv.Month,
			Year:                          qv.Year,
			PreviousResourceTestingDigest: qv.PreviousResourceTestingDigest,
			PreviousSpectrumDigest:        qv.PreviousSpectrumDigest,
			PreviousUniverseDigest:        qv.PreviousUniverseDigest,
			PreviousComputerDigest:        qv.PreviousComputerDigest,
			TxDigest:                      qv.TxDigest,
		}
		digest, err := v.digest()
		if err != nil {
			return 0, errors.Wrap(err, "getting digest")
		}
		if _, ok := votesHeatMap[digest]; !ok {
			votesHeatMap[digest] = 1
		} else {
			votesHeatMap[digest] += 1
		}
	}

	alignedVotes := 0
	for _, v := range votesHeatMap {
		if v > alignedVotes {
			alignedVotes = v
		}
	}

	return alignedVotes, nil
}

func votesCompareMapValidation(quorumVotes types.QuorumVotes) map[[32]byte]int {
	m := make(map[[32]byte]int)
	for _, quorumTickData := range quorumVotes {
		val, ok := m[quorumTickData.PreviousComputerDigest]
		if ok {
			m[quorumTickData.PreviousComputerDigest] = val + 1
		} else {
			m[quorumTickData.PreviousComputerDigest] = 1
		}
	}

	return m
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
		successVotes += 1
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

func Store(ctx context.Context, store *store.PebbleStore, tickNumber uint64, quorumVotes types.QuorumVotes) error {
	protoModel := qubicToProto(quorumVotes)

	err := store.SetQuorumTickData(ctx, tickNumber, protoModel)
	if err != nil {
		return errors.Wrap(err, "set quorum votes")
	}

	return nil
}
