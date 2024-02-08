package quorum

import (
	"context"
	"github.com/qubic/go-node-connector/types"
	"github.com/stretchr/testify/require"
	"testing"
)

// Sufficient quorum votes.
// Insufficient quorum votes.
// Mismatched votes in various fields.
func TestValidateVotes(t *testing.T) {
	originalData := types.ResponseQuorumTickData{
		QuorumData: []types.QuorumTickData{
			{
				ComputorIndex:                   1,
				Epoch:                           1,
				Tick:                            100,
				Millisecond:                     500,
				Second:                          30,
				Minute:                          15,
				Hour:                            12,
				Day:                             28,
				Month:                           2,
				Year:                            20,
				PreviousResourceTestingDigest:   1234567890123456789,
				SaltedResourceTestingDigest:     9876543210987654321,
				PreviousSpectrumDigest:          nonEmptyDigest(1),
				PreviousUniverseDigest:          nonEmptyDigest(2),
				PreviousComputerDigest:          nonEmptyDigest(3),
				SaltedSpectrumDigest:            nonEmptyDigest(4),
				SaltedUniverseDigest:            nonEmptyDigest(5),
				SaltedComputerDigest:            nonEmptyDigest(6),
				TxDigest:                        nonEmptyDigest(7),
				ExpectedNextTickTxDigest:        nonEmptyDigest(8),
				Signature:                       [64]byte{},
			},
			// Duplicate the first entry for a valid comparison base
			{
				ComputorIndex:                   2,
				Epoch:                           1,
				Tick:                            100,
				Millisecond:                     500,
				Second:                          30,
				Minute:                          15,
				Hour:                            12,
				Day:                             28,
				Month:                           2,
				Year:                            20,
				PreviousResourceTestingDigest:   1234567890123456789,
				SaltedResourceTestingDigest:     9876543210987654321,
				PreviousSpectrumDigest:          nonEmptyDigest(1),
				PreviousUniverseDigest:          nonEmptyDigest(2),
				PreviousComputerDigest:          nonEmptyDigest(3),
				SaltedSpectrumDigest:            nonEmptyDigest(4),
				SaltedUniverseDigest:            nonEmptyDigest(5),
				SaltedComputerDigest:            nonEmptyDigest(6),
				TxDigest:                        nonEmptyDigest(7),
				ExpectedNextTickTxDigest:        nonEmptyDigest(8),
				Signature:                       [64]byte{},
			},
		},
	}

	err := Validate(context.Background(), originalData, types.Computors{})
	require.ErrorContains(t, err, "not enough quorum votes")

	cases := []struct {
		name        string
		modify      func(*types.ResponseQuorumTickData)
		expectError bool
	}{
		{
			name: "valid data",
			modify: func(data *types.ResponseQuorumTickData) {},
			expectError: false,
		},
		// Test cases for mismatches in each field
		{
			name: "mismatched Second",
			modify: func(data *types.ResponseQuorumTickData) {
				data.QuorumData[1].Second += 1
			},
			expectError: true,
		},
		{
			name: "mismatched Minute",
			modify: func(data *types.ResponseQuorumTickData) {
				data.QuorumData[1].Minute += 1
			},
			expectError: true,
		},
		{
			name: "mismatched Hour",
			modify: func(data *types.ResponseQuorumTickData) {
				data.QuorumData[1].Hour += 1
			},
			expectError: true,
		},
		{
			name: "mismatched Day",
			modify: func(data *types.ResponseQuorumTickData) {
				data.QuorumData[1].Day += 1
			},
			expectError: true,
		},
		{
			name: "mismatched Month",
			modify: func(data *types.ResponseQuorumTickData) {
				data.QuorumData[1].Month += 1
			},
			expectError: true,
		},
		{
			name: "mismatched Year",
			modify: func(data *types.ResponseQuorumTickData) {
				data.QuorumData[1].Year += 1
			},
			expectError: true,
		},
		{
			name: "mismatched PreviousResourceTestingDigest",
			modify: func(data *types.ResponseQuorumTickData) {
				data.QuorumData[1].PreviousResourceTestingDigest += 1
			},
			expectError: true,
		},
		{
			name: "mismatched PreviousSpectrumDigest",
			modify: func(data *types.ResponseQuorumTickData) {
				data.QuorumData[1].PreviousSpectrumDigest = nonEmptyDigest(2)
			},
			expectError: true,
		},
		{
			name: "mismatched PreviousUniverseDigest",
			modify: func(data *types.ResponseQuorumTickData) {
				data.QuorumData[1].PreviousUniverseDigest[0] += 1
			},
			expectError: true,
		},
		{
			name: "mismatched PreviousComputerDigest",
			modify: func(data *types.ResponseQuorumTickData) {
				data.QuorumData[1].PreviousComputerDigest[0] += 1
			},
			expectError: true,
		},
		{
			name: "mismatched TxDigest",
			modify: func(data *types.ResponseQuorumTickData) {
				data.QuorumData[1].TxDigest[0] += 1
			},
			expectError: true,
		},
		{
			name: "mismatched SaltedSpectrumDigest",
			modify: func(data *types.ResponseQuorumTickData) {
				data.QuorumData[1].SaltedSpectrumDigest[0] += 1
			},
			expectError: false,
		},
		{
			name: "mismatched SaltedUniverseDigest",
			modify: func(data *types.ResponseQuorumTickData) {
				data.QuorumData[1].SaltedUniverseDigest[0] += 1
			},
			expectError: false,
		},
		{
			name: "mismatched SaltedComputerDigest",
			modify: func(data *types.ResponseQuorumTickData) {
				data.QuorumData[1].SaltedComputerDigest[0] += 1
			},
			expectError: false,
		},
		{
			name: "mismatched SaltedResourceTestingDigest",
			modify: func(data *types.ResponseQuorumTickData) {
				data.QuorumData[1].SaltedResourceTestingDigest += 1
			},
			expectError: false,
		},
		{
			name: "mismatched ExpectedNextTickTxDigest",
			modify: func(data *types.ResponseQuorumTickData) {
				data.QuorumData[1].ExpectedNextTickTxDigest[0] += 1
			},
			expectError: false,
		},
		{
			name: "mismatched Signature",
			modify: func(data *types.ResponseQuorumTickData) {
				data.QuorumData[1].Signature[0] += 1
			},
			expectError: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Make a deep copy of the original data to avoid mutation between tests
			dataCopy := deepCopy(originalData)
			tc.modify(&dataCopy)

			err := compareVotes(context.Background(), dataCopy)
			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func deepCopy(data types.ResponseQuorumTickData) types.ResponseQuorumTickData {
	cp := types.ResponseQuorumTickData{
		QuorumData: make([]types.QuorumTickData, len(data.QuorumData)),
	}

	for i, qd := range data.QuorumData {
		cp.QuorumData[i] = qd // Assuming QuorumTickData contains only value types; otherwise, further deep copy needed.
		// For [32]byte fields, direct assignment here is fine since arrays (unlike slices) are value types and copied.
	}

	return cp
}
