package processor

import (
	"context"
	"fmt"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/sync"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"io"
	"log"
	"time"
)

type SyncProcessor struct {
	syncConfiguration  SyncConfiguration
	syncServiceClient  protobuff.SyncServiceClient
	pebbleStore        *store.PebbleStore
	syncDelta          SyncDelta
	processTickTimeout time.Duration
	maxObjectRequest   uint32
}

func NewSyncProcessor(syncConfiguration SyncConfiguration, pebbleStore *store.PebbleStore, processTickTimeout time.Duration) *SyncProcessor {
	return &SyncProcessor{
		syncConfiguration:  syncConfiguration,
		pebbleStore:        pebbleStore,
		processTickTimeout: processTickTimeout,
	}
}

func (sp *SyncProcessor) Start() error {

	log.Printf("Connecting to bootstrap node %s...", sp.syncConfiguration.Source)

	grpcConnection, err := grpc.NewClient(sp.syncConfiguration.Source, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return errors.Wrap(err, "creating grpc connection to bootstrap")
	}
	defer grpcConnection.Close()

	syncServiceClient := protobuff.NewSyncServiceClient(grpcConnection)
	sp.syncServiceClient = syncServiceClient

	log.Println("Fetching bootstrap metadata...")
	bootstrapMetadata, err := sp.getBootstrapMetadata()
	if err != nil {
		return err
	}

	sp.maxObjectRequest = uint32(bootstrapMetadata.MaxObjectRequest)

	clientMetadata, err := sp.getClientMetadata()
	if err != nil {
		return errors.Wrap(err, "getting client metadata")
	}

	log.Println("Calculating synchronization delta...")
	syncDelta, err := sp.calculateSyncDelta(bootstrapMetadata, clientMetadata)
	if err != nil {
		return errors.Wrap(err, "calculating sync delta")
	}

	if len(syncDelta) == 0 {
		log.Println("Nothing to synchronize, resuming to processing network ticks.")
		return nil
	}

	log.Println("Synchronizing missing epoch information...")
	err = sp.syncEpochInfo(syncDelta)
	if err != nil {
		return errors.Wrap(err, "syncing epoch info")
	}

	sp.syncDelta = syncDelta

	log.Println("Starting tick synchronization")
	sp.sync()

	return nil
}

func (sp *SyncProcessor) getBootstrapMetadata() (*protobuff.SyncMetadataResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), sp.syncConfiguration.ResponseTimeout)
	defer cancel()

	metadata, err := sp.syncServiceClient.SyncGetBootstrapMetadata(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "getting bootstrap metadata")
	}

	return metadata, nil
}

func (sp *SyncProcessor) getClientMetadata() (*protobuff.SyncMetadataResponse, error) {

	processedTickIntervals, err := sp.pebbleStore.GetProcessedTickIntervals(nil)
	if err != nil {
		return nil, errors.Wrap(err, "getting processed tick intervals")
	}

	return &protobuff.SyncMetadataResponse{
		ArchiverVersion:        sync.ArchiverVersion,
		ProcessedTickIntervals: processedTickIntervals,
	}, nil
}

type EpochDelta struct {
	Epoch              uint32
	ProcessedIntervals []*protobuff.ProcessedTickInterval
}

type SyncDelta []EpochDelta

func areIntervalsEqual(a, b []*protobuff.ProcessedTickInterval) bool {
	if len(a) != len(b) {
		return false
	}

	for index := 0; index < len(a); index++ {
		if a[index] != b[index] {
			return false
		}
	}
	return true
}

func (sp *SyncProcessor) calculateSyncDelta(bootstrapMetadata, clientMetadata *protobuff.SyncMetadataResponse) (SyncDelta, error) {

	if bootstrapMetadata.ArchiverVersion != clientMetadata.ArchiverVersion {
		return nil, errors.New(fmt.Sprintf("client version (%s) does not match bootstrap version (%s)", clientMetadata.ArchiverVersion, bootstrapMetadata.ArchiverVersion))
	}

	bootstrapProcessedTicks := make(map[uint32][]*protobuff.ProcessedTickInterval)
	clientProcessedTicks := make(map[uint32][]*protobuff.ProcessedTickInterval)

	for _, epochIntervals := range bootstrapMetadata.ProcessedTickIntervals {
		bootstrapProcessedTicks[epochIntervals.Epoch] = epochIntervals.Intervals
	}

	for _, epochIntervals := range clientMetadata.ProcessedTickIntervals {
		clientProcessedTicks[epochIntervals.Epoch] = epochIntervals.Intervals
	}

	var syncDelta SyncDelta

	for epoch, processedIntervals := range bootstrapProcessedTicks {

		clientProcessedIntervals, exists := clientProcessedTicks[epoch]
		if !exists || !areIntervalsEqual(processedIntervals, clientProcessedIntervals) {
			epochDelta := EpochDelta{
				Epoch:              epoch,
				ProcessedIntervals: processedIntervals,
			}
			syncDelta = append(syncDelta, epochDelta)
		}
	}

	return syncDelta, nil
}

func (sp *SyncProcessor) storeEpochInfo(response *protobuff.SyncEpochInfoResponse) error {

	for _, epoch := range response.Epochs {
		err := sp.pebbleStore.SetComputors(context.Background(), epoch.ComputorList.Epoch, epoch.ComputorList)
		if err != nil {
			return errors.Wrapf(err, "storing computor list for epoch %d", epoch.ComputorList.Epoch)
		}

		err = sp.pebbleStore.SetLastTickQuorumDataPerEpoch(epoch.LastTickQuorumData, epoch.LastTickQuorumData.QuorumTickStructure.Epoch)
		if err != nil {
			return errors.Wrapf(err, "storing last tick quorum data for epoch %d", epoch.LastTickQuorumData.QuorumTickStructure.Epoch)
		}
	}

	return nil
}

func (sp *SyncProcessor) syncEpochInfo(delta SyncDelta) error {

	var epochs []uint32

	for _, epochDelta := range delta {
		epochs = append(epochs, epochDelta.Epoch)
	}

	ctx, cancel := context.WithTimeout(context.Background(), sp.syncConfiguration.ResponseTimeout)

	defer cancel()

	stream, err := sp.syncServiceClient.SyncGetEpochInformation(ctx, &protobuff.SyncEpochInfoRequest{Epochs: epochs})
	if err != nil {
		return errors.Wrap(err, "fetching epoch info")
	}

	for {
		data, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "reading stream")
		}

		err = sp.storeEpochInfo(data)
		if err != nil {
			return errors.Wrap(err, "storing epoch data")
		}
	}

	return nil
}

func (sp *SyncProcessor) sync() error {
	for _, epochDelta := range sp.syncDelta {

		log.Printf("Synchronizing ticks for epoch %d...\n", epochDelta.Epoch)

		for _, interval := range epochDelta.ProcessedIntervals {

			for tickNumber := interval.InitialProcessedTick; tickNumber < interval.LastProcessedTick; tickNumber += sp.maxObjectRequest {

				startTick := tickNumber
				endTick := startTick + sp.maxObjectRequest

				err := sp.processTick(tickNumber, tickNumber+20)
				if err != nil {
					return errors.Wrapf(err, "synchronizing tick %d", tickNumber)
				}

			}

		}
	}
	return nil
}

func (sp *SyncProcessor) processTicks(startTick, endTick uint32) error {

}
