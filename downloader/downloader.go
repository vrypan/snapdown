package downloader

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
)

var (
	Concurrency    = 5
	OutputBasePath = "downloaded_chunks"
	EndpointURL    string
	ProgressChan   chan<- ProgressUpdate
	CheckSizes     = true
	Network        = "MAINNET"
)

type ProgressUpdate struct {
	Shard           int
	ChunkName       string
	Percent         float64
	Done            bool
	BytesDownloaded int64
	BytesTotal      int64
	Quit            bool
	Error           error
}

type Metadata struct {
	KeyBase   string   `json:"key_base"`
	Chunks    []string `json:"chunks"`
	Timestamp int      `json:"timestamp"`
}

func ShardMetadata(endpointURL string, shard int) (*Metadata, error) {
	metadataURL := fmt.Sprintf("%s/FARCASTER_NETWORK_%s/%d/latest.json", endpointURL, Network, shard)

	resp, err := http.Get(metadataURL)
	if err != nil {
		return nil, fmt.Errorf("Error fetching metadata: %v\n", err)
	}
	defer resp.Body.Close()

	var metadata Metadata
	if err := json.NewDecoder(resp.Body).Decode(&metadata); err != nil {
		return nil, fmt.Errorf("Error decoding metadata: %v\n", err)
	}
	return &metadata, nil
}

func isLocalFileComplete(localPath, remoteURL string) (bool, int64, error) {
	info, err := os.Stat(localPath)
	if err != nil {
		return false, 0, err
	}
	localSize := info.Size()
	if !CheckSizes {
		return true, localSize, nil
	}

	resp, err := http.Head(remoteURL)
	if err != nil {
		return false, localSize, err
	}
	defer resp.Body.Close()
	remoteSize := resp.ContentLength
	if remoteSize == -1 {
		return false, localSize, fmt.Errorf("missing Content-Length in response for %s", remoteURL)
	}

	return localSize == remoteSize, localSize, nil
}

func sendProgressUpdate(ch chan<- ProgressUpdate, update ProgressUpdate) {
	select {
	case ch <- update:
	default:
		if update.BytesDownloaded < update.BytesTotal && update.Error == nil {
			return
		} else {
			ch <- update
		}
	}
}

func Download(shard int, metadata *Metadata) {
	progressChan := ProgressChan
	baseURL := fmt.Sprintf("%s/%s", EndpointURL, metadata.KeyBase)
	outputDir := filepath.Join(OutputBasePath, fmt.Sprintf("shard-%d", shard))
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		return
	}
	type chunkJob struct {
		chunk string
		idx   int
	}

	chunkJobs := make(chan chunkJob)
	var wg sync.WaitGroup

	worker := func() {
		buf := make([]byte, 128*1024) // Pre-allocated buffer per worker
		for job := range chunkJobs {
			chunk := job.chunk
			url := fmt.Sprintf("%s/%s", baseURL, chunk)
			if err := downloadChunk(shard, url, filepath.Join(outputDir, chunk), progressChan, chunk, buf); err != nil {
				sendProgressUpdate(progressChan, ProgressUpdate{
					Error: fmt.Errorf("shard=%d, url=%s, path=%s, error=%v", shard, url, filepath.Join(outputDir, chunk), err),
				})
			}
			wg.Done()
		}
	}

	for i := 0; i < Concurrency; i++ {
		go worker()
	}

	for i, chunk := range metadata.Chunks {
		wg.Add(1)
		chunkJobs <- chunkJob{chunk: chunk, idx: i}
	}
	close(chunkJobs)
	wg.Wait()
}

func downloadChunk(shard int, url, path string, progressChan chan<- ProgressUpdate, chunkName string, buf []byte) error {
	if _, err := os.Stat(path); err == nil {
		match, downloadedBytes, err := isLocalFileComplete(path, url)
		if err != nil {
			return fmt.Errorf("  [!] Error checking remote file: %v\n", err)
		} else if match {
			sendProgressUpdate(progressChan, ProgressUpdate{
				Shard: shard, ChunkName: chunkName,
				BytesDownloaded: downloadedBytes, BytesTotal: downloadedBytes,
				Done: true})
			return nil
		}
	}

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("http get failed: %w", err)
	}
	defer resp.Body.Close()

	out, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("create file failed: %w", err)
	}
	defer out.Close()

	total := resp.ContentLength
	if total <= 0 {
		return fmt.Errorf("invalid content length: %d", total)
	}

	const progressStep = 1 * 1024 * 1024
	var downloaded int64
	var lastReported int64

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("write failed: %w", writeErr)
			}
			downloaded += int64(n)
			if downloaded-lastReported >= progressStep || downloaded == total {
				sendProgressUpdate(progressChan, ProgressUpdate{
					Shard: shard, ChunkName: chunkName,
					BytesDownloaded: downloaded,
					BytesTotal:      total,
				})
				lastReported = downloaded
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read failed: %w", err)
		}
	}
	return nil
}
