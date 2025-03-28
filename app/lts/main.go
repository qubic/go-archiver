package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/ardanlabs/conf"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/lts/tx"
	"github.com/qubic/go-archiver/protobuff"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"net/http"
	"os"
	"time"
)

const prefix = "QUBIC_ARCHIVER"

func main() {
	if err := run(); err != nil {
		log.Fatalf("main: exited with error: %s", err.Error())
	}
}

func run() error {
	config := zap.NewProductionConfig()
	// this is just for sugar, to display a readable date instead of an epoch time
	config.EncoderConfig.EncodeTime = zapcore.TimeEncoderOfLayout(time.DateTime)

	logger, err := config.Build()
	if err != nil {
		fmt.Errorf("creating logger: %v", err)
	}
	defer logger.Sync()
	sLogger := logger.Sugar()

	var cfg struct {
		InternalStoreFolder       string        `conf:"default:store"`
		ArchiverHost              string        `conf:"default:127.0.0.1:6001"`
		ArchiverReadTimeout       time.Duration `conf:"default:10s"`
		ElasticSearchAddress      string        `conf:"default:http://127.0.0.1:9200"`
		ElasticSearchWriteTimeout time.Duration `conf:"default:5m"`
		BatchSize                 int           `conf:"default:10000"`
		NrWorkers                 int           `conf:"default:20"`
	}

	if err := conf.Parse(os.Args[1:], prefix, &cfg); err != nil {
		switch err {
		case conf.ErrHelpWanted:
			usage, err := conf.Usage(prefix, &cfg)
			if err != nil {
				return fmt.Errorf("generating config usage: %v", err)
			}
			fmt.Println(usage)
			return nil
		case conf.ErrVersionWanted:
			version, err := conf.VersionString(prefix, &cfg)
			if err != nil {
				return fmt.Errorf("generating config version: %v", err)
			}
			fmt.Println(version)
			return nil
		}
		return fmt.Errorf("parsing config: %v", err)
	}

	out, err := conf.String(&cfg)
	if err != nil {
		return fmt.Errorf("generating config for output: %v", err)
	}
	log.Printf("main: Config :\n%v\n", out)

	store, err := tx.NewProcessorStore(cfg.InternalStoreFolder)
	if err != nil {
		return fmt.Errorf("creating processor store: %v", err)
	}

	archiverConn, err := grpc.NewClient(cfg.ArchiverHost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return fmt.Errorf("creating archiver connection: %v", err)
	}

	archiverClient := protobuff.NewArchiveServiceClient(archiverConn)

	esInserter, err := NewElasticSearchTxsInserter(cfg.ElasticSearchAddress, "transactions", cfg.ElasticSearchWriteTimeout)
	if err != nil {
		return fmt.Errorf("creating elasticsearch tx inserter: %v", err)
	}

	proc, err := tx.NewProcessor(store, archiverClient, esInserter, cfg.BatchSize, sLogger, cfg.ArchiverReadTimeout, cfg.ElasticSearchWriteTimeout)
	if err != nil {
		return fmt.Errorf("creating processor: %v", err)
	}

	err = proc.Start(cfg.NrWorkers)
	if err != nil {
		return fmt.Errorf("starting processor: %v", err)
	}

	return nil
}

type ElasticSearchTxsInserter struct {
	port     string
	index    string
	esClient *elasticsearch.Client
}

func NewElasticSearchTxsInserter(address, index string, timeout time.Duration) (*ElasticSearchTxsInserter, error) {
	cfg := elasticsearch.Config{
		Addresses: []string{address},
		Transport: &http.Transport{
			MaxIdleConnsPerHost:   10,
			ResponseHeaderTimeout: timeout,
		},
	}

	esClient, err := elasticsearch.NewClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("creating elasticsearch client: %v", err)
	}

	return &ElasticSearchTxsInserter{
		index:    index,
		esClient: esClient,
	}, nil
}

func (es *ElasticSearchTxsInserter) PushSingleTx(ctx context.Context, tx tx.Tx) error {
	return errors.New("not implemented")
}

func (es *ElasticSearchTxsInserter) PushMultipleTx(ctx context.Context, txs []tx.Tx) error {
	var buf bytes.Buffer

	for _, tx := range txs {
		// Metadata line for each document
		meta := []byte(fmt.Sprintf(`{ "index": { "_index": "%s", "_id": "%s" } }%s`, es.index, tx.TxID, "\n"))
		buf.Write(meta)

		// Serialize the transaction to JSON
		data, err := json.Marshal(tx)
		if err != nil {
			return fmt.Errorf("error serializing transaction: %w", err)
		}
		buf.Write(data)
		buf.Write([]byte("\n")) // Add a newline between documents
	}

	// Send the bulk request
	res, err := es.esClient.Bulk(bytes.NewReader(buf.Bytes()), es.esClient.Bulk.WithRefresh("true"))
	if err != nil {
		return fmt.Errorf("bulk request failed: %w", err)
	}
	defer res.Body.Close()

	// Check response for errors
	if res.IsError() {
		return fmt.Errorf("bulk request error: %s", res.String())
	}

	return nil
}
