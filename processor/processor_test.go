package processor

import (
	"context"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	pb "github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-node-connector/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"log"
	"os"
	"path/filepath"
	"testing"
)

func TestProcessor_GetLastProcessedTick(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	require.NoError(t, err)
	defer os.RemoveAll(dbDir)

	testPath := filepath.Join(dbDir, "testdb")

	logger, _ := zap.NewDevelopment()
	s, err := store.NewPebbleStore(testPath, logger, 10)
	require.NoError(t, err)

	err = s.HandleEpochTransition(1)
	require.NoError(t, err)

	p := Processor{ps: s}

	currentTickInfo := types.TickInfo{Tick: 100, Epoch: 1}
	// first run with no last processed tick in storage should return 0 as last processed tick and epoch = currentTickInfo.Epoch
	expected := pb.ProcessedTick{TickNumber: 0, Epoch: 1}

	got, err := p.getLastProcessedTick(ctx, currentTickInfo)
	require.NoError(t, err)
	log.Printf("GOT: %v EXPECTED: %v", got, &expected)
	require.True(t, proto.Equal(got, &expected))
}

func TestProcessor_GetNextProcessingTick(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	require.NoError(t, err)
	defer os.RemoveAll(dbDir)

	testPath := filepath.Join(dbDir, "testdb")

	logger, _ := zap.NewDevelopment()
	s, err := store.NewPebbleStore(testPath, logger, 10)
	require.NoError(t, err)

	p := Processor{ps: s}

	currentTickInfo := types.TickInfo{Tick: 105, Epoch: 1, InitialTick: 100}
	lastTick := pb.ProcessedTick{TickNumber: 0, Epoch: 1}

	expected := pb.ProcessedTick{TickNumber: currentTickInfo.InitialTick, Epoch: 1}

	//first run should set next processing tick to initial tick of the current tick info
	got, err := p.getNextProcessingTick(ctx, &lastTick, currentTickInfo)
	require.NoError(t, err)
	require.True(t, proto.Equal(got, &expected))

	lastTick.TickNumber = currentTickInfo.InitialTick
	expected.TickNumber += 1
	//second run should set next processing tick to last + 1
	got, err = p.getNextProcessingTick(ctx, &lastTick, currentTickInfo)
	require.NoError(t, err)
	require.True(t, proto.Equal(got, &expected))

	currentTickInfo.InitialTick = 200
	currentTickInfo.Tick = 205
	currentTickInfo.Epoch = 2
	expected.TickNumber = 200
	expected.Epoch = 2

	// epoch change should set next tick to current tick info initial tick and epoch to current tick info epoch
	got, err = p.getNextProcessingTick(ctx, &lastTick, currentTickInfo)
	require.NoError(t, err)
	require.True(t, proto.Equal(got, &expected))
}

