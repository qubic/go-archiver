package processor

import (
	"cmp"
	"context"
	"encoding/binary"
	"fmt"
	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"

	"github.com/qubic/go-archiver/store"
	"github.com/qubic/go-archiver/utils"
	"github.com/qubic/go-archiver/validator"
	"github.com/qubic/go-archiver/validator/computors"
	"github.com/qubic/go-archiver/validator/tick"
	"github.com/qubic/go-node-connector/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"math/rand"
	"runtime"
	"slices"
	"strings"
	"sync"
	"time"
)

type SyncStatus struct {
	NodeVersion            string
	BootstrapAddresses     []string
	Delta                  SyncDelta
	LastSynchronizedTick   *protobuff.SyncLastSynchronizedTick
	CurrentEpoch           uint32
	CurrentTickRange       *protobuff.ProcessedTickInterval
	AverageTicksPerMinute  int
	LastFetchDuration      float32
	LastValidationDuration float32
	LastStoreDuration      float32
	LastTotalDuration      float32
	ObjectRequestCount     uint32
	FetchRoutineCount      int
	ValidationRoutineCount int
}

type SyncStatusMutex struct {
	mutex  sync.RWMutex
	Status *SyncStatus
}

func (ssm *SyncStatusMutex) setLastSynchronizedTick(lastSynchronizedTick *protobuff.SyncLastSynchronizedTick) {
	ssm.mutex.Lock()
	defer ssm.mutex.Unlock()
	ssm.Status.LastSynchronizedTick = lastSynchronizedTick
}

func (ssm *SyncStatusMutex) setCurrentEpoch(epoch uint32) {
	ssm.mutex.Lock()
	defer ssm.mutex.Unlock()
	ssm.Status.CurrentEpoch = epoch
}

func (ssm *SyncStatusMutex) setCurrentTickRange(currentTickRange *protobuff.ProcessedTickInterval) {
	ssm.mutex.Lock()
	defer ssm.mutex.Unlock()
	ssm.Status.CurrentTickRange = currentTickRange
}

func (ssm *SyncStatusMutex) setAverageTicksPerMinute(tickCount int) {
	ssm.mutex.Lock()
	defer ssm.mutex.Unlock()
	ssm.Status.AverageTicksPerMinute = tickCount
}

func (ssm *SyncStatusMutex) setLastFetchDuration(seconds float32) {
	ssm.mutex.Lock()
	defer ssm.mutex.Unlock()
	ssm.Status.LastFetchDuration = seconds
}

func (ssm *SyncStatusMutex) setLastValidationDuration(seconds float32) {
	ssm.mutex.Lock()
	defer ssm.mutex.Unlock()
	ssm.Status.LastValidationDuration = seconds
}

func (ssm *SyncStatusMutex) setLastStoreDuration(seconds float32) {
	ssm.mutex.Lock()
	defer ssm.mutex.Unlock()
	ssm.Status.LastStoreDuration = seconds
}

func (ssm *SyncStatusMutex) setLastTotalDuration(seconds float32) {
	ssm.mutex.Lock()
	defer ssm.mutex.Unlock()
	ssm.Status.LastTotalDuration = seconds
}

func (ssm *SyncStatusMutex) Get() SyncStatus {
	ssm.mutex.RLock()
	defer ssm.mutex.RUnlock()

	return SyncStatus{
		NodeVersion:            ssm.Status.NodeVersion,
		BootstrapAddresses:     ssm.Status.BootstrapAddresses,
		Delta:                  ssm.Status.Delta,
		LastSynchronizedTick:   proto.Clone(ssm.Status.LastSynchronizedTick).(*protobuff.SyncLastSynchronizedTick),
		CurrentEpoch:           ssm.Status.CurrentEpoch,
		CurrentTickRange:       proto.Clone(ssm.Status.CurrentTickRange).(*protobuff.ProcessedTickInterval),
		AverageTicksPerMinute:  ssm.Status.AverageTicksPerMinute,
		LastFetchDuration:      ssm.Status.LastFetchDuration,
		LastValidationDuration: ssm.Status.LastValidationDuration,
		LastStoreDuration:      ssm.Status.LastStoreDuration,
		LastTotalDuration:      ssm.Status.LastTotalDuration,
		ObjectRequestCount:     ssm.Status.ObjectRequestCount,
		FetchRoutineCount:      ssm.Status.FetchRoutineCount,
		ValidationRoutineCount: ssm.Status.ValidationRoutineCount,
	}

}

