package downloader

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
	// replace with actual module name
)

var (
	Concurrency    = 5
	OutputBasePath = "downloaded_chunks"
	EndpointURL    string
	ProgressChan   chan<- ProgressUpdate
)

type ProgressUpdate struct {
	ChunkName string
	Percent   float64
}

type Metadata struct {
	KeyBase string   `json:"key_base"`
	Chunks  []string `json:"chunks"`
}

func ShardMetadata(endpointURL string, shard int) (*Metadata, error) {
	metadataURL := fmt.Sprintf("%s/FARCASTER_NETWORK_MAINNET/%d/latest.json", endpointURL, shard)
	//fmt.Printf("[→] Fetching metadata: %s\n", metadataURL)

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

func isLocalFileComplete(localPath, remoteURL string) (bool, error) {
	info, err := os.Stat(localPath)
	if err != nil {
		return false, err
	}
	localSize := info.Size()
	resp, err := http.Head(remoteURL)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()
	remoteSize := resp.ContentLength
	if remoteSize == -1 {
		return false, fmt.Errorf("missing Content-Length in response for %s", remoteURL)
	}

	// Compare sizes
	return localSize == remoteSize, nil
}

func Download(shard int, metadata *Metadata) {
	progressChan := ProgressChan
	baseURL := fmt.Sprintf("%s/%s", EndpointURL, metadata.KeyBase)
	outputDir := filepath.Join(OutputBasePath, fmt.Sprintf("shard-%d", shard))
	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		return
	}
	var wg sync.WaitGroup
	sem := make(chan struct{}, Concurrency)

	for i, chunk := range metadata.Chunks {
		wg.Add(1)
		sem <- struct{}{} // acquire slot
		go func(chunk string, idx int) {
			defer wg.Done()
			defer func() { <-sem }() // release slot
			url := fmt.Sprintf("%s/%s", baseURL, chunk)
			progressChan <- ProgressUpdate{ChunkName: chunk, Percent: 0.0}
			if err := downloadChunk2(url, filepath.Join(outputDir, chunk), progressChan, chunk); err != nil {
				fmt.Printf("  [!] Error downloading %s: %v\n", chunk, err)
			}
			// This will instruct the UI to remove this chunk from its list
			time.Sleep(100 * time.Millisecond)
			progressChan <- ProgressUpdate{ChunkName: chunk, Percent: 200.0}
		}(chunk, i)
	}

	wg.Wait()
}

func downloadChunk(url, path string) error {
	if _, err := os.Stat(path); err == nil {
		match, err := isLocalFileComplete(path, url)
		if err != nil {
			return fmt.Errorf("  [!] Error checking remote remote file: %v\n", err)
		} else if match {
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

	_, err = io.Copy(out, resp.Body)

	return err
}

func downloadChunk2(url, path string, progressChan chan<- ProgressUpdate, chunkName string) error {
	if _, err := os.Stat(path); err == nil {
		match, err := isLocalFileComplete(path, url)
		if err != nil {
			return fmt.Errorf("  [!] Error checking remote file: %v\n", err)
		} else if match {
			// Send complete progress for skipped files
			progressChan <- ProgressUpdate{ChunkName: chunkName, Percent: 1.0}
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

	// Get total size
	total := resp.ContentLength
	if total <= 0 {
		return fmt.Errorf("invalid content length: %d", total)
	}

	// Set up progress tracking
	var downloaded int64
	buf := make([]byte, 32*1024) // 32 KB buffer

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, writeErr := out.Write(buf[:n]); writeErr != nil {
				return fmt.Errorf("write failed: %w", writeErr)
			}
			downloaded += int64(n)
			percent := float64(downloaded) / float64(total)
			progressChan <- ProgressUpdate{ChunkName: chunkName, Percent: percent}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return fmt.Errorf("read failed: %w", err)
		}
	}

	// Ensure final update is 100%
	progressChan <- ProgressUpdate{ChunkName: chunkName, Percent: 1.0}

	return nil
}
