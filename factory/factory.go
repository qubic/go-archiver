package factory

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	qubic "github.com/qubic/go-node-connector"
	"io"
	"math/rand"
	"net/http"
	"time"
)

type QubicConnectionFactory struct {
	nodeFetcherTimeout time.Duration
	nodeFetcherUrl     string
}

func NewQubicConnection(nodeFetcherTimeout time.Duration, nodeFetcherUrl string) *QubicConnectionFactory {
	return &QubicConnectionFactory{nodeFetcherTimeout: nodeFetcherTimeout, nodeFetcherUrl: nodeFetcherUrl}
}

func (cf *QubicConnectionFactory) Connect() (interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cf.nodeFetcherTimeout)
	defer cancel()

	peer, err := cf.getNewRandomPeer(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "getting new random peer")
	}

	client, err := qubic.NewClient(ctx, peer, "21841")
	if err != nil {
		return nil, errors.Wrap(err, "creating qubic client")
	}

	fmt.Printf("connected to: %s\n", peer)
	return client, nil
}

func (cf *QubicConnectionFactory) Close(v interface{}) error { return v.(*qubic.Client).Close() }

type response struct {
	Peers       []string `json:"peers"`
	Length      int      `json:"length"`
	LastUpdated int64    `json:"last_updated"`
}

func (cf *QubicConnectionFactory) getNewRandomPeer(ctx context.Context) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cf.nodeFetcherUrl, nil)
	if err != nil {
		return "", errors.Wrap(err, "creating new request")
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", errors.Wrap(err, "getting peers from node fetcher")
	}

	var resp response
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", errors.Wrap(err, "reading response body")
	}

	err = json.Unmarshal(body, &resp)
	if err != nil {
		return "", errors.Wrap(err, "unmarshalling response")
	}

	peer := resp.Peers[rand.Intn(len(resp.Peers))]

	fmt.Printf("Got %d new peers. Selected random %s\n", len(resp.Peers), peer)

	return peer, nil
}