var SynchronizationStatus *SyncStatusMutex

type SyncProcessor struct {
	syncConfiguration      SyncConfiguration
	syncServiceClients     []protobuff.SyncServiceClient
	pebbleStore            *store.PebbleStore
	syncDelta              SyncDelta
	processTickTimeout     time.Duration
	maxObjectRequest       uint32
	lastSynchronizedTick   *protobuff.SyncLastSynchronizedTick
	fetchRoutineCount      int
	validationRoutineCount int
}

func NewSyncProcessor(syncConfiguration SyncConfiguration, pebbleStore *store.PebbleStore) *SyncProcessor {

	fetchRoutineCount := min(6, runtime.NumCPU())

	return &SyncProcessor{
		syncConfiguration:      syncConfiguration,
		pebbleStore:            pebbleStore,
		syncServiceClients:     make([]protobuff.SyncServiceClient, 0),
		fetchRoutineCount:      fetchRoutineCount,
		validationRoutineCount: runtime.NumCPU(),
	}
}

func (sp *SyncProcessor) getRandomClient() (protobuff.SyncServiceClient, error) {

	if len(sp.syncServiceClients) == 0 {
		return nil, errors.New("no bootstrap connections available")
	}

	r := rand.New(rand.NewSource(time.Now().Unix()))
	index := r.Intn(len(sp.syncServiceClients))

	return sp.syncServiceClients[index], nil
}

func (sp *SyncProcessor) Start() error {

	grpcConnections := make([]*grpc.ClientConn, 0)

	var bootstrapMetadata *protobuff.SyncMetadataResponse

	clientMetadata, err := sp.getClientMetadata()
	if err != nil {
		return errors.Wrap(err, "getting client metadata")
	}

	fmt.Println("Sync sources:")
	for _, source := range sp.syncConfiguration.Sources {
		grpcConnection, err := grpc.NewClient(source, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return errors.Wrap(err, "creating grpc connection to bootstrap")
		}
		fmt.Println(source)

		grpcConnections = append(grpcConnections, grpcConnection)

		syncServiceClient := protobuff.NewSyncServiceClient(grpcConnection)

		metadata, err := sp.getBootstrapMetadata(syncServiceClient)
		if err != nil {
			log.Printf("Unable to get metadata for bootstrap node %s. It will not be used.\n", source)
			continue
		}

		versionCheck, err := sp.checkVersionSupport(clientMetadata.ArchiverVersion, metadata.ArchiverVersion)
		if err != nil {
			log.Printf("Bootstrap node %s does not match client version. It will not be used: %v\n", source, err)
			continue
		}

		if !versionCheck {
			log.Printf("Bootstrap node %s does not match client version. It will not be used. Client:%s, Bootstrap: %v\n", source, clientMetadata.ArchiverVersion, metadata.ArchiverVersion)
		}

		sp.syncServiceClients = append(sp.syncServiceClients, syncServiceClient)
		if bootstrapMetadata == nil {
			bootstrapMetadata = metadata
		}

		if sp.maxObjectRequest == 0 {
			sp.maxObjectRequest = uint32(bootstrapMetadata.MaxObjectRequest)
		}

		sp.maxObjectRequest = min(uint32(bootstrapMetadata.MaxObjectRequest), sp.maxObjectRequest)
	}

	defer func() {
		for _, grpcConnection := range grpcConnections {
			grpcConnection.Close()
		}
	}()

	if len(sp.syncServiceClients) == 0 || bootstrapMetadata == nil {
		log.Println("No suitable sync sources found, resuming to synchronizing current epoch.")
		return nil
	}

	metadataClient, err := sp.getRandomClient()
	if err != nil {
		return errors.Wrap(err, "getting random sync client")
	}

	lastSynchronizedTick, err := sp.pebbleStore.GetSyncLastSynchronizedTick()
	if err != nil {
		log.Printf("Error fetching last synchronized tick from disk: %v\n", err)
	}

	sp.lastSynchronizedTick = lastSynchronizedTick

	log.Println("Calculating synchronization delta...")
	syncDelta, err := sp.CalculateSyncDelta(bootstrapMetadata, clientMetadata, lastSynchronizedTick)
	if err != nil {
		return errors.Wrap(err, "calculating sync delta")
	}

	if len(syncDelta) == 0 {
		log.Println("Nothing to synchronize, resuming to processing network ticks.")
		return nil
	}

	log.Println("Synchronizing missing epoch information...")
	err = sp.syncEpochInfo(syncDelta, metadataClient)
	if err != nil {
		return errors.Wrap(err, "syncing epoch info")
	}

	sp.syncDelta = syncDelta

	log.Println("Starting tick synchronization")

	SynchronizationStatus = &SyncStatusMutex{
		mutex: sync.RWMutex{},
		Status: &SyncStatus{
			NodeVersion:            utils.ArchiverVersion,
			BootstrapAddresses:     sp.syncConfiguration.Sources,
			Delta:                  sp.syncDelta,
			LastSynchronizedTick:   sp.lastSynchronizedTick,
			ObjectRequestCount:     sp.maxObjectRequest,
			FetchRoutineCount:      sp.fetchRoutineCount,
			ValidationRoutineCount: sp.validationRoutineCount,
		},
	}

	err = sp.synchronize()
	if err != nil {
		return errors.Wrap(err, "performing synchronization")
	}

	return nil
}

