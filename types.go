package main

const (
	SignatureSize = 64
)

type QuorumDiff struct {
	SaltedResourceTestingDigest uint64
	SaltedSpectrumDigest        [32]byte
	SaltedUniverseDigest        [32]byte
	SaltedComputerDigest        [32]byte
	ExpectedNextTickTxDigest    [32]byte
	Signature                   [SignatureSize]byte
}
