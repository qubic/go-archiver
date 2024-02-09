package quorum

import (
	"encoding/binary"
	"encoding/hex"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-node-connector/types"
	"time"
)

func qubicModelToProto(model *types.ResponseQuorumTickData) *protobuff.QuorumTickData {
	firstQuorumTickData := model.QuorumData[0]
	protoQuorumTickData := protobuff.QuorumTickData{
		QuorumTickStructure: qubicModelTickStructureToProto(firstQuorumTickData),
		QuorumDiffPerComputor: make(map[uint32]*protobuff.QuorumDiff),
	}

	for _, quorumTickData := range model.QuorumData {
		protoQuorumTickData.QuorumDiffPerComputor[uint32(quorumTickData.ComputorIndex)] = qubicModelDiffToProto(quorumTickData)
	}

	return &protoQuorumTickData
}

func qubicModelTickStructureToProto(model types.QuorumTickData) *protobuff.QuorumTickStructure {
	date := time.Date(2000 + int(model.Year), time.Month(model.Month), int(model.Day), int(model.Hour), int(model.Minute), int(model.Second), 0, time.UTC)
	timestamp := date.UnixMilli() + int64(model.Millisecond)
	protoQuorumTickStructure := protobuff.QuorumTickStructure{
		ComputorIndex:                uint32(model.ComputorIndex),
		Epoch:                        uint32(model.Epoch),
		TickNumber:                   model.Tick,
		Timestamp:                    uint64(timestamp),
		PrevResourceTestingDigestHex: convertUint64ToHex(model.PreviousResourceTestingDigest),
		PrevSpectrumDigestHex:        hex.EncodeToString(model.PreviousSpectrumDigest[:]),
		PrevUniverseDigestHex:        hex.EncodeToString(model.PreviousUniverseDigest[:]),
		PrevComputerDigestHex:        hex.EncodeToString(model.PreviousComputerDigest[:]),
		TxDigestHex:                  hex.EncodeToString(model.TxDigest[:]),
	}

	return &protoQuorumTickStructure
}

func qubicModelDiffToProto(model types.QuorumTickData) *protobuff.QuorumDiff {
	protoQuorumDiff := protobuff.QuorumDiff{
		SaltedResourceTestingDigestHex: convertUint64ToHex(model.SaltedResourceTestingDigest),
		SaltedSpectrumDigestHex:        hex.EncodeToString(model.SaltedSpectrumDigest[:]),
		SaltedUniverseDigestHex:        hex.EncodeToString(model.SaltedUniverseDigest[:]),
		SaltedComputerDigestHex:        hex.EncodeToString(model.SaltedComputerDigest[:]),
		ExpectedNextTickTxDigestHex:    hex.EncodeToString(model.ExpectedNextTickTxDigest[:]),
		SignatureHex:                   hex.EncodeToString(model.Signature[:]),
	}
	return &protoQuorumDiff
}

func convertUint64ToHex(value uint64) string {
	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, value)
	return hex.EncodeToString(b)
}

