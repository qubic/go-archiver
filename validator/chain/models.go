package chain

import (
	"bytes"
	"fmt"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-node-connector/v2/types"
	"google.golang.org/protobuf/proto"
	"sort"
)

type Chain struct {
	_                             uint16 //padding
	Epoch                         uint16
	Tick                          uint32
	Millisecond                   uint16
	Second                        uint8
	Minute                        uint8
	Hour                          uint8
	Day                           uint8
	Month                         uint8
	Year                          uint8
	PreviousResourceTestingDigest uint32
	PreviousTransactionBodyDigest uint32
	PreviousSpectrumDigest        [32]byte
	PreviousUniverseDigest        [32]byte
	PreviousComputerDigest        [32]byte
	TxDigest                      [32]byte

	PreviousTickChainDigest [32]byte
}

func (c *Chain) Digest() ([32]byte, error) {
	b, err := c.MarshallBinary()
	if err != nil {
		return [32]byte{}, fmt.Errorf("serializing vote: %w", err)
	}

	digest, err := utils.K12Hash(b)
	if err != nil {
		return [32]byte{}, fmt.Errorf("hashing vote: %w", err)
	}

	return digest, nil
}

func (c *Chain) MarshallBinary() ([]byte, error) {
	b, err := utils.BinarySerialize(c)
	if err != nil {
		return nil, fmt.Errorf("serializing vote: %w", err)
	}

	return b, nil
}

type Store struct {
	PreviousTickStoreDigest [32]byte
	ValidTxs                []types.Transaction
	TickTxsStatus           *protobuff.TickTransactionsStatus
}

func (s *Store) MarshallBinary() ([]byte, error) {
	var buff bytes.Buffer
	_, err := buff.Write(s.PreviousTickStoreDigest[:])
	if err != nil {
		return nil, fmt.Errorf("writing previousTickStoreDigest: %w", err)
	}

	digests := make([][32]byte, 0, len(s.ValidTxs))
	for _, tx := range s.ValidTxs {
		digest, err := tx.Digest()
		if err != nil {
			return nil, fmt.Errorf("marshalling tx: %w", err)
		}

		digests = append(digests, digest)
	}

	sortByteSlices(digests)

	for _, digest := range digests {
		_, err = buff.Write(digest[:])
		if err != nil {
			return nil, fmt.Errorf("writing digest: %w", err)
		}
	}

	b, err := proto.Marshal(s.TickTxsStatus)
	if err != nil {
		return nil, fmt.Errorf("marshalling tickTxsStatus: %w", err)
	}

	_, err = buff.Write(b)
	if err != nil {
		return nil, fmt.Errorf("writing tickTxsStatus: %w", err)
	}

	return buff.Bytes(), nil
}

func (s *Store) Digest() ([32]byte, error) {
	b, err := s.MarshallBinary()
	if err != nil {
		return [32]byte{}, fmt.Errorf("serializing store: %w", err)
	}

	digest, err := utils.K12Hash(b)
	if err != nil {
		return [32]byte{}, fmt.Errorf("hashing vote: %w", err)
	}

	return digest, nil
}

func sortByteSlices(slice [][32]byte) {
	sort.SliceStable(slice, func(i, j int) bool {
		return bytes.Compare(slice[i][:], slice[j][:]) == -1
	})
}
