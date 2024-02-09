package main

import (
	"context"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/validator"
	"go.uber.org/zap"
	"log"
	"os"
	"strconv"
	"time"

	qubic "github.com/qubic/go-node-connector"
)

var nodePort = "21841"

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("Please provide tick number and nodeIP")
	}

	ip := os.Args[1]

	tickNumber, err := strconv.ParseUint(os.Args[2], 10, 64)
	if err != nil {
		log.Fatalf("Parsing tick number. err: %s", err.Error())
	}

	err = run(ip, tickNumber)
	if err != nil {
		log.Fatalf(err.Error())
	}
}

func run(nodeIP string, tickNumber uint64) error {
	client, err := qubic.NewClient(context.Background(), nodeIP, nodePort)
	if err != nil {
		log.Fatalf("creating qubic sdk: err: %s", err.Error())
	}
	// releasing tcp connection related resources
	defer client.Close()

	logger, _ := zap.NewDevelopment()
	val := validator.NewValidator(client, store.NewPebbleStore(nil, logger))

	failedTicks := 0
	start := time.Now().Unix()
	for t := tickNumber; t < tickNumber+30; t++ {
		stepIn := time.Now().Unix()
		if err := val.ValidateTick(context.Background(), t); err != nil {
			failedTicks += 1
			log.Printf("Failed to validate tick %d: %s", t, err.Error())
			continue
			//return errors.Wrapf(err, "validating tick: %d", t)
		}
		stepOut := time.Now().Unix()
		log.Printf("Tick %d validated in %d seconds", t, stepOut-stepIn)
	}
	end := time.Now().Unix()
	log.Printf("30 ticks validated in %d seconds", end-start)

	return nil
}
