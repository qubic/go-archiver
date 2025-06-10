package store

import (
	"encoding/json"
	"fmt"
	"github.com/cockroachdb/pebble"
	"log"
	"net/http"
	"sync"
	"time"
)

type EventListener struct {
	PebbleListener pebble.EventListener
	compactionInfo *PebbleCompactionInfo
	mutex          sync.RWMutex
}

func NewEventListener() *EventListener {
	el := EventListener{}

	listener := pebble.EventListener{}
	listener.BackgroundError = el.backgroundError
	listener.CompactionBegin = el.compactionBegin
	listener.CompactionEnd = el.compactionEnd
	listener.FlushBegin = el.flushBegin
	listener.FlushEnd = el.flushEnd
	listener.WriteStallBegin = el.writeStallBegin
	listener.WriteStallEnd = el.writeStallEnd

	el.PebbleListener = listener
	el.compactionInfo = &PebbleCompactionInfo{
		WritesStalled: false,
		Errors:        make([]PebbleErrorWithTimeStamp, 0),
		Compactions:   make(map[int]*PebbleCompaction),
		Flushes:       make(map[int]*PebbleFlush),
	}

	return &el
}

func (el *EventListener) backgroundError(err error) {
	log.Printf("[PEBBLE]: Encountered background error: %v\n", err)

	el.mutex.Lock()
	defer el.mutex.Unlock()

	pebbleError := PebbleErrorWithTimeStamp{
		Err:       err,
		Timestamp: time.Now().UTC().Format(time.RFC822),
	}
	el.compactionInfo.Errors = append(el.compactionInfo.Errors, pebbleError)
}

func (el *EventListener) compactionBegin(info pebble.CompactionInfo) {
	log.Printf("[PEBBLE]: Compaction triggered:\n")
	log.Printf("  JobID: %d\n", info.JobID)
	log.Printf("  Reason: %s\n", info.Reason)

	var fromLevels []PebbleLevel
	for _, level := range info.Input {
		log.Printf("  From Level %d - %s\n", level.Level, level.String())
		fromLevels = append(fromLevels, PebbleLevel{
			Level:       level.Level,
			Description: level.String(),
		})
	}

	log.Printf("  To level %d %ss\n", info.Output.Level, info.Output.String())

	el.mutex.Lock()
	defer el.mutex.Unlock()

	el.compactionInfo.Compactions[info.JobID] = &PebbleCompaction{
		PebbleDiskOperation: PebbleDiskOperation{
			Reason:    info.Reason,
			Finished:  false,
			StartedAt: time.Now().UTC().Format(time.RFC822),
			EndedAt:   "",
			Duration:  "",
		},
		From: fromLevels,
		To: PebbleLevel{
			Level:       info.Output.Level,
			Description: info.Output.String(),
		},
	}

}

func (el *EventListener) compactionEnd(info pebble.CompactionInfo) {
	log.Printf("[PEBBLE]: Compaction with JobID %d ended. Took %v\n", info.JobID, info.TotalDuration)

	el.mutex.Lock()
	defer el.mutex.Unlock()

	el.compactionInfo.Compactions[info.JobID].Finished = true
	el.compactionInfo.Compactions[info.JobID].EndedAt = time.Now().UTC().Format(time.RFC822)
	el.compactionInfo.Compactions[info.JobID].Duration = info.TotalDuration.String()
}

func (el *EventListener) flushBegin(info pebble.FlushInfo) {
	log.Printf("[PEBBLE]: Flush triggered:\n")
	log.Printf("  JobID: %d\n", info.JobID)
	log.Printf("  Reason: %s\n", info.Reason)

	el.mutex.Lock()
	defer el.mutex.Unlock()

	el.compactionInfo.Flushes[info.JobID] = &PebbleFlush{
		PebbleDiskOperation{
			Reason:    info.Reason,
			Finished:  false,
			StartedAt: time.Now().UTC().Format(time.RFC822),
			EndedAt:   "",
			Duration:  "",
		},
	}
}

func (el *EventListener) flushEnd(info pebble.FlushInfo) {
	log.Printf("[PEBBLE]: Flush with JobID %d ended. Took %v\n", info.JobID, info.TotalDuration)

	el.mutex.Lock()
	defer el.mutex.Unlock()

	el.compactionInfo.Flushes[info.JobID].EndedAt = time.Now().UTC().Format(time.RFC822)
	el.compactionInfo.Flushes[info.JobID].Duration = info.TotalDuration.String()
}

func (el *EventListener) writeStallBegin(info pebble.WriteStallBeginInfo) {
	log.Printf("[PEBBLE]: Writes stalled. Reason: %s\n", info.Reason)

	el.mutex.Lock()
	defer el.mutex.Unlock()

	el.compactionInfo.WritesStalled = true
}

func (el *EventListener) writeStallEnd() {
	log.Printf("[PEBBLE]: Writes resumed.")

	el.mutex.Lock()
	defer el.mutex.Unlock()

	el.compactionInfo.WritesStalled = false
}

type PebbleErrorWithTimeStamp struct {
	Err       error  `json:"err"`
	Timestamp string `json:"timestamp"`
}

type PebbleLevel struct {
	Level       int    `json:"level"`
	Description string `json:"description"`
}

type PebbleDiskOperation struct {
	Reason    string `json:"reason"`
	Finished  bool   `json:"finished"`
	StartedAt string `json:"startedAt"`
	EndedAt   string `json:"endedAt"`
	Duration  string `json:"duration"`
}

type PebbleCompaction struct {
	PebbleDiskOperation
	From []PebbleLevel `json:"from"`
	To   PebbleLevel   `json:"to"`
}
type PebbleFlush struct {
	PebbleDiskOperation
}

type PebbleCompactionInfo struct {
	WritesStalled bool                       `json:"writesStalled"`
	Errors        []PebbleErrorWithTimeStamp `json:"errors"`
	Compactions   map[int]*PebbleCompaction  `json:"compactions"`
	Flushes       map[int]*PebbleFlush       `json:"flushes"`
}

func (el *EventListener) HandleCompactionInfoEndpoint(w http.ResponseWriter, _ *http.Request) {

	el.mutex.RLock()
	compactionInfo := el.compactionInfo
	el.mutex.RUnlock()

	data, err := json.Marshal(compactionInfo)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)

		_, err := w.Write([]byte(err.Error()))
		if err != nil {
			fmt.Printf("Failed to return error on compaction info endpoint: %v\n", err)
		}
	}

	_, err = w.Write(data)
	if err != nil {
		_, err := w.Write([]byte(err.Error()))
		if err != nil {
			fmt.Printf("Failed to return compaction info: %v\n", err)
		}
	}

}
