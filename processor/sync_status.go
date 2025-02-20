package processor

import (
	"github.com/qubic/go-archiver/protobuff"
	"google.golang.org/protobuf/proto"
	"sync"
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
