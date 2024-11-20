package store

import (
	"github.com/cockroachdb/pebble"
	"log"
)

type PebbleEventListener struct {
	pebble.EventListener
}

func NewPebbleEventListener() *pebble.EventListener {

	listener := pebble.EventListener{}
	listener.BackgroundError = backgroundError
	listener.CompactionBegin = compactionBegin
	listener.CompactionEnd = compactionEnd
	listener.FlushBegin = flushBegin
	listener.FlushEnd = flushEnd
	listener.WriteStallBegin = writeStallBegin
	listener.WriteStallEnd = writeStallEnd

	return &listener
}

func backgroundError(err error) {

	log.Printf("[PEBBLE]: Encountered background error: %v\n", err)

}

func compactionBegin(info pebble.CompactionInfo) {

	log.Printf("[PEBBLE]: Compaction triggered:\n")
	log.Printf("JobID: %d\n", info.JobID)
	log.Printf("Reason: %s\n", info.Reason)
	for _, level := range info.Input {
		log.Printf("From Level %d - %s\n", level.Level, level.String())
	}
	log.Printf("To level %d %ss\n", info.Output.Level, info.Output.String())

}

func compactionEnd(info pebble.CompactionInfo) {
	log.Printf("[PEBBLE]: Compaction with JobID %d ended. Took %v\n", info.JobID, info.TotalDuration)
}

func flushBegin(info pebble.FlushInfo) {
	log.Printf("[PEBBLE]: Flush triggered:\n")
	log.Printf("JobID: %d\n", info.JobID)
	log.Printf("Reason: %s\n", info.Reason)

}

func flushEnd(info pebble.FlushInfo) {
	log.Printf("[PEBBLE]: Flush with JobID %d ended. Took %v\n", info.JobID, info.TotalDuration)
}

func writeStallBegin(info pebble.WriteStallBeginInfo) {
	log.Printf("[PEBBLE]: Writes stalled. Reason: %s\n", info.Reason)
}

func writeStallEnd() {
	log.Printf("[PEBBLE]: Writes resumed.")
}
