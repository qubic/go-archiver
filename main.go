package main

import (
	"fmt"
	"github.com/ardanlabs/conf"
	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/factory"
	"github.com/qubic/go-archiver/processor"
	"github.com/qubic/go-archiver/rpc"
	"github.com/qubic/go-archiver/store"
	"github.com/silenceper/pool"
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
			ReadTimeout     time.Duration `conf:"default:5s"`
			WriteTimeout    time.Duration `conf:"default:5s"`
			ShutdownTimeout time.Duration `conf:"default:5s"`
			HttpHost        string        `conf:"default:0.0.0.0:8000"`
			GrpcHost        string        `conf:"default:0.0.0.0:8001"`
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
			NodePort             string        `conf:"default:21841"`
			StorageFolder        string        `conf:"default:store"`
			ProcessTickTimeout   time.Duration `conf:"default:5s"`
			NrPeersToBroadcastTx int           `conf:"default:2"`
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

	fact := factory.NewQubicConnection(cfg.Pool.NodeFetcherTimeout, cfg.Pool.NodeFetcherUrl)
	poolConfig := pool.Config{
		InitialCap: cfg.Pool.InitialCap,
		MaxIdle:    cfg.Pool.MaxIdle,
		MaxCap:     cfg.Pool.MaxCap,
		Factory:    fact.Connect,
		Close:      fact.Close,
		//The maximum idle time of the connection, the connection exceeding this time will be closed, which can avoid the problem of automatic failure when connecting to EOF when idle
		IdleTimeout: cfg.Pool.IdleTimeout,
	}
	chPool, err := pool.NewChannelPool(&poolConfig)
	if err != nil {
		return errors.Wrap(err, "creating new connection pool")
	}

	rpcServer := rpc.NewServer(cfg.Server.GrpcHost, cfg.Server.HttpHost, ps, chPool, cfg.Qubic.NrPeersToBroadcastTx)
	rpcServer.Start()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	p := processor.NewProcessor(chPool, ps, cfg.Qubic.ProcessTickTimeout)
	archiveErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		archiveErrors <- p.Start()
	}()

	for {
		select {
		case <-shutdown:
			return errors.New("shutting down")
		case err := <-archiveErrors:
			return errors.Wrap(err, "archiver error")
		}
	}
}
