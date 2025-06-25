package downloader

/*
OK, This file may seem strage, because it actually uses tar
to extract files. It turns out that using tar is ~30% faster
than my Go-native implementation. Totally worth the hack.
*/

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type XUpdMsg struct {
	Shard      int
	Idx        int
	Total      int
	File       string
	TotalBytes int64
	Error      error
	Quit       bool
}

func ExtractWithNativeTar(rootSrcDir, dstDir string, shardId int, progressCh chan<- XUpdMsg) {
	srcDir := filepath.Join(rootSrcDir, fmt.Sprintf("shard-%d", shardId))
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		progressCh <- XUpdMsg{
			Shard: shardId,
			Error: err,
			Quit:  true,
		}
		return
	}

	// Preallocate fileNames slice and use append-less assignment for better efficiency
	fileNames := make([]string, 0, len(entries))
	for _, f := range entries {
		if f.Type().IsRegular() {
			fileNames = append(fileNames, filepath.Join(srcDir, f.Name()))
		}
	}
	sort.Strings(fileNames)
	if len(fileNames) == 0 {
		progressCh <- XUpdMsg{
			Shard: shardId,
			Error: fmt.Errorf("no files to extract"),
			Quit:  true,
		}
		return
	}

	if err := os.MkdirAll(dstDir, os.ModePerm); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		return
	}
	cmd := exec.Command("tar", "xzvf", "-", "-C", dstDir)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		progressCh <- XUpdMsg{Shard: shardId, Error: err, Quit: true}
		return
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		progressCh <- XUpdMsg{Shard: shardId, Error: err, Quit: true}
		return
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		progressCh <- XUpdMsg{Shard: shardId, Error: err, Quit: true}
		return
	}

	if err := cmd.Start(); err != nil {
		progressCh <- XUpdMsg{
			Shard: shardId,
			Error: err,
			Quit:  true,
		}
		return
	}

	var wg sync.WaitGroup
	wg.Add(3)
	go trackTarOutput(stdout, dstDir, shardId, progressCh, &wg)
	go trackTarOutput(stderr, dstDir, shardId, progressCh, &wg)

	// Stream input files into tar's stdin
	go func() {
		defer wg.Done()
		defer stdin.Close()
		buf := make([]byte, 1<<20) // Use a 1MB buffer for efficient file copy
		for i, filePath := range fileNames {
			file, err := os.Open(filePath)
			if err != nil {
				progressCh <- XUpdMsg{
					Shard: shardId,
					Error: err,
				}
				return
			}
			_, err = io.CopyBuffer(stdin, file, buf)
			file.Close()
			if err != nil {
				progressCh <- XUpdMsg{
					Shard: shardId,
					Error: err,
				}
				return
			}

			// Track input file progress here
			progressCh <- XUpdMsg{
				Total: len(fileNames),
				Shard: shardId,
				Idx:   i + 1, // Input file index
				//File:  filePath,
			}
		}
	}()

	// Wait for background goroutines to finish and then cmd.Wait
	wg.Wait()
	if err := cmd.Wait(); err != nil {
		progressCh <- XUpdMsg{
			Shard: shardId,
			Error: err,
			Quit:  true,
		}
		return
	}
}

func trackTarOutput(r io.ReadCloser, dstDir string, shardId int, progressCh chan<- XUpdMsg, wg *sync.WaitGroup) {
	defer wg.Done()
	defer r.Close()
	scanner := bufio.NewScanner(r)
	var (
		lastFilePath      string
		totalBytesWritten int64 = 0
	)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		var fileName string
		if strings.HasPrefix(line, "x ") {
			fileName = strings.TrimPrefix(line, "x ")
		} else {
			fileName = line
		}
		fullPath := filepath.Join(dstDir, fileName)

		// Report previous file size now (because it's done)
		if lastFilePath != "" {
			fileInfo, err := os.Lstat(lastFilePath)
			var size int64 = 0
			if err == nil && fileInfo.Mode().IsRegular() {
				size = fileInfo.Size()
			}

			totalBytesWritten += size

			// Send update message for the completed file
			progressCh <- XUpdMsg{
				Shard:      shardId,
				TotalBytes: totalBytesWritten,
				File:       lastFilePath,
			}
		}

		// Track current file for the next iteration
		lastFilePath = fullPath
	}

	// After loop ends, report the final file (it doesn't have a "next" file to trigger it)
	if lastFilePath != "" {
		fileInfo, err := os.Lstat(lastFilePath)
		var size int64 = 0
		if err == nil && fileInfo.Mode().IsRegular() {
			size = fileInfo.Size()
		}

		totalBytesWritten += size

		progressCh <- XUpdMsg{
			Shard:      shardId,
			TotalBytes: totalBytesWritten,
			File:       lastFilePath,
		}
	}

	if err := scanner.Err(); err != nil {
		progressCh <- XUpdMsg{
			Shard: shardId,
			Error: err,
		}
	}
}
