package processor

import (
	"cmp"
	"context"
	"fmt"
	"github.com/cockroachdb/pebble"
	"github.com/pkg/errors"
	"github.com/qubic/go-archiver/protobuff"
	"github.com/schollz/progressbar/v3"

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

type EpochDelta struct {
	Epoch              uint32
	ProcessedIntervals []*protobuff.ProcessedTickInterval
}

type SyncDelta []EpochDelta

type SyncProcessor struct {
	syncConfiguration      SyncConfiguration
	syncServiceClients     []protobuff.SyncServiceClient
	PebbleStore            *store.PebbleStore
	syncDelta              SyncDelta
	processTickTimeout     time.Duration
	maxObjectRequest       uint32
	lastSynchronizedTick   *protobuff.SyncLastSynchronizedTick
	validationRoutineCount int
}

func NewSyncProcessor(syncConfiguration SyncConfiguration, pebbleStore *store.PebbleStore) *SyncProcessor {

	return &SyncProcessor{
		syncConfiguration:      syncConfiguration,
		PebbleStore:            pebbleStore,
		syncServiceClients:     make([]protobuff.SyncServiceClient, 0),
		validationRoutineCount: runtime.NumCPU(),
	}
}

func (sp *SyncProcessor) getRandomBootstrap() (protobuff.SyncServiceClient, error) {

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

	metadataClient, err := sp.getRandomBootstrap()
	if err != nil {
		return errors.Wrap(err, "getting random sync client")
	}

	lastSynchronizedTick, err := sp.PebbleStore.GetSyncLastSynchronizedTick()
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
			FetchRoutineCount:      sp.syncConfiguration.FetchRoutineCount,
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

	processedTickIntervals, err := sp.PebbleStore.GetProcessedTickIntervals(nil)
	if err != nil {
		return nil, errors.Wrap(err, "getting processed tick intervals")
	}

	return &protobuff.SyncMetadataResponse{
		ArchiverVersion:        utils.ArchiverVersion,
		ProcessedTickIntervals: processedTickIntervals,
	}, nil
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
		err := sp.PebbleStore.SetComputors(context.Background(), epoch.ComputorList.Epoch, epoch.ComputorList)
		if err != nil {
			return errors.Wrapf(err, "storing computor list for epoch %d", epoch.ComputorList.Epoch)
		}

		err = sp.PebbleStore.SetLastTickQuorumDataPerEpochIntervals(epoch.ComputorList.Epoch, epoch.LastTickQuorumDataPerIntervals)
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

		// Mitigation for epochs that cannot be fully synchronized
		if epochDelta.Epoch == 105 || epochDelta.Epoch == 106 {
			continue
		}

		if sp.lastSynchronizedTick.Epoch > epochDelta.Epoch {
			continue
		}

		SynchronizationStatus.setCurrentEpoch(epochDelta.Epoch)

		log.Printf("Synchronizing ticks for epoch %d...\n", epochDelta.Epoch)

		processedTickIntervalsForEpoch, err := sp.PebbleStore.GetProcessedTickIntervalsPerEpoch(nil, epochDelta.Epoch)
		if err != nil {
			return errors.Wrapf(err, "getting processed tick intervals for epoch %d", epochDelta.Epoch)
		}

		computorList, err := sp.PebbleStore.GetComputors(nil, epochDelta.Epoch)
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

			//Handle skipped intervals
			if initialIntervalTick > sp.lastSynchronizedTick.TickNumber {
				err = sp.PebbleStore.SetSkippedTicksInterval(nil, &protobuff.SkippedTicksInterval{
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
				var secondStart time.Time

				var validatedTicks []*validator.ValidatedTick

				for {
					secondStart = time.Now()

					fetchedTicks := sp.fetchTicks(startTick, endTick)
					SynchronizationStatus.setLastFetchDuration(float32(time.Since(secondStart).Seconds()))
					secondStart = time.Now()

					validatedTicks, err = sp.validateTicks(fetchedTicks, initialIntervalTick, qubicComputors)
					if err != nil {
						log.Printf("Failed to validate tick range [%d - %d]: %v\n", startTick, endTick, err)
						continue
					}
					SynchronizationStatus.setLastValidationDuration(float32(time.Since(secondStart).Seconds()))
					secondStart = time.Now()
					break
				}

				lastSynchronizedTick, err := sp.storeTicks(validatedTicks, epochDelta.Epoch, processedTickIntervalsForEpoch, initialIntervalTick)
				if err != nil {
					return errors.Wrapf(err, "storing processed tick range %d - %d", startTick, endTick)
				}
				SynchronizationStatus.setLastStoreDuration(float32(time.Since(secondStart).Seconds()))
				secondStart = time.Now()

				sp.lastSynchronizedTick = lastSynchronizedTick
				SynchronizationStatus.setLastSynchronizedTick(lastSynchronizedTick)

				elapsed := time.Since(start)
				SynchronizationStatus.setLastTotalDuration(float32(elapsed.Seconds()))

				ticksPerMinute := int(float64(len(validatedTicks)) / elapsed.Seconds() * 60)
				SynchronizationStatus.setAverageTicksPerMinute(ticksPerMinute)

				log.Printf("Done processing %d ticks. Took: %v | Average time / tick: %v\n", len(validatedTicks), elapsed, elapsed.Seconds()/float64(len(validatedTicks)))
			}
		}

	}

	err := tick.CalculateEmptyTicksForAllEpochs(sp.PebbleStore, true)
	if err != nil {
		return errors.Wrap(err, "calculating empty ticks after synchronization")
	}

	log.Println("Finished synchronizing ticks.")
	err = sp.PebbleStore.DeleteSyncLastSynchronizedTick()
	if err != nil {
		return errors.Wrap(err, "resetting synchronization index")
	}
	return nil
}

func (sp *SyncProcessor) performTickInfoRequest(ctx context.Context, randomClient protobuff.SyncServiceClient, compression grpc.CallOption, start, end uint32) (protobuff.SyncService_SyncGetTickInformationClient, error) {

	stream, err := randomClient.SyncGetTickInformation(ctx, &protobuff.SyncTickInfoRequest{
		FirstTick: start,
		LastTick:  end,
	}, compression, grpc.MaxCallRecvMsgSize(600*1024*1024)) // 629145600 bytes - ~ 629.1456 MB
	if err != nil {
		return nil, errors.Wrap(err, "fetching tick information")
	}
	return stream, nil
}

func (sp *SyncProcessor) readTicksFromStream(stream protobuff.SyncService_SyncGetTickInformationClient, bar *progressbar.ProgressBar) ([]*protobuff.SyncTickData, error) {

	var ticks []*protobuff.SyncTickData
	for {
		data, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "reading ticks from stream")
		}

		ticks = append(ticks, data.Ticks...)
		_ = bar.Add(len(data.Ticks))
	}
	return ticks, nil
}