func (sp *SyncProcessor) getBootstrapMetadata(syncClient protobuff.SyncServiceClient) (*protobuff.SyncMetadataResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), sp.syncConfiguration.ResponseTimeout)
	defer cancel()

	metadata, err := syncClient.SyncGetBootstrapMetadata(ctx, nil)
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
		ArchiverVersion:        utils.ArchiverVersion,
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

		if !proto.Equal(a[index], b[index]) {
			return false
		}
	}
	return true
}

func (sp *SyncProcessor) CalculateSyncDelta(bootstrapMetadata, clientMetadata *protobuff.SyncMetadataResponse, lastSynchronizedTick *protobuff.SyncLastSynchronizedTick) (SyncDelta, error) {

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

		if lastSynchronizedTick != nil && lastSynchronizedTick.Epoch == epoch {

			var intervals []*protobuff.ProcessedTickInterval

			foundIncompleteInterval := false

			for _, interval := range processedIntervals {
				if !foundIncompleteInterval && lastSynchronizedTick.TickNumber >= interval.InitialProcessedTick && lastSynchronizedTick.TickNumber <= interval.LastProcessedTick {
					intervals = append(intervals, &protobuff.ProcessedTickInterval{
						InitialProcessedTick: lastSynchronizedTick.TickNumber,
						LastProcessedTick:    interval.LastProcessedTick,
					})
					foundIncompleteInterval = true
					continue
				}
				intervals = append(intervals, interval)
			}
			syncDelta = append(syncDelta, EpochDelta{
				Epoch:              epoch,
				ProcessedIntervals: intervals,
			})
			continue
		}

		clientProcessedIntervals, exists := clientProcessedTicks[epoch]

		if !exists || !areIntervalsEqual(processedIntervals, clientProcessedIntervals) {
			epochDelta := EpochDelta{
				Epoch:              epoch,
				ProcessedIntervals: processedIntervals,
			}
			syncDelta = append(syncDelta, epochDelta)
		}

	}

	slices.SortFunc(syncDelta, func(a, b EpochDelta) int {
		return cmp.Compare(a.Epoch, b.Epoch)
	})

	return syncDelta, nil
}

