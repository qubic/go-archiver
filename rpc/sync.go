package rpc

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/processor"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-archiver/validator/quorum"
	"github.com/qubic/go-archiver/validator/tick"
	"google.golang.org/grpc/codes"
	_ "google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/emptypb"
	"sync"
)

var _ protobuff.SyncServiceServer = &SyncService{}

type SyncClientService struct {
	protobuff.UnimplementedSyncClientServiceServer
}

func NewSyncClientService() *SyncClientService {
	return &SyncClientService{}
}

type SyncService struct {
	protobuff.UnimplementedSyncServiceServer
	store                  *store.PebbleStore
	bootstrapConfiguration BootstrapConfiguration
	connectionCount        currentConnectionCount
}

func NewSyncService(pebbleStore *store.PebbleStore, bootstrapConfiguration BootstrapConfiguration) *SyncService {
	return &SyncService{
		bootstrapConfiguration: bootstrapConfiguration,
		store:                  pebbleStore,
		connectionCount: currentConnectionCount{
			mutex: sync.RWMutex{},
			value: 0,
		},
	}
}

func (ss *SyncService) SyncGetBootstrapMetadata(ctx context.Context, _ *emptypb.Empty) (*protobuff.SyncMetadataResponse, error) {

	processedIntervals, err := ss.store.GetProcessedTickIntervals(ctx)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "cannot get processed tick intervals: %v", err)
	}

	return &protobuff.SyncMetadataResponse{
		ArchiverVersion:  utils.ArchiverVersion,
		MaxObjectRequest: int32(ss.bootstrapConfiguration.MaximumRequestedItems),
		//SkippedTickIntervals:   skippedIntervals.SkippedTicks,
		ProcessedTickIntervals: processedIntervals,
	}, nil
}

func (ss *SyncService) sendEpochInformationResponse(epochs []*protobuff.SyncEpochData, stream protobuff.SyncService_SyncGetEpochInformationServer) error {
	response := &protobuff.SyncEpochInfoResponse{
		Epochs: epochs,
	}
	if err := stream.Send(response); err != nil {
		return errors.Wrapf(err, "streaming epoch response")
	}
	return nil
}

func (ss *SyncService) SyncGetEpochInformation(req *protobuff.SyncEpochInfoRequest, stream protobuff.SyncService_SyncGetEpochInformationServer) error {

	if len(req.Epochs) > ss.bootstrapConfiguration.MaximumRequestedItems {
		return status.Errorf(codes.OutOfRange, "the number of requested epochs (%d) exceeds the maximum allowed (%d)", len(req.Epochs), ss.bootstrapConfiguration.MaximumRequestedItems)
	}

	var epochs []*protobuff.SyncEpochData

	for _, epoch := range req.Epochs {
		computors, err := ss.store.GetComputors(context.Background(), epoch)
		if err != nil {
			return status.Errorf(codes.Internal, "getting epoch computors: %v", err)
		}

		lastTickQuorumDataPerIntervals, err := ss.store.GetLastTickQuorumDataListPerEpochInterval(epoch)
		if err != nil {
			return status.Errorf(codes.Internal, "getting quorum data for epoch's last tick: %v", err)
		}

		epochData := &protobuff.SyncEpochData{
			ComputorList:                   computors,
			LastTickQuorumDataPerIntervals: lastTickQuorumDataPerIntervals,
		}

		epochs = append(epochs, epochData)

		if len(epochs) >= ss.bootstrapConfiguration.BatchSize {
			err := ss.sendEpochInformationResponse(epochs, stream)
			if err != nil {
				return errors.Wrap(err, "sending epoch information")
			}
			epochs = make([]*protobuff.SyncEpochData, 0)
		}
	}

	err := ss.sendEpochInformationResponse(epochs, stream)
	if err != nil {
		return errors.Wrap(err, "sending epoch information")
	}

	return nil
}

func (ss *SyncService) sendTickInformationResponse(ticks []*protobuff.SyncTickData, stream protobuff.SyncService_SyncGetTickInformationServer) error {
	response := &protobuff.SyncTickInfoResponse{
		Ticks: ticks,
	}
	if err := stream.Send(response); err != nil {
		return errors.Wrapf(err, "streaming tick response")
	}
	return nil
}