func TestProcessor_ProcessStatus(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	require.NoError(t, err)
	defer os.RemoveAll(dbDir)

	testPath := filepath.Join(dbDir, "testdb")

	logger, _ := zap.NewDevelopment()
	s, err := store.NewPebbleStore(testPath, logger, 10)
	require.NoError(t, err)

	err = s.HandleEpochTransition(1)
	require.NoError(t, err)

	p := Processor{ps: s}

	// first run of the archiver
	lastTick := pb.ProcessedTick{TickNumber: 99, Epoch: 1}
	nextTick := pb.ProcessedTick{TickNumber: 100, Epoch: 1}

	err = p.processStatus(ctx, &lastTick, &nextTick)
	require.NoError(t, err)

	expected := []*pb.ProcessedTickIntervalsPerEpoch{
		{
			Epoch: nextTick.Epoch,
			Intervals: []*pb.ProcessedTickInterval{
				{
					InitialProcessedTick: nextTick.TickNumber,
					LastProcessedTick:    nextTick.TickNumber,
				},
			},
		},
	}
	got, err := s.GetProcessedTickIntervals()
	require.NoError(t, err)
	diff := cmp.Diff(got, expected, cmpopts.IgnoreUnexported(pb.ProcessedTickInterval{}, pb.ProcessedTickIntervalsPerEpoch{}))
	require.True(t, cmp.Equal(diff, ""))

	lastTick.TickNumber = nextTick.TickNumber
	nextTick.TickNumber += 1

	err = p.processStatus(ctx, &lastTick, &nextTick)
	require.NoError(t, err)

	expected[0].Intervals[0].LastProcessedTick = nextTick.TickNumber
	got, err = s.GetProcessedTickIntervals()
	require.NoError(t, err)

	diff = cmp.Diff(got, expected, cmpopts.IgnoreUnexported(pb.ProcessedTickInterval{}, pb.ProcessedTickIntervalsPerEpoch{}))
	require.True(t, cmp.Equal(diff, ""))

	//skipped ticks in the same epoch
	lastTick.TickNumber = nextTick.TickNumber
	nextTick = pb.ProcessedTick{TickNumber: 150, Epoch: 1}
	err = p.processStatus(ctx, &lastTick, &nextTick)
	require.NoError(t, err)
	expected[0].Intervals = append(expected[0].Intervals, &pb.ProcessedTickInterval{
		InitialProcessedTick: nextTick.TickNumber,
		LastProcessedTick:    nextTick.TickNumber,
	})

	got, err = s.GetProcessedTickIntervals()
	require.NoError(t, err)

	diff = cmp.Diff(got, expected, cmpopts.IgnoreUnexported(pb.ProcessedTickInterval{}, pb.ProcessedTickIntervalsPerEpoch{}))
	require.True(t, cmp.Equal(diff, ""))

	lastTick.TickNumber = nextTick.TickNumber
	nextTick.TickNumber += 1
	err = p.processStatus(ctx, &lastTick, &nextTick)
	require.NoError(t, err)

	expected[0].Intervals[1].LastProcessedTick = nextTick.TickNumber
	got, err = s.GetProcessedTickIntervals()
	require.NoError(t, err)

	diff = cmp.Diff(got, expected, cmpopts.IgnoreUnexported(pb.ProcessedTickInterval{}, pb.ProcessedTickIntervalsPerEpoch{}))
	require.True(t, cmp.Equal(diff, ""))

	// new epoch
	err = s.HandleEpochTransition(2)
	require.NoError(t, err)
	lastTick.TickNumber = nextTick.TickNumber
	nextTick = pb.ProcessedTick{TickNumber: 200, Epoch: 2}
	err = p.processStatus(ctx, &lastTick, &nextTick)
	require.NoError(t, err)
	expected = append(expected, &pb.ProcessedTickIntervalsPerEpoch{
		Epoch: nextTick.Epoch,
		Intervals: []*pb.ProcessedTickInterval{
			{
				InitialProcessedTick: nextTick.TickNumber,
				LastProcessedTick:    nextTick.TickNumber,
			},
		},
	})

	got, err = s.GetProcessedTickIntervals()
	require.NoError(t, err)

	diff = cmp.Diff(got, expected, cmpopts.IgnoreUnexported(pb.ProcessedTickInterval{}, pb.ProcessedTickIntervalsPerEpoch{}))
	log.Printf("DIFF: %s\n", diff)
	require.True(t, cmp.Equal(diff, ""))

	lastTick.TickNumber = nextTick.TickNumber
	nextTick.TickNumber += 1

	err = p.processStatus(ctx, &lastTick, &nextTick)
	require.NoError(t, err)

	expected[1].Intervals[0].LastProcessedTick = nextTick.TickNumber
	got, err = s.GetProcessedTickIntervals()
	require.NoError(t, err)

	diff = cmp.Diff(got, expected, cmpopts.IgnoreUnexported(pb.ProcessedTickInterval{}, pb.ProcessedTickIntervalsPerEpoch{}))
	require.True(t, cmp.Equal(diff, ""))
}

func TestProcessor_ProcessStatusOnthefly(t *testing.T) {
	ctx := context.Background()

	// Setup test environment
	dbDir, err := os.MkdirTemp("", "pebble_test")
	require.NoError(t, err)
	defer os.RemoveAll(dbDir)

	testPath := filepath.Join(dbDir, "testdb")

	logger, _ := zap.NewDevelopment()
	s, err := store.NewPebbleStore(testPath, logger, 10)
	require.NoError(t, err)

	err = s.HandleEpochTransition(1)
	require.NoError(t, err)

	p := Processor{ps: s}

	// first run of the archiver
	lastTick := pb.ProcessedTick{TickNumber: 99, Epoch: 1}
	nextTick := pb.ProcessedTick{TickNumber: 100, Epoch: 1}

	err = p.processStatus(ctx, &lastTick, &nextTick)
	require.NoError(t, err)

	expected := []*pb.ProcessedTickIntervalsPerEpoch{
		{
			Epoch: nextTick.Epoch,
			Intervals: []*pb.ProcessedTickInterval{
				{
					InitialProcessedTick: nextTick.TickNumber,
					LastProcessedTick:    nextTick.TickNumber,
				},
			},
		},
	}
	got, err := s.GetProcessedTickIntervals()
	require.NoError(t, err)
	diff := cmp.Diff(got, expected, cmpopts.IgnoreUnexported(pb.ProcessedTickInterval{}, pb.ProcessedTickIntervalsPerEpoch{}))
	require.True(t, cmp.Equal(diff, ""))
}
