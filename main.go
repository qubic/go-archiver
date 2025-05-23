package main

import (
	"fmt"
	"github.com/ardanlabs/conf"
	"github.com/cockroachdb/pebble"
	grpcProm "github.com/grpc-ecosystem/go-grpc-middleware/providers/prometheus"
	"github.com/pkg/errors"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/qubic/go-archiver/processor"
	"github.com/qubic/go-archiver/rpc"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/validator/tick"
	qubic "github.com/qubic/go-node-connector"
	"github.com/qubic/go-node-connector/types"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"runtime"
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
			ProfilingHost     string        `conf:"default:0.0.0.0:8002"`
			NodeSyncThreshold int           `conf:"default:3"`
			ChainTickFetchUrl string        `conf:"default:http://127.0.0.1:8080/max-tick"`
		}
		Pool struct {
			NodeFetcherUrl     string        `conf:"default:http://127.0.0.1:8080/status"`
			NodeFetcherTimeout time.Duration `conf:"default:2s"`
			InitialCap         int           `conf:"default:5"`
			MaxIdle            int           `conf:"default:20"`
			MaxCap             int           `conf:"default:30"`
			IdleTimeout        time.Duration `conf:"default:15s"`
		}
		Qubic struct {
			NodePort                      string        `conf:"default:21841"`
			StorageFolder                 string        `conf:"default:store"`
			ProcessTickTimeout            time.Duration `conf:"default:5s"`
			DisableTransactionStatusAddon bool          `conf:"default:false"`
			ArbitratorIdentity            string        `conf:"default:AFZPUAIYVPNUYGJRQVLUKOPPVLHAZQTGLYAAUUNBXFTVTAMSBKQBLEIEPCVJ"`
		}
		Store struct {
			ResetEmptyTickKeys bool `conf:"default:false"`
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

	l1Options := pebble.LevelOptions{
		BlockRestartInterval: 16,
		BlockSize:            4096,
		BlockSizeThreshold:   90,
		Compression:          pebble.NoCompression,
		FilterPolicy:         nil,
		FilterType:           pebble.TableFilter,
		IndexBlockSize:       4096,
		TargetFileSize:       268435456, // 256 MB
	}
	l2Options := pebble.LevelOptions{
		BlockRestartInterval: 16,
		BlockSize:            4096,
		BlockSizeThreshold:   90,
		Compression:          pebble.ZstdCompression,
		FilterPolicy:         nil,
		FilterType:           pebble.TableFilter,
		IndexBlockSize:       4096,
		TargetFileSize:       l1Options.TargetFileSize * 10, // 2.5 GB
	}
	l3Options := pebble.LevelOptions{
		BlockRestartInterval: 16,
		BlockSize:            4096,
		BlockSizeThreshold:   90,
		Compression:          pebble.ZstdCompression,
		FilterPolicy:         nil,
		FilterType:           pebble.TableFilter,
		IndexBlockSize:       4096,
		TargetFileSize:       l2Options.TargetFileSize * 10, // 25 GB
	}
	l4Options := pebble.LevelOptions{
		BlockRestartInterval: 16,
		BlockSize:            4096,
		BlockSizeThreshold:   90,
		Compression:          pebble.ZstdCompression,
		FilterPolicy:         nil,
		FilterType:           pebble.TableFilter,
		IndexBlockSize:       4096,
		TargetFileSize:       l3Options.TargetFileSize * 10, // 250 GB
	}

	pebbleOptions := pebble.Options{
		Levels:                   []pebble.LevelOptions{l1Options, l2Options, l3Options, l4Options},
		MaxConcurrentCompactions: func() int { return runtime.NumCPU() },
		MemTableSize:             268435456, // 256 MB
		EventListener:            store.NewPebbleEventListener(),
	}

	db, err := pebble.Open(cfg.Qubic.StorageFolder, &pebbleOptions)
	if err != nil {
		return errors.Wrap(err, "opening db with zstd compression")
	}
	defer db.Close()

	ps := store.NewPebbleStore(db, nil)

	if cfg.Store.ResetEmptyTickKeys {
		fmt.Printf("Resetting empty ticks for all epochs...\n")
		err = tick.ResetEmptyTicksForAllEpochs(ps)
		if err != nil {
			return errors.Wrap(err, "resetting empty ticks keys")
		}
	}

	err = tick.CalculateEmptyTicksForAllEpochs(ps)
	if err != nil {
		return errors.Wrap(err, "calculating empty ticks for all epochs")
	}

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

	srvMetrics := grpcProm.NewServerMetrics(
		grpcProm.WithServerCounterOptions(grpcProm.WithConstLabels(prometheus.Labels{"namespace": "archiver"})),
	)
	reg := prometheus.NewRegistry()
	reg.MustRegister(srvMetrics)
	reg.MustRegister(collectors.NewGoCollector())

	rpcServer := rpc.NewServer(cfg.Server.GrpcHost, cfg.Server.HttpHost, cfg.Server.NodeSyncThreshold, cfg.Server.ChainTickFetchUrl, ps, p)
	err = rpcServer.Start(srvMetrics.UnaryServerInterceptor())
	if err != nil {
		return errors.Wrap(err, "starting rpc server")
	}

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	arbitratorID := types.Identity(cfg.Qubic.ArbitratorIdentity)
	arbitratorPubKey, err := arbitratorID.ToPubKey(false)
	if err != nil {
		return errors.Wrapf(err, "getting arbitrator public key from [%s]", cfg.Qubic.ArbitratorIdentity)
	}

	proc := processor.NewProcessor(p, ps, cfg.Qubic.ProcessTickTimeout, arbitratorPubKey, cfg.Qubic.DisableTransactionStatusAddon)
	procErrors := make(chan error, 1)

	// Start the service listening for requests.
	go func() {
		procErrors <- proc.Start()
	}()

	pprofErrors := make(chan error, 1)

	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{EnableOpenMetrics: true}))
	go func() {
		pprofErrors <- http.ListenAndServe(cfg.Server.ProfilingHost, nil)
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