func (sp *SyncProcessor) checkVersionSupport(clientVersion, bootstrapVersion string) (bool, error) {

	if clientVersion[0] != 'v' && bootstrapVersion[0] != 'v' {
		return clientVersion == bootstrapVersion, nil
	}

	clientSplit := strings.Split(clientVersion, ".")
	bootstrapSplit := strings.Split(bootstrapVersion, ".")

	if len(clientSplit) != len(bootstrapSplit) {
		return false, errors.Errorf("mismatch between client and bootstrap version format: client: %s, bootstrap: %s", clientVersion, bootstrapVersion)
	}

	for index := 0; index < len(clientSplit)-1; index++ {
		if clientSplit[index] != bootstrapSplit[index] {
			return false, errors.Errorf("mismatch between client and bootstrap versions: client: %s, bootstrap: %s", clientVersion, bootstrapVersion)
		}
	}
	return true, nil
}

func (sp *SyncProcessor) storeEpochInfo(response *protobuff.SyncEpochInfoResponse) error {

	for _, epoch := range response.Epochs {
		err := sp.pebbleStore.SetComputors(context.Background(), epoch.ComputorList.Epoch, epoch.ComputorList)
		if err != nil {
			return errors.Wrapf(err, "storing computor list for epoch %d", epoch.ComputorList.Epoch)
		}

		err = sp.pebbleStore.SetLastTickQuorumDataPerEpochIntervals(epoch.ComputorList.Epoch, epoch.LastTickQuorumDataPerIntervals)
		if err != nil {
			return errors.Wrapf(err, "storing last tick quorum data for epoch %d", epoch.ComputorList.Epoch)
		}
	}

	return nil
}

