package validator

import (
	pb "github.com/qubic/go-archiver/protobuff"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"
	"testing"
)

func TestProto(t *testing.T) {
	//test if using proto.Marshal with pb.OldComputors and then Unmarshal to pb.Computors works
	old := pb.OldComputorsList{
		Computors: []*pb.OldComputors{
			{
				Epoch:        1,
				Identities:   []string{"111", "111"},
				SignatureHex: "1111",
			},
			{
				Epoch:        2,
				Identities:   []string{"bbb", "222"},
				SignatureHex: "2222",
			},
		},
	}

	serialized, err := proto.Marshal(&old)
	require.NoError(t, err)
	var newComps pb.ComputorsList
	err = proto.Unmarshal(serialized, &newComps)
	require.NoError(t, err)
}