func (sp *SyncProcessor) fetchTicks(startTick, endTick uint32) []*protobuff.SyncTickData {

	ctx, cancel := context.WithTimeout(context.Background(), sp.syncConfiguration.ResponseTimeout)
	defer cancel()

	var compression grpc.CallOption = grpc.EmptyCallOption{}
	if sp.syncConfiguration.EnableCompression {
		compression = grpc.UseCompressor(gzip.Name)
	}

	routineCount := sp.syncConfiguration.FetchRoutineCount
	var waitGroup sync.WaitGroup
	receivedTicksChannel := make(chan []*protobuff.SyncTickData, routineCount)

	tickDifference := endTick - startTick
	batchSize := tickDifference / uint32(routineCount)

	//startTime := time.Now()

	log.Printf("Fetching tick range [%d - %d] on %d routines\n", startTick, endTick, routineCount)

	// TODO: Remove or make toggle-able
	bar := progressbar.Default(int64(tickDifference))
	bar.Describe("Fetching ticks")

	for index := range routineCount {
		waitGroup.Add(1)

		// Split tick range across goroutines
		// EX: startTick = 0, endTick = 1000, routineCount = 2 => routine 0 fetches 0 - 500, routine 1 fetches 501 - 1000
		start := startTick + (batchSize * uint32(index))
		end := start + batchSize - 1

		if end > endTick || index == (int(routineCount)-1) {
			end = endTick
		}

		if end == 15959704 { // Mitigation for bad last tick on epoch 126
			end = 15959703
		}

		go func(receivedTicksChannel chan<- []*protobuff.SyncTickData) {
			defer waitGroup.Done()

			log.Printf("[Routine %d] Fetching tick range %d - %d", index, start, end)

			for {
				randomClient, err := sp.getRandomBootstrap()
				if err != nil {
					log.Printf("[Routine %d] Failed to connect to random bootstrap: %v\n", index, err)
					time.Sleep(sp.syncConfiguration.RetryTimeout)
					continue
				}

				//t := time.Now()

				stream, err := sp.performTickInfoRequest(ctx, randomClient, compression, start, end)
				if err != nil {

					log.Printf("[Routine %d] Failed to request tick range [%d - %d]: %v\n", index, start, end, err)
					time.Sleep(sp.syncConfiguration.RetryTimeout)
					continue
				}

				receivedTicks, err := sp.readTicksFromStream(stream, bar)
				if err != nil {
					log.Printf("[Routine %d] Failed to read tick range [%d - %d]: %v\n", index, start, end, err)
					time.Sleep(sp.syncConfiguration.RetryTimeout)
					continue
				}

				//elapsed := time.Since(t)
				//log.Printf("[Routine %d] Done fetching tick range [%d - %d]. Took %fs\n", index, start, end, elapsed.Seconds())

				receivedTicksChannel <- receivedTicks

				break
			}
		}(receivedTicksChannel)
	}

	waitGroup.Wait()

	//fmt.Printf("Done fetching tick range [%d - %d]. Took: %f\n", startTick, endTick, time.Since(startTime).Seconds())

	var totalReceivedTicks []*protobuff.SyncTickData

	for _ = range routineCount {
		receivedTicks := <-receivedTicksChannel
		totalReceivedTicks = append(totalReceivedTicks, receivedTicks...)
	}

	_ = bar.Close()

	slices.SortFunc(totalReceivedTicks, func(a, b *protobuff.SyncTickData) int {
		// Compare against tick number from quorum data, as tick data may be nil
		return cmp.Compare(a.QuorumData.QuorumTickStructure.TickNumber, b.QuorumData.QuorumTickStructure.TickNumber)
	})

	return totalReceivedTicks
}

