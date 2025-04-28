package quorum

import (
	"context"
	"github.com/qubic/go-node-connector/types"
	"github.com/stretchr/testify/require"
	"testing"
)

var mockSigVerifierFunc = func(ctx context.Context, pubkey [32]byte, digest [32]byte, sig [64]byte) error {
	return nil
}

// Sufficient quorum votes.
// Insufficient quorum votes.
// Mismatched votes in various fields.
func TestValidateVotes(t *testing.T) {
	originalData := types.QuorumVotes{
		types.QuorumTickVote{
			ComputorIndex:                 1,
			Epoch:                         1,
			Tick:                          100,
			Millisecond:                   500,
			Second:                        30,
			Minute:                        15,
			Hour:                          12,
			Day:                           28,
			Month:                         2,
			Year:                          20,
			PreviousResourceTestingDigest: 1234567890,
			SaltedResourceTestingDigest:   876543210,
			PreviousSpectrumDigest:        nonEmptyDigest(1),
			PreviousUniverseDigest:        nonEmptyDigest(2),
			PreviousComputerDigest:        nonEmptyDigest(3),
			SaltedSpectrumDigest:          nonEmptyDigest(4),
			SaltedUniverseDigest:          nonEmptyDigest(5),
			SaltedComputerDigest:          nonEmptyDigest(6),
			TxDigest:                      nonEmptyDigest(7),
			ExpectedNextTickTxDigest:      nonEmptyDigest(8),
			Signature:                     [64]byte{},
		},
		// Duplicate the first entry for a valid comparison base
		{
			ComputorIndex:                 2,
			Epoch:                         1,
			Tick:                          100,
			Millisecond:                   500,
			Second:                        30,
			Minute:                        15,
			Hour:                          12,
			Day:                           28,
			Month:                         2,
			Year:                          20,
			PreviousResourceTestingDigest: 1234567890,
			SaltedResourceTestingDigest:   3543210321,
			PreviousSpectrumDigest:        nonEmptyDigest(1),
			PreviousUniverseDigest:        nonEmptyDigest(2),
			PreviousComputerDigest:        nonEmptyDigest(3),
			SaltedSpectrumDigest:          nonEmptyDigest(4),
			SaltedUniverseDigest:          nonEmptyDigest(5),
			SaltedComputerDigest:          nonEmptyDigest(6),
			TxDigest:                      nonEmptyDigest(7),
			ExpectedNextTickTxDigest:      nonEmptyDigest(8),
			Signature:                     [64]byte{},
		},
	}

	_, err := Validate(context.Background(), mockSigVerifierFunc, originalData, types.Computors{})
	require.ErrorContains(t, err, "not enough quorum votes")

	cases := []struct {
		name          string
		modify        func(votes *types.QuorumVotes)
		expectedVotes int
		expectError   bool
	}{
		{
			name:          "valid data",
			modify:        func(votes *types.QuorumVotes) {},
			expectedVotes: 2,
		},
		// Test cases for mismatches in each field
		{
			name: "mismatched Second",
			modify: func(votes *types.QuorumVotes) {
				valueVotes := *votes
				valueVotes[1].Second += 1
				*votes = valueVotes
			},
			expectedVotes: 1,
		},
		{
			name: "mismatched Minute",
			modify: func(votes *types.QuorumVotes) {
				valueVotes := *votes
				valueVotes[1].Minute += 1
				*votes = valueVotes
			},
			expectedVotes: 1,
		},
		{
			name: "mismatched Hour",
			modify: func(votes *types.QuorumVotes) {
				valueVotes := *votes
				valueVotes[1].Hour += 1
				*votes = valueVotes
			},
			expectedVotes: 1,
		},
		{
			name: "mismatched Day",
			modify: func(votes *types.QuorumVotes) {
				valueVotes := *votes
				valueVotes[1].Day += 1
				*votes = valueVotes
			},
			expectedVotes: 1,
		},
		{
			name: "mismatched Month",
			modify: func(votes *types.QuorumVotes) {
				valueVotes := *votes
				valueVotes[1].Month += 1
				*votes = valueVotes
			},
			expectedVotes: 1,
		},
		{
			name: "mismatched Year",
			modify: func(votes *types.QuorumVotes) {
				valueVotes := *votes
				valueVotes[1].Year += 1
				*votes = valueVotes
			},
			expectedVotes: 1,
		},
		{
			name: "mismatched PreviousResourceTestingDigest",
			modify: func(votes *types.QuorumVotes) {
				valueVotes := *votes
				valueVotes[1].PreviousResourceTestingDigest += 1
				*votes = valueVotes
			},
			expectedVotes: 1,
		},
		{
			name: "mismatched PreviousSpectrumDigest",
			modify: func(votes *types.QuorumVotes) {
				valueVotes := *votes
				valueVotes[1].PreviousSpectrumDigest = nonEmptyDigest(2)
				*votes = valueVotes
			},
			expectedVotes: 1,
		},
		{
			name: "mismatched PreviousUniverseDigest",
			modify: func(votes *types.QuorumVotes) {
				valueVotes := *votes
				valueVotes[1].PreviousUniverseDigest[0] += 1
				*votes = valueVotes
			},
			expectedVotes: 1,
		},
		{
			name: "mismatched PreviousComputerDigest",
			modify: func(votes *types.QuorumVotes) {
				valueVotes := *votes
				valueVotes[1].PreviousComputerDigest[0] += 1
				*votes = valueVotes
			},
			expectedVotes: 1,
		},
		{
			name: "mismatched TxDigest",
			modify: func(votes *types.QuorumVotes) {
				valueVotes := *votes
				valueVotes[1].TxDigest[0] += 1
				*votes = valueVotes
			},
			expectedVotes: 1,
		},
		{
			name: "mismatched SaltedSpectrumDigest",
			modify: func(votes *types.QuorumVotes) {
				valueVotes := *votes
				valueVotes[1].SaltedSpectrumDigest[0] += 1
				*votes = valueVotes
			},
			expectedVotes: 2,
		},
		{
			name: "mismatched SaltedUniverseDigest",
			modify: func(votes *types.QuorumVotes) {
				valueVotes := *votes
				valueVotes[1].SaltedUniverseDigest[0] += 1
				*votes = valueVotes
			},
			expectedVotes: 2,
		},
		{
			name: "mismatched SaltedComputerDigest",
			modify: func(votes *types.QuorumVotes) {
				valueVotes := *votes
				valueVotes[1].SaltedComputerDigest[0] += 1
				*votes = valueVotes
			},
			expectedVotes: 2,
		},
		{
			name: "mismatched SaltedResourceTestingDigest",
			modify: func(votes *types.QuorumVotes) {
				valueVotes := *votes
				valueVotes[1].SaltedResourceTestingDigest += 1
				*votes = valueVotes
			},
			expectedVotes: 2,
		},
		{
			name: "mismatched ExpectedNextTickTxDigest",
			modify: func(votes *types.QuorumVotes) {
				valueVotes := *votes
				valueVotes[1].ExpectedNextTickTxDigest[0] += 1
				*votes = valueVotes
			},
			expectedVotes: 2,
		},
		{
			name: "mismatched Signature",
			modify: func(votes *types.QuorumVotes) {
				valueVotes := *votes
				valueVotes[1].Signature[0] += 1
				*votes = valueVotes
			},
			expectedVotes: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Make a deep copy of the original data to avoid mutation between tests
			dataCopy := deepCopy(originalData)
			tc.modify(&dataCopy)

			alignedVotes, err := getAlignedVotes(dataCopy)
			require.NoError(t, err)
			require.Equal(t, len(alignedVotes), tc.expectedVotes)
		})
	}
}

func deepCopy(votes types.QuorumVotes) types.QuorumVotes {
	cp := make(types.QuorumVotes, len(votes))

	for i, qv := range votes {
		cp[i] = qv // Assuming QuorumTickData contains only value types; otherwise, further deep copy needed.
		// For [32]byte fields, direct assignment here is fine since arrays (unlike slices) are value types and copied.
	}

	return cp
}

func TestByteSwap(t *testing.T) {

	testData := []struct {
		name     string
		data     []byte
		expected []byte
	}{
		{
			name:     "Test_ByteSwap1",
			data:     []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
			expected: []byte{0x06, 0x05, 0x04, 0x03, 0x02, 0x01},
		},
	}

	for _, testCase := range testData {
		t.Run(testCase.name, func(t *testing.T) {

			got := swapBytes(testCase.data)
			require.Equal(t, testCase.expected, got)
		})
	}

}
