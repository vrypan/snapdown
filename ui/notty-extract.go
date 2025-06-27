package ui

import (
	"log"

	"github.com/vrypan/snapdown/downloader"
)

type NoTtyExtract struct {
	MaxShard           int
	CurrentShard       int
	CurrentFile        string
	ShardChunks        map[int]int
	ShardChunk         map[int]int
	ShardTotalBytesOut map[int]int64
	updatesCh          <-chan downloader.XUpdMsg
	Errors             []error
}

func NewNoTtyExtract(maxShard int, updates <-chan downloader.XUpdMsg) *NoTtyExtract {
	return &NoTtyExtract{
		MaxShard:           maxShard,
		CurrentShard:       0,
		updatesCh:          updates,
		ShardTotalBytesOut: make(map[int]int64, maxShard+1),
	}
}

func (l *NoTtyExtract) Run() {
	for update := range l.updatesCh {
		switch {
		case update.Error != nil:
			l.Errors = append(l.Errors, update.Error)
			log.Printf("[ERROR] %v\n", update.Error)
		case update.Quit:
			for s, bytes := range l.ShardTotalBytesOut {
				log.Printf("[WROTE] Shard %d - Total bytes out %d (%s)\n", s, bytes, bytesHuman(bytes))
			}
			return
		case update.TotalBytes > 0:
			l.ShardTotalBytesOut[update.Shard] = update.TotalBytes
			l.CurrentFile = update.File
			log.Printf("[WROTE] %s (%d bytes)\n", update.File, update.TotalBytes)
		default:
			l.CurrentShard = update.Shard
		}
	}
}