func (sp *SyncProcessor) validateTicks(tickInfoResponses []*protobuff.SyncTickData, initialIntervalTick uint32, computors types.Computors) ([]*validator.ValidatedTick, error) {

	syncValidator := validator.NewSyncValidator(initialIntervalTick, computors, tickInfoResponses, sp.PebbleStore, sp.lastSynchronizedTick)
	validatedTicks, err := syncValidator.Validate(sp.validationRoutineCount)
	if err != nil {
		return nil, errors.Wrap(err, "validating ticks")
	}

	return validatedTicks, nil
}

func (sp *SyncProcessor) storeTicks(validatedTicks []*validator.ValidatedTick, epoch uint32, processedTickIntervalsPerEpoch *protobuff.ProcessedTickIntervalsPerEpoch, initialIntervalTick uint32) (*protobuff.SyncLastSynchronizedTick, error) {

	if epoch == 0 {
		return nil, errors.Errorf("epoch is 0")
	}

	db := sp.PebbleStore.GetDB()

	batch := db.NewBatch()
	defer batch.Close()

	log.Println("Storing validated ticks...")

	var lastSynchronizedTick protobuff.SyncLastSynchronizedTick
	lastSynchronizedTick.Epoch = epoch

	for _, validatedTick := range validatedTicks {

		tickNumber := validatedTick.AlignedVotes.QuorumTickStructure.TickNumber

		err := store.AddQuorumTickDataStoredToBatch(batch, validatedTick.AlignedVotes, tickNumber)
		if err != nil {
			return nil, errors.Wrap(err, "storing quorum data")
		}

		err = store.AddTickDataToBatch(batch, validatedTick.TickData, tickNumber)
		if err != nil {
			return nil, errors.Wrap(err, "storing tick data")
		}

		err = store.AddTransactionsToBatch(batch, validatedTick.ValidTransactions)
		if err != nil {
			return nil, errors.Wrap(err, "storing valid transactions")
		}

		err = store.AddTransactionsPerIdentityToBatch(batch, validatedTick.ValidTransactions, tickNumber)
		if err != nil {
			return nil, errors.Wrap(err, "storing transactions per identity")
		}

		err = store.AddApprovedTransactionsToBatch(batch, validatedTick.ApprovedTransactions, tickNumber)
		if err != nil {
			return nil, errors.Wrap(err, "storing approved transactions")
		}

		err = store.AddTransactionsStatusToBatch(batch, validatedTick.ApprovedTransactions)
		if err != nil {
			return nil, errors.Wrap(err, "storing transactions status")
		}

		err = store.AddChainDigestToBatch(batch, validatedTick.ChainHash, tickNumber)
		if err != nil {
			return nil, errors.Wrap(err, "storing chain hash")
		}

		err = store.AddStoreDigestToBatch(batch, validatedTick.StoreHash, tickNumber)
		if err != nil {
			return nil, errors.Wrap(err, "storing store hash")
		}

		lastSynchronizedTick.TickNumber = tickNumber
		lastSynchronizedTick.ChainHash = validatedTick.ChainHash[:]
		lastSynchronizedTick.StoreHash = validatedTick.StoreHash[:]
	}

	err := store.AddLastProcessedTickPerEpochToBatch(batch, epoch, lastSynchronizedTick.TickNumber)
	if err != nil {
		return nil, errors.Wrap(err, "storing last processed tick per epoch")
	}

	err = store.AddLastProcessedTickToBatch(batch, epoch, lastSynchronizedTick.TickNumber)
	if err != nil {
		return nil, errors.Wrap(err, "storing last processed tick")
	}

	// Handle beginning of epoch
	if len(processedTickIntervalsPerEpoch.Intervals) == 0 {
		processedTickIntervalsPerEpoch.Intervals = []*protobuff.ProcessedTickInterval{
			{
				InitialProcessedTick: initialIntervalTick,
				LastProcessedTick:    lastSynchronizedTick.TickNumber,
			},
		}
	}

	// Handle new tick interval within epoch
	if processedTickIntervalsPerEpoch.Intervals[len(processedTickIntervalsPerEpoch.Intervals)-1].InitialProcessedTick != initialIntervalTick {
		processedTickIntervalsPerEpoch.Intervals = append(processedTickIntervalsPerEpoch.Intervals, &protobuff.ProcessedTickInterval{
			InitialProcessedTick: initialIntervalTick,
			LastProcessedTick:    lastSynchronizedTick.TickNumber,
		})
	}

	processedTickIntervalsPerEpoch.Intervals[len(processedTickIntervalsPerEpoch.Intervals)-1].LastProcessedTick = lastSynchronizedTick.TickNumber

	err = store.AddProcessedTickIntervalsPerEpochToBatch(batch, processedTickIntervalsPerEpoch, epoch)
	if err != nil {
		return nil, errors.Wrap(err, "storing processed tick intervals per epoch")
	}

	err = store.AddLastSynchronizedTickToBatch(batch, &lastSynchronizedTick)
	if err != nil {
		return nil, errors.Wrap(err, "storing last synchronized tick")
	}

	err = batch.Commit(pebble.Sync)
	if err != nil {
		return nil, errors.Wrap(err, "commiting batch")
	}

	return &lastSynchronizedTick, nil
}

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
