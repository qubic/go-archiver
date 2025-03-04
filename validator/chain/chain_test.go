package chain

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestChain_Digest(t *testing.T) {
	chain := Chain{
		Epoch:                         20,
		Tick:                          30,
		Millisecond:                   40,
		Second:                        50,
		Minute:                        60,
		Hour:                          70,
		Day:                           80,
		Month:                         90,
		Year:                          10,
		PreviousResourceTestingDigest: 13423432,
		PreviousTransactionBodyDigest: 23432431,
		PreviousSpectrumDigest:        [32]byte{},
		PreviousUniverseDigest:        [32]byte{},
		PreviousComputerDigest:        [32]byte{},
		TxDigest:                      [32]byte{},
	}
	_, err := chain.Digest()
	require.NoError(t, err)
}

func TestChain_MarshallBinary(t *testing.T) {
	chain := Chain{
		Epoch:                         20,
		Tick:                          30,
		Millisecond:                   40,
		Second:                        50,
		Minute:                        60,
		Hour:                          70,
		Day:                           80,
		Month:                         90,
		Year:                          10,
		PreviousResourceTestingDigest: 13423432,
		PreviousTransactionBodyDigest: 23432431,
		PreviousSpectrumDigest:        [32]byte{},
		PreviousUniverseDigest:        [32]byte{},
		PreviousComputerDigest:        [32]byte{},
		TxDigest:                      [32]byte{},
	}
	b, err := chain.MarshallBinary()
	require.NoError(t, err)
	require.Equal(t, len(b), 184)
}