func (ss *SyncService) SyncGetTickInformation(req *protobuff.SyncTickInfoRequest, stream protobuff.SyncService_SyncGetTickInformationServer) error {

	available := ss.bootstrapConfiguration.MaxConcurrentConnections - ss.connectionCount.getValue()
	if available < 1 {
		return utils.SyncMaxConnReachedErr
	}
	ss.connectionCount.increment()
	defer ss.connectionCount.decrement()

	tickDifference := int(req.LastTick - req.FirstTick)

	if tickDifference > ss.bootstrapConfiguration.MaximumRequestedItems || tickDifference < 0 {
		return status.Errorf(codes.OutOfRange, "the number of requested ticks (%d) is not within the allowed range (0 - %d)", tickDifference, ss.bootstrapConfiguration.MaximumRequestedItems)
	}

	var ticks []*protobuff.SyncTickData

	fmt.Printf("RANGE: [%d - %d]\n", req.FirstTick, req.LastTick)

	for tickNumber := req.FirstTick; tickNumber <= req.LastTick; tickNumber++ {
		tickData, err := ss.store.GetTickData(context.Background(), tickNumber)
		if err != nil {
			return status.Errorf(codes.Internal, "getting tick data for tick %d: %v", tickNumber, err)
		}

		quorumData, err := quorum.GetQuorumTickData(tickNumber, ss.store)
		if err != nil {
			return status.Errorf(codes.Internal, "getting quorum data for tick %d: %v", tickNumber, err)
		}

		transactions, err := ss.store.GetTickTransactions(context.Background(), tickNumber)
		if err != nil {
			return status.Errorf(codes.Internal, "getting transactions for tick %d: %v", tickNumber, err)
		}

		transactionStatuses, err := ss.store.GetTickTransactionsStatus(context.Background(), uint64(tickNumber))
		if err != nil {
			return status.Errorf(codes.Internal, "getting transaction statuses for tick %d: %v", tickNumber, err)
		}

		if tickNumber != quorumData.QuorumTickStructure.TickNumber || (!tick.CheckIfTickIsEmptyProto(tickData) && tickData.TickNumber != tickNumber) {
			fmt.Printf("Asked: %d, Got Quorum: %d, Got TickNumber: %d\n", tickNumber, quorumData.QuorumTickStructure.TickNumber, tickData.TickNumber)
			return errors.New("read tick from store does not match asked tick")
		}

		syncTickData := &protobuff.SyncTickData{
			TickData:           tickData,
			QuorumData:         quorumData,
			Transactions:       transactions,
			TransactionsStatus: transactionStatuses.Transactions,
		}

		ticks = append(ticks, syncTickData)

		if len(ticks) >= ss.bootstrapConfiguration.BatchSize {
			err := ss.sendTickInformationResponse(ticks, stream)
			if err != nil {
				return errors.Wrap(err, "sending tick information")
			}
			ticks = make([]*protobuff.SyncTickData, 0)
		}

	}
	err := ss.sendTickInformationResponse(ticks, stream)
	if err != nil {
		return errors.Wrap(err, "sending tick information")
	}
	return nil
}

func (scs *SyncClientService) SyncGetStatus(ctx context.Context, _ *emptypb.Empty) (*protobuff.SyncStatus, error) {

	if processor.SynchronizationStatus == nil {
		return nil, status.Errorf(codes.Unavailable, "synchronization status not available yet, please try again in a couple of minutes")
	}

	syncStatus := processor.SynchronizationStatus.Get()

	delta := make([]*protobuff.ProcessedTickIntervalsPerEpoch, 0)

	for _, epochDelta := range syncStatus.Delta {
		delta = append(delta, &protobuff.ProcessedTickIntervalsPerEpoch{
			Epoch:     epochDelta.Epoch,
			Intervals: epochDelta.ProcessedIntervals,
		})
	}

	currentCompactions := store.CompactionCount.Get()

	return &protobuff.SyncStatus{
		NodeVersion:            syncStatus.NodeVersion,
		BootstrapAddresses:     syncStatus.BootstrapAddresses,
		LastSynchronizedTick:   syncStatus.LastSynchronizedTick,
		CurrentEpoch:           syncStatus.CurrentEpoch,
		CurrentTickRange:       syncStatus.CurrentTickRange,
		AverageTicksPerMinute:  int32(syncStatus.AverageTicksPerMinute),
		LastFetchDuration:      syncStatus.LastFetchDuration,
		LastValidationDuration: syncStatus.LastValidationDuration,
		LastStoreDuration:      syncStatus.LastStoreDuration,
		LastTotalDuration:      syncStatus.LastTotalDuration,
		ObjectRequestCount:     int32(syncStatus.ObjectRequestCount),
		FetchRoutineCount:      int32(syncStatus.FetchRoutineCount),
		ValidationRoutineCount: int32(syncStatus.ValidationRoutineCount),
		CurrentCompactionCount: int32(currentCompactions),
		Delta: &protobuff.SyncDelta{
			DeltaPerEpoch: delta,
		},
	}, nil

}

type currentConnectionCount struct {
	mutex sync.RWMutex
	value int
}

func (c *currentConnectionCount) getValue() int {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	return c.value
}

func (c *currentConnectionCount) increment() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.value += 1
}

func (c *currentConnectionCount) decrement() {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	c.value -= 1
}