func (sp *SyncProcessor) syncEpochInfo(delta SyncDelta, syncClient protobuff.SyncServiceClient) error {

	var epochs []uint32

	for _, epochDelta := range delta {
		epochs = append(epochs, epochDelta.Epoch)
	}

	ctx, cancel := context.WithTimeout(context.Background(), sp.syncConfiguration.ResponseTimeout)

	defer cancel()

	stream, err := syncClient.SyncGetEpochInformation(ctx, &protobuff.SyncEpochInfoRequest{Epochs: epochs})
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

func (sp *SyncProcessor) synchronize() error {

	for _, epochDelta := range sp.syncDelta {

		if epochDelta.Epoch == 105 {
			continue
		}

		if sp.lastSynchronizedTick.Epoch > epochDelta.Epoch {
			continue
		}

		SynchronizationStatus.setCurrentEpoch(epochDelta.Epoch)

		log.Printf("Synchronizing ticks for epoch %d...\n", epochDelta.Epoch)

		processedTickIntervalsForEpoch, err := sp.pebbleStore.GetProcessedTickIntervalsPerEpoch(nil, epochDelta.Epoch)
		if err != nil {
			return errors.Wrapf(err, "getting processed tick intervals for epoch %d", epochDelta.Epoch)
		}

		computorList, err := sp.pebbleStore.GetComputors(nil, epochDelta.Epoch)
		if err != nil {
			return errors.Wrapf(err, "reading computor list from disk for epoch %d", epochDelta.Epoch)
		}

		if len(epochDelta.ProcessedIntervals) == 0 {
			return errors.New(fmt.Sprintf("no processed tick intervals in delta for epoch %d", epochDelta.Epoch))
		}

		log.Printf("Validating computor list")
		err = computors.ValidateProto(nil, validator.GoSchnorrqVerify, computorList)
		if err != nil {
			return errors.Wrapf(err, "validating computors for epoch %d", epochDelta.Epoch)
		}

		qubicComputors, err := computors.ProtoToQubic(computorList)
		if err != nil {
			return errors.Wrap(err, "converting computors to qubic format")
		}

		for _, interval := range epochDelta.ProcessedIntervals {

			SynchronizationStatus.setCurrentTickRange(interval)

			fmt.Printf("Processing range [%d - %d]\n", interval.InitialProcessedTick, interval.LastProcessedTick)

			initialIntervalTick := interval.InitialProcessedTick

			if initialIntervalTick > sp.lastSynchronizedTick.TickNumber {
				err = sp.pebbleStore.SetSkippedTicksInterval(nil, &protobuff.SkippedTicksInterval{
					StartTick: sp.lastSynchronizedTick.TickNumber + 1,
					EndTick:   initialIntervalTick - 1,
				})
				if err != nil {
					return errors.Wrap(err, "appending skipped tick interval")
				}
			}

			for tickNumber := interval.InitialProcessedTick; tickNumber <= interval.LastProcessedTick; tickNumber += sp.maxObjectRequest {

				startTick := tickNumber
				endTick := startTick + sp.maxObjectRequest - 1
				if endTick > interval.LastProcessedTick {
					endTick = interval.LastProcessedTick
				}

				start := time.Now()

				secondStart := time.Now()

				fetchedTicks, err := sp.fetchTicks(startTick, endTick)
				if err != nil {
					return errors.Wrapf(err, "fetching tick range %d - %d", startTick, endTick)
				}
				SynchronizationStatus.setLastFetchDuration(float32(time.Since(secondStart).Seconds()))
				secondStart = time.Now()

				processedTicks, err := sp.processTicks(fetchedTicks, initialIntervalTick, qubicComputors)
				if err != nil {
					return errors.Wrapf(err, "processing tick range %d - %d", startTick, endTick)
				}
				SynchronizationStatus.setLastValidationDuration(float32(time.Since(secondStart).Seconds()))
				secondStart = time.Now()

				lastSynchronizedTick, err := sp.storeTicks(processedTicks, epochDelta.Epoch, processedTickIntervalsForEpoch, initialIntervalTick)
				if err != nil {
					return errors.Wrapf(err, "storing processed tick range %d - %d", startTick, endTick)
				}
				SynchronizationStatus.setLastStoreDuration(float32(time.Since(secondStart).Seconds()))
				secondStart = time.Now()

				sp.lastSynchronizedTick = lastSynchronizedTick
				SynchronizationStatus.setLastSynchronizedTick(lastSynchronizedTick)

				elapsed := time.Since(start)
				SynchronizationStatus.setLastTotalDuration(float32(elapsed.Seconds()))

				ticksPerMinute := int(float64(len(processedTicks)) / elapsed.Seconds() * 60)
				SynchronizationStatus.setAverageTicksPerMinute(ticksPerMinute)

				log.Printf("Done processing %d ticks. Took: %v | Average time / tick: %v\n", len(processedTicks), elapsed, elapsed.Seconds()/float64(len(processedTicks)))
			}
		}

	}

	err := tick.CalculateEmptyTicksForAllEpochs(sp.pebbleStore, true)
	if err != nil {
		return errors.Wrap(err, "calculating empty ticks after synchronization")
	}

	log.Println("Finished synchronizing ticks.")
	err = sp.pebbleStore.DeleteSyncLastSynchronizedTick()
	if err != nil {
		return errors.Wrap(err, "resetting synchronization index")
	}
	return nil
}

func (sp *SyncProcessor) performTickInfoRequest(ctx context.Context, randomClient protobuff.SyncServiceClient, compression grpc.CallOption, start, end uint32) (protobuff.SyncService_SyncGetTickInformationClient, error) {

	stream, err := randomClient.SyncGetTickInformation(ctx, &protobuff.SyncTickInfoRequest{
		FirstTick: start,
		LastTick:  end,
	}, compression)
	if err != nil {
		return nil, errors.Wrap(err, "fetching tick information")
	}
	return stream, nil
}

func (sp *SyncProcessor) fetchTicks(startTick, endTick uint32) ([]*protobuff.SyncTickData, error) {

	ctx, cancel := context.WithTimeout(context.Background(), sp.syncConfiguration.ResponseTimeout)
	defer cancel()

	var compression grpc.CallOption = grpc.EmptyCallOption{}
	if sp.syncConfiguration.EnableCompression {
		compression = grpc.UseCompressor(gzip.Name)
	}

	var responses []*protobuff.SyncTickData

	mutex := sync.RWMutex{}
	routineCount := sp.fetchRoutineCount
	tickDifference := endTick - startTick
	batchSize := tickDifference / uint32(routineCount)
	errChannel := make(chan error, routineCount)
	var waitGroup sync.WaitGroup
	startTime := time.Now()
	counter := 0

	log.Printf("Fetching tick range [%d - %d] on %d routines\n", startTick, endTick, routineCount)

	for index := range routineCount {
		waitGroup.Add(1)

		start := startTick + (batchSize * uint32(index))
		end := start + batchSize - 1

		if end > endTick || index == (int(routineCount)-1) {
			end = endTick
		}

		if end == 15959704 {
			end = 15959703
		}

		go func(errChannel chan<- error) {
			defer waitGroup.Done()

			log.Printf("[Routine %d] Fetching tick range %d - %d", index, start, end)

			lastTime := time.Now()

			var stream protobuff.SyncService_SyncGetTickInformationClient

			for i := 0; i < sp.syncConfiguration.RetryCount; i++ {

				if i == sp.syncConfiguration.RetryCount-1 {
					errChannel <- errors.Errorf("failed to fetch tick range [%d - %d] after retrying %d times", start, end, sp.syncConfiguration.RetryCount)
					return
				}

				randomClient, err := sp.getRandomClient()
				if err != nil {
					errChannel <- errors.Wrap(err, "getting random sync client")
					return
				}

				s, err := sp.performTickInfoRequest(ctx, randomClient, compression, start, end)
				if err != nil {
					if errors.Is(err, utils.SyncMaxConnReachedErr) {
						log.Printf("Failed to fetch tick range [%d - %d]: %v\n", start, end, err)
						continue
					}
					errChannel <- errors.Wrapf(err, "fetching tick range [%d - %d]", start, end)
					return
				}

				stream = s
				break
			}

			for {
				data, err := stream.Recv()
				if err == io.EOF {
					break
				}
				if err != nil {
					errChannel <- errors.Wrap(err, "reading tick information stream")
					return
				}

				mutex.Lock()
				responses = append(responses, data.Ticks...)
				counter += len(data.Ticks)
				mutex.Unlock()

				elapsed := time.Since(lastTime)
				rate := float64(len(data.Ticks)) / elapsed.Seconds()

				var firstFetchedTick uint32
				var lastFetchedTick uint32

				if len(data.Ticks) > 0 {
					firstFetchedTick = data.Ticks[0].QuorumData.QuorumTickStructure.TickNumber
					lastFetchedTick = data.Ticks[len(data.Ticks)-1].QuorumData.QuorumTickStructure.TickNumber
				}

				fmt.Printf("[Routine %d]: Fetched %d ticks - [%d - %d] Took: %v | Rate: %f t/s - ~ %d t/m | Total: %d\n", index, len(data.Ticks), firstFetchedTick, lastFetchedTick, time.Since(lastTime), rate, int(rate*60), counter)

				lastTime = time.Now()
			}

			fmt.Printf("Routine %d finished\n", index)

			errChannel <- nil

		}(errChannel)
	}

	waitGroup.Wait()

	fmt.Printf("Done fetching %d ticks. Took: %v\n", counter, time.Since(startTime))

	for _ = range routineCount {
		err := <-errChannel
		if err != nil {
			return nil, errors.Wrap(err, "fetching ticks concurrently")
		}
	}

	return responses, nil

}

func (sp *SyncProcessor) processTicks(tickInfoResponses []*protobuff.SyncTickData, initialIntervalTick uint32, computors types.Computors) (validator.ValidatedTicks, error) {

	syncValidator := validator.NewSyncValidator(initialIntervalTick, computors, tickInfoResponses, sp.pebbleStore, sp.lastSynchronizedTick)
	validatedTicks, err := syncValidator.Validate(sp.validationRoutineCount)
	if err != nil {
		return nil, errors.Wrap(err, "validating ticks")
	}

	return validatedTicks, nil
}

func (sp *SyncProcessor) storeTicks(validatedTicks validator.ValidatedTicks, epoch uint32, processedTickIntervalsPerEpoch *protobuff.ProcessedTickIntervalsPerEpoch, initialIntervalTick uint32) (*protobuff.SyncLastSynchronizedTick, error) {

	if epoch == 0 {
		return nil, errors.Errorf("epoch is 0")
	}

	db := sp.pebbleStore.GetDB()

	batch := db.NewBatch()
	defer batch.Close()

	log.Println("Storing validated ticks...")

	var lastSynchronizedTick protobuff.SyncLastSynchronizedTick
	lastSynchronizedTick.Epoch = epoch

	for _, validatedTick := range validatedTicks {

		tickNumber := validatedTick.AlignedVotes.QuorumTickStructure.TickNumber

		fmt.Printf("Storing data for tick %d\n", tickNumber)

		quorumDataKey := store.AssembleKey(store.QuorumData, tickNumber)
		serializedData, err := proto.Marshal(validatedTick.AlignedVotes)
		if err != nil {
			return nil, errors.Wrapf(err, "serializing aligned votes for tick %d", tickNumber)
		}
		err = batch.Set(quorumDataKey, serializedData, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "adding aligned votes to batch for tick %d", tickNumber)
		}

		tickDataKey := store.AssembleKey(store.TickData, tickNumber)
		serializedData, err = proto.Marshal(validatedTick.TickData)
		if err != nil {
			return nil, errors.Wrapf(err, "serializing tick data for tick %d", tickNumber)
		}
		err = batch.Set(tickDataKey, serializedData, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "adding tick data to batch for tick %d", tickNumber)
		}

		for _, transaction := range validatedTick.ValidTransactions {
			transactionKey := store.AssembleKey(store.Transaction, transaction.TxId)
			serializedData, err = proto.Marshal(transaction)
			if err != nil {
				return nil, errors.Wrapf(err, "deserializing transaction %s for tick %d", transaction.TxId, tickNumber)
			}
			err = batch.Set(transactionKey, serializedData, nil)
			if err != nil {
				return nil, errors.Wrapf(err, "addin transaction %s to batch for tick %d", transaction.TxId, tickNumber)
			}
		}

		transactionsPerIdentity := removeNonTransferTransactionsAndSortPerIdentity(validatedTick.ValidTransactions)
		for identity, transactions := range transactionsPerIdentity {
			identityTransfersPerTickKey := store.AssembleKey(store.IdentityTransferTransactions, identity)
			identityTransfersPerTickKey = binary.BigEndian.AppendUint64(identityTransfersPerTickKey, uint64(tickNumber))

			serializedData, err = proto.Marshal(&protobuff.TransferTransactionsPerTick{
				TickNumber:   tickNumber,
				Identity:     identity,
				Transactions: transactions,
			})
			if err != nil {
				return nil, errors.Wrapf(err, "serializing transfer transactions for tickl %d", tickNumber)
			}
			err = batch.Set(identityTransfersPerTickKey, serializedData, nil)
			if err != nil {
				return nil, errors.Wrapf(err, "adding transafer transactions to batch for tick %d", tickNumber)
			}
		}

		tickTxStatusKey := store.AssembleKey(store.TickTransactionsStatus, uint64(tickNumber))
		serializedData, err = proto.Marshal(validatedTick.ApprovedTransactions)
		if err != nil {
			return nil, errors.Wrapf(err, "serializing transaction statuses for tick %d", tickNumber)
		}
		err = batch.Set(tickTxStatusKey, serializedData, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "adding transactions statuses to batch for tick %d", tickNumber)
		}
		for _, transaction := range validatedTick.ApprovedTransactions.Transactions {
			approvedTransactionKey := store.AssembleKey(store.TransactionStatus, transaction.TxId)
			serializedData, err = proto.Marshal(transaction)
			if err != nil {
				return nil, errors.Wrapf(err, "serialzing approved transaction %s for tick %d", transaction.TxId, tickNumber)
			}
			err = batch.Set(approvedTransactionKey, serializedData, nil)
			if err != nil {
				return nil, errors.Wrapf(err, "adding approved transaction %s to batch for tick %d", transaction.TxId, tickNumber)
			}
		}

		chainDigestKey := store.AssembleKey(store.ChainDigest, tickNumber)
		err = batch.Set(chainDigestKey, validatedTick.ChainHash[:], nil)
		if err != nil {
			return nil, errors.Wrapf(err, "adding chain hash to batch for tick %d", tickNumber)
		}

		storeDigestKey := store.AssembleKey(store.StoreDigest, tickNumber)
		err = batch.Set(storeDigestKey, validatedTick.StoreHash[:], nil)
		if err != nil {
			return nil, errors.Wrapf(err, "adding store hash to batch for tick %d", tickNumber)
		}

		lastSynchronizedTick.TickNumber = tickNumber
		lastSynchronizedTick.ChainHash = validatedTick.ChainHash[:]
		lastSynchronizedTick.StoreHash = validatedTick.StoreHash[:]
	}

	lastProcessedTickPerEpochKey := store.AssembleKey(store.LastProcessedTickPerEpoch, epoch)
	value := make([]byte, 4)
	binary.LittleEndian.PutUint32(value, lastSynchronizedTick.TickNumber)
	err := batch.Set(lastProcessedTickPerEpochKey, value, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "adding last processed tick %d for epoch %d to batch", lastSynchronizedTick.TickNumber, epoch)
	}

	lastProcessedTickProto := &protobuff.ProcessedTick{
		TickNumber: lastSynchronizedTick.TickNumber,
		Epoch:      epoch,
	}
	lastProcessedTickKey := []byte{store.LastProcessedTick}
	serializedData, err := proto.Marshal(lastProcessedTickProto)
	if err != nil {
		return nil, errors.Wrapf(err, "serializing last processed tick %d", lastSynchronizedTick.TickNumber)
	}
	err = batch.Set(lastProcessedTickKey, serializedData, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "adding last processed tick %d to batch", lastSynchronizedTick.TickNumber)
	}

	if len(processedTickIntervalsPerEpoch.Intervals) == 0 {
		processedTickIntervalsPerEpoch.Intervals = []*protobuff.ProcessedTickInterval{
			{
				InitialProcessedTick: initialIntervalTick,
				LastProcessedTick:    lastSynchronizedTick.TickNumber,
			},
		}
	}

	if processedTickIntervalsPerEpoch.Intervals[len(processedTickIntervalsPerEpoch.Intervals)-1].InitialProcessedTick != initialIntervalTick {
		processedTickIntervalsPerEpoch.Intervals = append(processedTickIntervalsPerEpoch.Intervals, &protobuff.ProcessedTickInterval{
			InitialProcessedTick: initialIntervalTick,
			LastProcessedTick:    lastSynchronizedTick.TickNumber,
		})
	}

	processedTickIntervalsPerEpoch.Intervals[len(processedTickIntervalsPerEpoch.Intervals)-1].LastProcessedTick = lastSynchronizedTick.TickNumber

	processedTickIntervalsPerEpochKey := store.AssembleKey(store.ProcessedTickIntervals, epoch)
	serializedData, err = proto.Marshal(processedTickIntervalsPerEpoch)
	if err != nil {
		return nil, errors.Wrapf(err, "serializing processed tick intervals for epoch %d", epoch)
	}
	err = batch.Set(processedTickIntervalsPerEpochKey, serializedData, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "adding processed tick intervals for epoch %d to batch", epoch)
	}

	lastSynchronizedTickKey := []byte{store.SyncLastSynchronizedTick}
	serializedData, err = proto.Marshal(&lastSynchronizedTick)
	if err != nil {
		return nil, errors.Wrap(err, "serializing last synchronized tick")
	}
	err = batch.Set(lastSynchronizedTickKey, serializedData, nil)
	if err != nil {
		return nil, errors.Wrap(err, "adding synchronization index to batch")
	}

	err = batch.Commit(pebble.Sync)
	if err != nil {
		return nil, errors.Wrap(err, "commiting batch")
	}

	return &lastSynchronizedTick, nil
}

func removeNonTransferTransactionsAndSortPerIdentity(transactions []*protobuff.Transaction) map[string][]*protobuff.Transaction {

	transferTransactions := make([]*protobuff.Transaction, 0)
	for _, transaction := range transactions {
		if transaction.Amount == 0 {
			continue
		}
		transferTransactions = append(transferTransactions, transaction)
	}
	transactionsPerIdentity := make(map[string][]*protobuff.Transaction)
	for _, transaction := range transferTransactions {
		_, exists := transactionsPerIdentity[transaction.DestId]
		if !exists {
			transactionsPerIdentity[transaction.DestId] = make([]*protobuff.Transaction, 0)
		}
		_, exists = transactionsPerIdentity[transaction.SourceId]
		if !exists {
			transactionsPerIdentity[transaction.SourceId] = make([]*protobuff.Transaction, 0)
		}
	}

	return transactionsPerIdentity

}
