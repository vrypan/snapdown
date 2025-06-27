package ui

import (
	"log"

	"github.com/vrypan/snapdown/downloader"
)

type NoTTYUnpack struct {
	MaxShard           int
	CurrentShard       int
	CurrentFile        string
	ShardChunks        map[int]int
	ShardChunk         map[int]int
	ShardTotalBytesOut map[int]int64
	updatesCh          <-chan downloader.XUpdMsg
	Errors             []error
}

func NewNoTTYUnpack(maxShard int, updates <-chan downloader.XUpdMsg) *NoTTYUnpack {
	return &NoTTYUnpack{
		MaxShard:           maxShard,
		CurrentShard:       0,
		updatesCh:          updates,
		ShardChunks:        make(map[int]int, maxShard+1),
		ShardChunk:         make(map[int]int, maxShard+1),
		ShardTotalBytesOut: make(map[int]int64, maxShard+1),
	}
}

func (l *NoTTYUnpack) Run() {
	for update := range l.updatesCh {
		switch {
		case update.Error != nil:
			l.Errors = append(l.Errors, update.Error)
			log.Printf("[ERROR] %v\n", update.Error)
		case update.Quit:
			for s, bytes := range l.ShardTotalBytesOut {
				log.Printf("Total bytes out for shard-%d: %s\n", s, bytesHuman(bytes))
			}
			return
		case update.TotalBytes > 0:
			l.ShardTotalBytesOut[update.Shard] = update.TotalBytes
			l.CurrentFile = update.File
			log.Printf("[WROTE] %s (%d bytes)\n", update.File, update.TotalBytes)
		default:
			l.CurrentShard = update.Shard
			l.ShardChunks[update.Shard] = update.Total
			l.ShardChunk[update.Shard] = update.Idx
		}
	}
}
