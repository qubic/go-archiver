package chain

import (
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/utils"
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
