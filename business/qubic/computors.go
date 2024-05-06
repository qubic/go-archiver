package qubic

import (
	"context"
	"encoding/hex"
	"github.com/pkg/errors"
	pb "github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-node-connector/types"
)

type ComputorsFetcher interface {
	GetComputors(ctx context.Context) (types.Computors, error)
}

func GetComputors(ctx context.Context, fetcher ComputorsFetcher) (*pb.Computors, error) {
	qubicComps, err := fetcher.GetComputors(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "fetching computors")
	}

	converter := computorsProtoConverter{comps: qubicComps}
	comps, err := converter.toProto()
	if err != nil {
		return nil, errors.Wrap(err, "converting computors to proto")
	}

	return comps, nil
}

type computorsProtoConverter struct {
	comps types.Computors
}

func (c *computorsProtoConverter) toProto() (*pb.Computors, error) {
	identities, err := pubKeysToIdentities(c.comps.PubKeys)
	if err != nil {
		return nil, errors.Wrap(err, "converting pubKeys to identities")
	}

	digest, err := c.getDigest()
	if err != nil {
		return nil, errors.Wrap(err, "creating computors digest")
	}

	return &pb.Computors{
		Epoch:        uint32(c.comps.Epoch),
		Identities:   identities,
		SignatureHex: hex.EncodeToString(c.comps.Signature[:]),
		DigestHex:    hex.EncodeToString(digest[:]),
	}, nil
}

func pubKeysToIdentities(pubKeys [types.NumberOfComputors][32]byte) ([]string, error) {
	identities := make([]string, 0)
	for _, identity := range pubKeys {
		if identity == [32]byte{} {
			continue
		}
		var id types.Identity
		id, err := id.FromPubKey(identity, false)
		if err != nil {
			return nil, errors.Wrapf(err, "getting identity from pubKey hex %s", hex.EncodeToString(identity[:]))
		}
		identities = append(identities, id.String())
	}
	return identities, nil
}

func (c *computorsProtoConverter) getDigest() ([32]byte, error) {
	serialized, err := utils.BinarySerialize(c.comps)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "serializing data")
	}

	// remove signature from computors data
	computorsData := serialized[:len(serialized)-64]
	digest, err := utils.K12Hash(computorsData)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "hashing computors data")
	}

	return digest, nil
}
