package main

import (
	"context"
	"fmt"
	"github.com/ardanlabs/conf"
	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/rpc"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/validator"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	qubic "github.com/qubic/go-node-connector"
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
		}
		Qubic struct {
			NodeIp       string `conf:"default:212.51.150.253"`
			NodePort     string `conf:"default:21841"`
			FallbackTick uint64 `conf:"default:12543674"`
			BatchSize    uint64 `conf:"default:500"`
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

	db, err := pebble.Open("store", &pebble.Options{})
	if err != nil {
		log.Fatalf("err opening pebble: %s", err.Error())
	}
	defer db.Close()

	ps := store.NewPebbleStore(db, nil)

	rpcServer := rpc.NewServer("0.0.0.0:8001", "0.0.0.0:8000", ps, nil)
	rpcServer.Start()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(1 * time.Second)

	for {
		select {
		case <-shutdown:
			log.Fatalf("Shutting down")
		case <-ticker.C:
			err = validateMultiple(cfg.Qubic.NodeIp, cfg.Qubic.NodePort, cfg.Qubic.FallbackTick, cfg.Qubic.BatchSize, ps)
			if err != nil {
				log.Printf("Error running batch. Retrying...: %s", err.Error())
			}
			log.Printf("Batch completed, continuing to next one")
		}

	}
}

func validateMultiple(nodeIP string, nodePort string, fallbackStartTick uint64, batchSize uint64, ps *store.PebbleStore) error {
	client, err := qubic.NewConnection(context.Background(), nodeIP, nodePort)
	if err != nil {
		return errors.Wrap(err, "creating qubic client")
	}
	defer client.Close()

	val := validator.NewValidator(client, ps)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	tickInfo, err := client.GetTickInfo(ctx)
	if err != nil {
		return errors.Wrap(err, "getting tick info")
	}
	targetTick := uint64(tickInfo.Tick)
	startTick, err := getNextProcessingTick(ctx, fallbackStartTick, ps)
	if err != nil {
		return errors.Wrap(err, "getting next processing tick")
	}

	if targetTick <= startTick {
		return errors.Errorf("Starting tick %d is in the future. Latest tick is: %d", startTick, tickInfo.Tick)
	}

	if targetTick-startTick > batchSize {
		targetTick = startTick + batchSize
	}

	log.Printf("Starting from tick: %d, validating until tick: %d", startTick, targetTick)
	start := time.Now().Unix()
	for t := startTick; t < targetTick; t++ {
		stepIn := time.Now().Unix()
		if err := val.ValidateTick(context.Background(), t); err != nil {
			return errors.Wrapf(err, "validating tick %d", t)
		}
		err := ps.SetLastProcessedTick(ctx, t)
		if err != nil {
			return errors.Wrapf(err, "setting last processed tick %d", t)
		}
		stepOut := time.Now().Unix()
		log.Printf("Tick %d validated in %d seconds", t, stepOut-stepIn)
	}
	end := time.Now().Unix()
	log.Printf("%d ticks validated in %d seconds", targetTick-startTick, end-start)
	return nil
}

func getNextProcessingTick(ctx context.Context, fallBackTick uint64, ps *store.PebbleStore) (uint64, error) {
	lastTick, err := ps.GetLastProcessedTick(ctx)
	if err != nil {
		if errors.Cause(err) == store.ErrNotFound {
			return fallBackTick, nil
		}

		return 0, errors.Wrap(err, "getting last processed tick")
	}

	return lastTick + 1, nil
}
