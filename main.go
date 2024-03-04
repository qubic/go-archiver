package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/ardanlabs/conf"
	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/rpc"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/validator"
	"github.com/silenceper/pool"
	"io"
	"log"
	"math/rand"
	"net/http"
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
			HttpHost        string        `conf:"0.0.0.0:8000"`
			GrpcHost        string        `conf:"0.0.0.0:8001"`
		}
		Qubic struct {
			NodeIp              string `conf:"default:212.51.150.253"`
			NodePort            string `conf:"default:21841"`
			FallbackTick        uint64 `conf:"default:12710000"`
			BatchSize           uint64 `conf:"default:500"`
			NodeFetcherEndpoint string `conf:"default:http://127.0.0.1:8080/peers"`
			StorageFolder       string `conf:"default:store"`
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

	rpcServer := rpc.NewServer(cfg.Server.GrpcHost, cfg.Server.HttpHost, ps, nil)
	rpcServer.Start()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	ticker := time.NewTicker(5 * time.Second)

	cf := &connectionFactory{nodeFetcherHost: cfg.Qubic.NodeFetcherEndpoint}
	//p, err := connpool.NewChannelPool(5, 30, cf.connect)
	//if err != nil {
	//	return errors.Wrap(err, "creating new connection pool")
	//}

	poolConfig := pool.Config{
		InitialCap: 5,
		MaxIdle:    20,
		MaxCap:     30,
		Factory:    cf.connect,
		Close:      cf.close,
		//Ping:       ping,
		//The maximum idle time of the connection, the connection exceeding this time will be closed, which can avoid the problem of automatic failure when connecting to EOF when idle
		IdleTimeout: 15 * time.Second,
	}
	p, err := pool.NewChannelPool(&poolConfig)
	if err != nil {
		return errors.Wrap(err, "creating new connection pool")
	}

	for {
		select {
		case <-shutdown:
			log.Fatalf("Shutting down")
		case <-ticker.C:
			err := do(p, cfg.Qubic.FallbackTick, cfg.Qubic.BatchSize, ps)
			if err != nil {
				log.Printf("do err: %s. pool len: %d\n", err.Error(), p.Len())
			}
		}
	}
}

func do(pool pool.Pool, fallbackTick, batchSize uint64, ps *store.PebbleStore) error {
	//ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	//defer cancel()

	v, err := pool.Get()
	if err != nil {
		return errors.Wrap(err, "getting initial connection")
	}

	client := v.(*qubic.Client)
	err = validateMultiple(client, fallbackTick, batchSize, ps)
	if err != nil {
		cErr := pool.Close(v)
		if cErr != nil {
			log.Printf("Closing conn failed: %s", cErr.Error())
		}
		return errors.Wrap(err, "validating multiple")
	} else {
		err = pool.Put(client)
		if err != nil {
			log.Printf("Putting conn back to pool failed: %s", err.Error())
		}
	}

	log.Println("Batch completed")

	return nil
}

func validateMultiple(client *qubic.Client, fallbackStartTick uint64, batchSize uint64, ps *store.PebbleStore) error {
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	val := validator.NewValidator(client, ps)
	tickInfo, err := client.GetTickInfo(ctx)
	if err != nil {
		return errors.Wrap(err, "getting tick info")
	}
	targetTick := uint64(tickInfo.Tick)
	startTick, err := getNextProcessingTick(ctx, fallbackStartTick, ps)
	if err != nil {
		return errors.Wrap(err, "getting next processing tick")
	}

	if targetTick-startTick > batchSize {
		targetTick = startTick + batchSize
	}

	if targetTick <= startTick {
		return errors.Errorf("Target processing tick %d is not greater than last processing tick %d", targetTick, startTick)
	}

	log.Printf("Current batch starting from tick: %d until tick: %d", startTick, targetTick)
	start := time.Now().Unix()
	for t := startTick; t <= targetTick; t++ {
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

type connectionFactory struct {
	nodeFetcherHost string
}

func (cf *connectionFactory) connect() (interface{}, error) {
	peer, err := getNewRandomPeer(cf.nodeFetcherHost)
	if err != nil {
		return nil, errors.Wrap(err, "getting new random peer")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := qubic.NewClient(ctx, peer, "21841")
	if err != nil {
		return nil, errors.Wrap(err, "creating qubic client")
	}

	fmt.Printf("connected to: %s\n", peer)
	return client, nil
}

func (cf *connectionFactory) close(v interface{}) error { return v.(*qubic.Client).Close() }

type response struct {
	Peers       []string `json:"peers"`
	Length      int      `json:"length"`
	LastUpdated int64    `json:"last_updated"`
}

func getNewRandomPeer(host string) (string, error) {
	res, err := http.Get(host)
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
