package quorum

import (
	"encoding/hex"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
	"testing"
	"time"
)

func TestQubicModelToProto(t *testing.T) {
	// Setup mock input data
	mockData := types.QuorumVotes{
		types.QuorumTickVote{
			ComputorIndex:                 1,
			Epoch:                         2,
			Tick:                          3,
			Year:                          21,
			Month:                         1,
			Day:                           2,
			Hour:                          3,
			Minute:                        4,
			Second:                        5,
			Millisecond:                   600,
			PreviousResourceTestingDigest: 123456789,
			PreviousSpectrumDigest:        generateDigest("PreviousSpectrum"),
			PreviousUniverseDigest:        generateDigest("PreviousUniverse"),
			PreviousComputerDigest:        generateDigest("PreviousComputer"),
			SaltedResourceTestingDigest:   987654321,
			SaltedSpectrumDigest:          generateDigest("SaltedSpectrum"),
			SaltedUniverseDigest:          generateDigest("SaltedUniverse"),
			SaltedComputerDigest:          generateDigest("SaltedComputer"),
			TxDigest:                      [32]byte{0xc3},
			ExpectedNextTickTxDigest:      [32]byte{0x1b},
			Signature:                     [64]byte{0x4c},
		},
		{
			ComputorIndex:                 3,
			Epoch:                         2,
			Tick:                          3,
			Year:                          21,
			Month:                         1,
			Day:                           2,
			Hour:                          3,
			Minute:                        4,
			Second:                        5,
			Millisecond:                   600,
			PreviousResourceTestingDigest: 123456789,
			PreviousSpectrumDigest:        generateDigest("PreviousSpectrum"),
			PreviousUniverseDigest:        generateDigest("PreviousUniverse"),
			PreviousComputerDigest:        generateDigest("PreviousComputer"),
			SaltedResourceTestingDigest:   987651321,
			SaltedSpectrumDigest:          generateDigest("SaltedSpectrum22"),
			SaltedUniverseDigest:          generateDigest("SaltedUniverse22"),
			SaltedComputerDigest:          generateDigest("SaltedComputer11"),
			TxDigest:                      [32]byte{0xc3},
			ExpectedNextTickTxDigest:      [32]byte{0x1c},
			Signature:                     [64]byte{0x4d},
		},
		// Additional QuorumTickData could be added here
	}

	// Setup expected output data
	expectedProtoQuorumTickData := &protobuff.QuorumTickData{
		QuorumTickStructure: &protobuff.QuorumTickStructure{
			ComputorIndex:                1,
			Epoch:                        2,
			TickNumber:                   3,
			Timestamp:                    uint64(time.Date(2021, 1, 2, 3, 4, 5, 600*1000000, time.UTC).UnixMilli()),
			PrevResourceTestingDigestHex: convertUint64ToHex(123456789),
			PrevSpectrumDigestHex:        fillStringTo(64, hex.EncodeToString(mockData[0].PreviousSpectrumDigest[:])),
			PrevUniverseDigestHex:        fillStringTo(64, hex.EncodeToString(mockData[0].PreviousUniverseDigest[:])),
			PrevComputerDigestHex:        fillStringTo(64, hex.EncodeToString(mockData[0].PreviousComputerDigest[:])),
			TxDigestHex:                  fillStringTo(64, "c3"),
			// Populate the rest of the fields in the expected struct
		},
		QuorumDiffPerComputor: map[uint32]*protobuff.QuorumDiff{
			1: {
				SaltedResourceTestingDigestHex: convertUint64ToHex(987654321),
				SaltedSpectrumDigestHex:        fillStringTo(64, hex.EncodeToString(mockData[0].SaltedSpectrumDigest[:])),
				SaltedUniverseDigestHex:        fillStringTo(64, hex.EncodeToString(mockData[0].SaltedUniverseDigest[:])),
				SaltedComputerDigestHex:        fillStringTo(64, hex.EncodeToString(mockData[0].SaltedComputerDigest[:])),
				ExpectedNextTickTxDigestHex:    fillStringTo(64, "1b"),
				SignatureHex:                   fillStringTo(128, "4c"),
			},
			3: {
				SaltedResourceTestingDigestHex: convertUint64ToHex(987651321),
				SaltedSpectrumDigestHex:        fillStringTo(64, hex.EncodeToString(mockData[1].SaltedSpectrumDigest[:])),
				SaltedUniverseDigestHex:        fillStringTo(64, hex.EncodeToString(mockData[1].SaltedUniverseDigest[:])),
				SaltedComputerDigestHex:        fillStringTo(64, hex.EncodeToString(mockData[1].SaltedComputerDigest[:])),
				ExpectedNextTickTxDigestHex:    fillStringTo(64, "1c"),
				SignatureHex:                   fillStringTo(128, "4d"),
			},
			// Additional diffs for other ComputorIndex if needed
		},
	}

	// Invoke the function under test
	result := qubicToProto(mockData)

	if diff := cmp.Diff(expectedProtoQuorumTickData, result, cmpopts.IgnoreUnexported(protobuff.QuorumTickData{}, protobuff.QuorumTickStructure{}, protobuff.QuorumDiff{})); diff != "" {
		t.Errorf("Unexpected result: %v", diff)
	}
}

func generateDigest(s string) [32]byte {
	var digest [32]byte
	copy(digest[:], s)
	return digest
}

func nonEmptyDigest(input byte) [32]byte {
	var digest [32]byte
	for i := range digest {
		digest[i] = input
	}
	return digest
}

func fillStringTo(nrChars int, value string) string {
	for len(value) < nrChars {
		value = value + "0"
	}
	return value
}
