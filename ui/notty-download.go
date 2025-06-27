package ui

import (
	"log"

	"github.com/vrypan/snapdown/downloader"
)

type NoTTYDownload struct {
	ShardMetadata map[int]*downloader.Metadata
	progressChan  <-chan downloader.ProgressUpdate
	shardBytes    map[int]int64
	Errors        []error
	MaxJobs       int
}

func NewNoTTYDownload(metadata map[int]*downloader.Metadata, progressChan <-chan downloader.ProgressUpdate, maxJobs int) *NoTTYDownload {
	return &NoTTYDownload{
		ShardMetadata: metadata,
		MaxJobs:       maxJobs,
		progressChan:  progressChan,
		shardBytes:    make(map[int]int64),
	}
}

func (d *NoTTYDownload) Run() {
	for update := range d.progressChan {
		if update.Quit {
			for s, bytes := range d.shardBytes {
				log.Printf("[DOWNLOADED] Shard %d - Total bytes in %d (%s)\n", s, bytes, bytesHuman(bytes))
			}
			return
		}
		if update.Error != nil {
			d.Errors = append(d.Errors, update.Error)
			log.Printf("[ERROR] %v\n", update.Error)
			continue
		}
		if update.BytesDownloaded == update.BytesTotal {
			d.shardBytes[update.Shard] += update.BytesTotal
			log.Printf("[DOWNLOADED] Shard %d - %s (%d bytes)\n", update.Shard, update.ChunkName, update.BytesDownloaded)
		}
	}
}
