package qchain

import (
	"github.com/stretchr/testify/require"
	"testing"
)

func TestQChain_Digest(t *testing.T) {
	qChain := QChain{
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
		PreviousSpectrumDigest:        [32]byte{},
		PreviousUniverseDigest:        [32]byte{},
		PreviousComputerDigest:        [32]byte{},
		TxDigest:                      [32]byte{},
	}
	_, err := qChain.Digest()
	require.NoError(t, err)
}
