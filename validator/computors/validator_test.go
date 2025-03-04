package computors

import (
	"bufio"
	"context"
	"encoding/base64"
	"github.com/qubic/go-node-connector/types"
	"github.com/qubic/go-schnorrq"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func GoSchnorrqVerify(_ context.Context, pubkey [32]byte, digest [32]byte, sig [64]byte) error {
	return schnorrq.Verify(pubkey, digest, sig)
}

func TestValidator_Validate(t *testing.T) {
	arbId := types.Identity("AFZPUAIYVPNUYGJRQVLUKOPPVLHAZQTGLYAAUUNBXFTVTAMSBKQBLEIEPCVJ")
	arbPubKey, err := arbId.ToPubKey(false)
	assert.NoError(t, err)

	signature := "LPxFlBBQJimhb1/+YcjVdt5J7KCK1BTXJGTRoQ3mGX6M+ZeR3vg0PKx4d3U072QXP3/0yzoS4FD3sv1QEhkYAA=="
	decodedSigSlice, err := base64.StdEncoding.DecodeString(signature)
	assert.NoError(t, err)

	comps := types.Computors{
		Epoch:     148,
		PubKeys:   readCompKeys(t),
		Signature: [64]byte(decodedSigSlice),
	}

	err = Validate(context.Background(), GoSchnorrqVerify, comps, arbPubKey)
	assert.NoError(t, err, "expected valid signature")

	notArbId := types.Identity("ARBITEEEESZOQGTWOTQAFKWNQDJBRDVMKOFMFEMRMEELXYWYNTCPKQEDLBWH")
	notArbKey, err := notArbId.ToPubKey(false)
	assert.NoError(t, err)

	err = Validate(context.Background(), GoSchnorrqVerify, comps, notArbKey)
	assert.Error(t, err, "expected invalid signature")
}

func readCompKeys(t *testing.T) [676][32]byte {
	f, err := os.Open("testdata/computor-pub-keys.txt")
	assert.NoError(t, err)
	defer func(f *os.File) {
		assert.NoError(t, f.Close())
	}(f)
	scanner := bufio.NewScanner(f)

	var keys = make([][32]byte, 676)
	index := 0
	for scanner.Scan() {
		decoded, err := base64.StdEncoding.DecodeString(scanner.Text())
		assert.NoError(t, err)
		keys[index] = [32]byte(decoded)
		index++
	}

	assert.NoError(t, scanner.Err())
	return [676][32]byte(keys)
}
