package chain

import (
	"bytes"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-node-connector/types"
	"google.golang.org/protobuf/proto"
	"log"
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
	PreviousResourceTestingDigest uint64
	PreviousSpectrumDigest        [32]byte
	PreviousUniverseDigest        [32]byte
	PreviousComputerDigest        [32]byte
	TxDigest                      [32]byte

	PreviousTickChainDigest [32]byte
}

func (c *Chain) Digest() ([32]byte, error) {
	b, err := c.MarshallBinary()
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "serializing vote")
	}

	digest, err := utils.K12Hash(b)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "hashing vote")
	}

	return digest, nil
}

func (c *Chain) MarshallBinary() ([]byte, error) {
	b, err := utils.BinarySerialize(c)
	if err != nil {
		return nil, errors.Wrap(err, "serializing vote")
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
		return nil, errors.Wrap(err, "writing previousTickStoreDigest")
	}

	for i, tx := range s.ValidTxs {
		digest, err := tx.Digest()
		if err != nil {
			return nil, errors.Wrap(err, "marshalling tx")
		}

		log.Printf("tx index: %d marshalled binary hex: %x\n", i, digest)
		_, err = buff.Write(digest[:])
		if err != nil {
			return nil, errors.Wrap(err, "writing digest")
		}
	}

	b, err := proto.Marshal(s.TickTxsStatus)
	if err != nil {
		return nil, errors.Wrap(err, "marshalling tickTxsStatus")
	}

	log.Printf("marshalled proto hex: %x\n", b)

	_, err = buff.Write(b)
	if err != nil {
		return nil, errors.Wrap(err, "writing tickTxsStatus")
	}

	return buff.Bytes(), nil
}

func (s *Store) Digest() ([32]byte, error) {
	b, err := s.MarshallBinary()
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "serializing store")
	}

	log.Printf("marshalled binary hex: %x\n", b)

	digest, err := utils.K12Hash(b)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "hashing vote")
	}

	return digest, nil
}
