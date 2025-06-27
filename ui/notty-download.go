package ui

import (
	"log"

	"github.com/vrypan/snapdown/downloader"
)

type NoTTYDownload struct {
	ShardMetadata map[int]*downloader.Metadata
	progressChan  <-chan downloader.ProgressUpdate
	Errors        []error
	MaxJobs       int
}

func NewNoTTYDownload(metadata map[int]*downloader.Metadata, progressChan <-chan downloader.ProgressUpdate, maxJobs int) *NoTTYDownload {
	return &NoTTYDownload{
		ShardMetadata: metadata,
		MaxJobs:       maxJobs,
		progressChan:  progressChan,
	}
}

func (d *NoTTYDownload) Run() {
	for update := range d.progressChan {
		if update.Quit {
			return
		}

		if update.Error != nil {
			d.Errors = append(d.Errors, update.Error)
			log.Printf("[ERROR] %v\n", update.Error)
			continue
		}

		if update.BytesDownloaded == update.BytesTotal {
			log.Printf("[DOWNLOADED] Shard %d - %s (%d bytes)\n", update.Shard, update.ChunkName, update.BytesDownloaded)
		}
	}
	log.Println("Download progress channel closed. Exiting.")
}
