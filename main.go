package main

import (
	"fmt"
	"github.com/ardanlabs/conf"
	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/processor"
	"github.com/qubic/go-archiver/rpc"
	"github.com/qubic/go-archiver/store"
	qubic "github.com/qubic/go-node-connector"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"
)

const prefix = "QUBIC_ARCHIVER"

func main() {
	if err := run(); err != nil {
		log.Fatalf("main: exited with error: %s", err.Error())
	}
}

func run() error {
	var cfg struct {
		Server struct {
			ReadTimeout       time.Duration `conf:"default:5s"`
			WriteTimeout      time.Duration `conf:"default:5s"`
			ShutdownTimeout   time.Duration `conf:"default:5s"`
			HttpHost          string        `conf:"default:0.0.0.0:8000"`
			GrpcHost          string        `conf:"default:0.0.0.0:8001"`
			NodeSyncThreshold int           `conf:"default:2"`
			ChainTickFetchUrl string        `conf:"default:http://127.0.0.1:8080/chain-tick"`
		}
		Pool struct {
			NodeFetcherUrl     string        `conf:"default:http://127.0.0.1:8080/peers"`
			NodeFetcherTimeout time.Duration `conf:"default:2s"`
			InitialCap         int           `conf:"default:5"`
			MaxIdle            int           `conf:"default:20"`
			MaxCap             int           `conf:"default:30"`
			IdleTimeout        time.Duration `conf:"default:15s"`
		}
		Qubic struct {
			NodePort           string        `conf:"default:21841"`
			StorageFolder      string        `conf:"default:store"`
			ProcessTickTimeout time.Duration `conf:"default:5s"`
		}
	}

	if err := conf.Parse(os.Args[1:], prefix, &cfg); err != nil {
		switch err {
		case conf.ErrHelpWanted:
			usage, err := conf.Usage(prefix, &cfg)
			if err != nil {
				return errors.Wrap(err, "generating config usage")
			}
			fmt.Println(usage)
			return nil
		case conf.ErrVersionWanted:
			version, err := conf.VersionString(prefix, &cfg)
			if err != nil {
				return errors.Wrap(err, "generating config version")
			}
			fmt.Println(version)
			return nil
		}
		return errors.Wrap(err, "parsing config")
	}

	out, err := conf.String(&cfg)
	if err != nil {
		return errors.Wrap(err, "generating config for output")
	}
	log.Printf("main: Config :\n%v\n", out)

	db, err := pebble.Open(cfg.Qubic.StorageFolder, &pebble.Options{})
	if err != nil {
		log.Fatalf("err opening pebble: %s", err.Error())
	}
	defer db.Close()

	ps := store.NewPebbleStore(db, nil)

	p, err := qubic.NewPoolConnection(qubic.PoolConfig{
		InitialCap:         cfg.Pool.InitialCap,
		MaxCap:             cfg.Pool.MaxCap,
		MaxIdle:            cfg.Pool.MaxIdle,
		IdleTimeout:        cfg.Pool.IdleTimeout,
		NodeFetcherUrl:     cfg.Pool.NodeFetcherUrl,
		NodeFetcherTimeout: cfg.Pool.NodeFetcherTimeout,
		NodePort:           cfg.Qubic.NodePort,
	})
	if err != nil {
		return errors.Wrap(err, "creating qubic pool")
	}

	rpcServer := rpc.NewServer(cfg.Server.GrpcHost, cfg.Server.HttpHost, cfg.Server.NodeSyncThreshold, cfg.Server.ChainTickFetchUrl, ps, p)
	rpcServer.Start()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	proc := processor.NewProcessor(p, ps, cfg.Qubic.ProcessTickTimeout)
	procErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		procErrors <- proc.Start()
	}()

	for {
		select {
		case <-shutdown:
			return errors.New("shutting down")
		case err := <-procErrors:
			return errors.Wrap(err, "archiver error")
		}
	}
}
