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
	"github.com/qubic/go-node-connector/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/protobuf/proto"
	"io"
	"log"
	"runtime"
	"slices"
	"sync"
	"time"
)

type SyncProcessor struct {
	syncConfiguration  SyncConfiguration
	syncServiceClient  protobuff.SyncServiceClient
	pebbleStore        *store.PebbleStore
	syncDelta          SyncDelta
	processTickTimeout time.Duration
	maxObjectRequest   uint32
	lastProcessedTick  uint32
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
	err = sp.syncEpochInfo(syncDelta, bootstrapMetadata)
	if err != nil {
		return errors.Wrap(err, "syncing epoch info")
	}

	sp.syncDelta = syncDelta

	log.Println("Starting tick synchronization")
	err = sp.synchronize()
	if err != nil {
		return errors.Wrap(err, "performing synchronization")
	}

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

	slices.SortFunc(syncDelta, func(a, b EpochDelta) int {
		return cmp.Compare(a.Epoch, b.Epoch)
	})

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

func (sp *SyncProcessor) syncEpochInfo(delta SyncDelta, metadata *protobuff.SyncMetadataResponse) error {

	// TODO: remove skipped tick intervals from proto file
	/*err := sp.pebbleStore.SetSkippedTickIntervalList(&protobuff.SkippedTicksIntervalList{
		SkippedTicks: metadata.SkippedTickIntervals,
	})
	if err != nil {
		return errors.Wrap(err, "saving skipped tick intervals from bootstrap")
	}*/

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

func (sp *SyncProcessor) synchronize() error {

	for _, epochDelta := range sp.syncDelta {

		switch epochDelta.Epoch {
		case 123:
			continue
		case 124:
		default:
		}

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

			initialIntervalTick := interval.InitialProcessedTick

			if initialIntervalTick > sp.lastProcessedTick {
				err = sp.pebbleStore.SetSkippedTicksInterval(nil, &protobuff.SkippedTicksInterval{
					StartTick: sp.lastProcessedTick + 1,
					EndTick:   initialIntervalTick - 1,
				})
				if err != nil {
					return errors.Wrap(err, "appending skipped tick interval")
				}
			}

			chainHash := &[32]byte{}
			storeHash := &[32]byte{}

			for tickNumber := interval.InitialProcessedTick; tickNumber <= interval.LastProcessedTick; tickNumber += sp.maxObjectRequest {

				startTick := tickNumber
				endTick := startTick + sp.maxObjectRequest - 1
				if endTick > interval.LastProcessedTick {
					endTick = interval.LastProcessedTick
				}

				duration := time.Now()

				fetchedTicks, err := sp.fetchTicks(startTick, endTick)
				if err != nil {
					return errors.Wrapf(err, "fetching tick range %d - %d", startTick, endTick)
				}

				processedTicks, err := sp.processTicks(fetchedTicks, initialIntervalTick, qubicComputors, chainHash, storeHash)
				if err != nil {
					return errors.Wrapf(err, "processing tick range %d - %d", startTick, endTick)
				}
				lastProcessedTick, err := sp.storeTicks(processedTicks, epochDelta.Epoch, processedTickIntervalsForEpoch, initialIntervalTick)
				sp.lastProcessedTick = lastProcessedTick
				if err != nil {
					return errors.Wrapf(err, "storing processed tick range %d - %d", startTick, endTick)
				}

				elapsed := time.Since(duration)

				log.Printf("Done processing %d ticks. Took: %v | Average time / tick: %v\n", sp.maxObjectRequest, elapsed, elapsed.Seconds()/float64(sp.maxObjectRequest))

			}

		}
	}
	return nil
}

func (sp *SyncProcessor) fetchTicks(startTick, endTick uint32) ([]*protobuff.SyncTickData, error) {

	//TODO: We are currently fetching a large process of ticks, and using the default will cause the method to error before we are finished
	//ctx, cancel := context.WithTimeout(context.Background(), sp.syncConfiguration.ResponseTimeout)
	//defer cancel()
	ctx := context.Background()

	var compression grpc.CallOption = grpc.EmptyCallOption{}

	if sp.syncConfiguration.EnableCompression {
		compression = grpc.UseCompressor(gzip.Name)
	}

	var responses []*protobuff.SyncTickData

	mutex := sync.RWMutex{}
	routineCount := runtime.NumCPU()
	batchSize := sp.maxObjectRequest / uint32(routineCount)
	errChannel := make(chan error, routineCount)
	var waitGroup sync.WaitGroup
	startTime := time.Now()
	counter := 0

	for index := range routineCount {
		waitGroup.Add(1)

		start := startTick + (batchSize * uint32(index))
		end := start + batchSize - 1

		if end > endTick || index == (int(routineCount)-1) {
			end = endTick
		}

		go func(errChannel chan<- error) {
			defer waitGroup.Done()

			log.Printf("[Routine %d] Fetching tick range %d - %d", index, start, end)

			stream, err := sp.syncServiceClient.SyncGetTickInformation(ctx, &protobuff.SyncTickInfoRequest{
				FistTick: start,
				LastTick: end,
			}, compression)
			if err != nil {
				errChannel <- errors.Wrap(err, "fetching tick information")
				return
			}

			lastTime := time.Now()

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

func (sp *SyncProcessor) processTicks(tickInfoResponses []*protobuff.SyncTickData, initialIntervalTick uint32, computors types.Computors, chainHash *[32]byte, storeHash *[32]byte) (validator.ValidatedTicks, error) {

	syncValidator := validator.NewSyncValidator(initialIntervalTick, computors, tickInfoResponses, sp.processTickTimeout, sp.pebbleStore, chainHash, storeHash)
	validatedTicks, err := syncValidator.Validate()
	if err != nil {
		return nil, errors.Wrap(err, "validating ticks")
	}

	return validatedTicks, nil
}

func (sp *SyncProcessor) storeTicks(validatedTicks validator.ValidatedTicks, epoch uint32, processedTickIntervalsPerEpoch *protobuff.ProcessedTickIntervalsPerEpoch, initialTickInterval uint32) (uint32, error) {

	if epoch == 0 {
		panic("Epoch is not supposed to be 0!!!")
	}

	db := sp.pebbleStore.GetDB()

	batch := db.NewBatch()
	defer batch.Close()

	log.Println("Storing validated ticks...")

	var lastProcessedTick uint32

	for _, validatedTick := range validatedTicks {

		tickNumber := validatedTick.AlignedVotes.QuorumTickStructure.TickNumber

		fmt.Printf("Storing data for tick %d\n", tickNumber)

		quorumDataKey := store.AssembleKey(store.QuorumData, tickNumber)
		serializedData, err := proto.Marshal(validatedTick.AlignedVotes)
		if err != nil {
			return 0, errors.Wrapf(err, "serializing aligned votes for tick %d", tickNumber)
		}
		err = batch.Set(quorumDataKey, serializedData, nil)
		if err != nil {
			return 0, errors.Wrapf(err, "adding aligned votes to batch for tick %d", tickNumber)
		}

		tickDataKey := store.AssembleKey(store.TickData, tickNumber)
		serializedData, err = proto.Marshal(validatedTick.TickData)
		if err != nil {
			return 0, errors.Wrapf(err, "serializing tick data for tick %d", tickNumber)
		}
		err = batch.Set(tickDataKey, serializedData, nil)
		if err != nil {
			return 0, errors.Wrapf(err, "adding tick data to batch for tick %d", tickNumber)
		}

		for _, transaction := range validatedTick.ValidTransactions {
			transactionKey := store.AssembleKey(store.Transaction, transaction.TxId)
			serializedData, err = proto.Marshal(transaction)
			if err != nil {
				return 0, errors.Wrapf(err, "deserializing transaction %s for tick %d", transaction.TxId, tickNumber)
			}
			err = batch.Set(transactionKey, serializedData, nil)
			if err != nil {
				return 0, errors.Wrapf(err, "addin transaction %s to batch for tick %d", transaction.TxId, tickNumber)
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
				return 0, errors.Wrapf(err, "serializing transfer transactions for tickl %d", tickNumber)
			}
			err = batch.Set(identityTransfersPerTickKey, serializedData, nil)
			if err != nil {
				return 0, errors.Wrapf(err, "adding transafer transactions to batch for tick %d", tickNumber)
			}
		}

		tickTxStatusKey := store.AssembleKey(store.TickTransactionsStatus, uint64(tickNumber))
		serializedData, err = proto.Marshal(validatedTick.ApprovedTransactions)
		if err != nil {
			return 0, errors.Wrapf(err, "serializing transaction statuses for tick %d", tickNumber)
		}
		err = batch.Set(tickTxStatusKey, serializedData, nil)
		if err != nil {
			return 0, errors.Wrapf(err, "adding transactions statuses to batch for tick %d", tickNumber)
		}
		for _, transaction := range validatedTick.ApprovedTransactions.Transactions {
			approvedTransactionKey := store.AssembleKey(store.TransactionStatus, transaction.TxId)
			serializedData, err = proto.Marshal(transaction)
			if err != nil {
				return 0, errors.Wrapf(err, "serialzing approved transaction %s for tick %d", transaction.TxId, tickNumber)
			}
			err = batch.Set(approvedTransactionKey, serializedData, nil)
			if err != nil {
				return 0, errors.Wrapf(err, "adding approved transaction %s to batch for tick %d", transaction.TxId, tickNumber)
			}
		}

		chainDigestKey := store.AssembleKey(store.ChainDigest, tickNumber)
		err = batch.Set(chainDigestKey, validatedTick.ChainHash[:], nil)
		if err != nil {
			return 0, errors.Wrapf(err, "adding chain hash to batch for tick %d", tickNumber)
		}

		storeDigestKey := store.AssembleKey(store.StoreDigest, tickNumber)
		err = batch.Set(storeDigestKey, validatedTick.StoreHash[:], nil)
		if err != nil {
			return 0, errors.Wrapf(err, "adding store hash to batch for tick %d", tickNumber)
		}

		lastProcessedTick = tickNumber
	}

	lastProcessedTickPerEpochKey := store.AssembleKey(store.LastProcessedTickPerEpoch, epoch)
	value := make([]byte, 4)
	binary.LittleEndian.PutUint32(value, lastProcessedTick)
	err := batch.Set(lastProcessedTickPerEpochKey, value, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "adding last processed tick %d for epoch %d to batch", lastProcessedTick, epoch)
	}

	lastProcessedTickProto := &protobuff.ProcessedTick{
		TickNumber: lastProcessedTick,
		Epoch:      epoch,
	}
	lastProcessedTickKey := []byte{store.LastProcessedTick}
	serializedData, err := proto.Marshal(lastProcessedTickProto)
	if err != nil {
		return 0, errors.Wrapf(err, "serializing last processed tick %d", lastProcessedTick)
	}
	err = batch.Set(lastProcessedTickKey, serializedData, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "adding last processed tick %d to batch", lastProcessedTick)
	}

	if len(processedTickIntervalsPerEpoch.Intervals) == 0 {
		processedTickIntervalsPerEpoch = &protobuff.ProcessedTickIntervalsPerEpoch{
			Epoch: epoch,
			Intervals: []*protobuff.ProcessedTickInterval{
				{
					InitialProcessedTick: initialTickInterval,
					LastProcessedTick:    lastProcessedTick,
				},
			},
		}
	} else {
		processedTickIntervalsPerEpoch.Intervals[len(processedTickIntervalsPerEpoch.Intervals)-1].LastProcessedTick = lastProcessedTick
	}

	processedTickIntervalsPerEpochKey := store.AssembleKey(store.ProcessedTickIntervals, epoch)
	serializedData, err = proto.Marshal(processedTickIntervalsPerEpoch)
	if err != nil {
		return 0, errors.Wrapf(err, "serializing processed tick intervals for epoch %d", epoch)
	}
	err = batch.Set(processedTickIntervalsPerEpochKey, serializedData, nil)
	if err != nil {
		return 0, errors.Wrapf(err, "adding processed tick intervals for epochg %d to batch", epoch)
	}

	err = batch.Commit(pebble.Sync)
	if err != nil {
		return 0, errors.Wrap(err, "commiting batch")
	}

	return lastProcessedTick, nil
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
