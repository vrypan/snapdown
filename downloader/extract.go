package downloader

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
)

type XUpdMsg struct {
	Shard      int
	Idx        int
	Total      int
	File       string
	TotalBytes int64
	Error      error
}

// chainedFileReader implements io.Reader by reading sequentially from a list of files.
// It ensures files are closed as soon as they're done, and reuses the buffer efficiently.
type chainedFileReader struct {
	files []string
	index int
	curr  *os.File
}

func newChainedFileReader(files []string) *chainedFileReader {
	return &chainedFileReader{files: files, index: 0}
}

func (r *chainedFileReader) CurrentFileIndex() int {
	if r.index == 0 && r.curr == nil {
		return 0
	}
	return r.index
}
func (r *chainedFileReader) Read(p []byte) (int, error) {
	for {
		if r.curr == nil {
			if r.index >= len(r.files) {
				return 0, io.EOF
			}
			file, err := os.Open(r.files[r.index])
			if err != nil {
				return 0, err
			}
			r.curr = file
		}

		n, err := r.curr.Read(p)
		if err == io.EOF {
			r.curr.Close()
			r.curr = nil
			r.index++
			if n > 0 {
				return n, nil
			}
			continue
		}
		return n, err
	}
}

// Extract untars and ungzips concatenated parts from srcDir into dstDir.
func Extract(rootSrcDir, dstDir string, shardId int, progressCh chan<- XUpdMsg) {
	var totalBytesWritten int64 = 0
	srcDir := filepath.Join(rootSrcDir, fmt.Sprintf("shard-%d", shardId))
	entries, err := os.ReadDir(srcDir)
	if err != nil {
		panic(err)
	}

	fileNames := make([]string, 0, len(entries))
	for _, f := range entries {
		if f.Type().IsRegular() {
			fileNames = append(fileNames, filepath.Join(srcDir, f.Name()))
		}
	}
	sort.Strings(fileNames)
	if len(fileNames) == 0 {
		panic("no files to extract")
	}

	reader := newChainedFileReader(fileNames)
	gzipReader, err := gzip.NewReader(reader)
	if err != nil {
		panic(err)
	}
	defer gzipReader.Close()

	tarReader := tar.NewReader(gzipReader)
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			panic(err)
		}

		targetPath := filepath.Join(dstDir, header.Name)
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(targetPath, os.FileMode(header.Mode)); err != nil {
				progressCh <- XUpdMsg{
					Total: len(fileNames),
					Shard: shardId,
					Idx:   reader.CurrentFileIndex() + 1,
					File:  targetPath,
					Error: err,
				}
				return
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
				progressCh <- XUpdMsg{
					Total: len(fileNames),
					Shard: shardId,
					Idx:   reader.CurrentFileIndex() + 1,
					File:  targetPath,
					Error: err,
				}
				return
			}
			outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				progressCh <- XUpdMsg{
					Total: len(fileNames),
					Shard: shardId,
					Idx:   reader.CurrentFileIndex() + 1,
					File:  targetPath,
					Error: err,
				}
				return
			}
			bytesOut, err := io.Copy(outFile, tarReader)
			if err != nil {
				outFile.Close()
				progressCh <- XUpdMsg{
					Total:      len(fileNames),
					Shard:      shardId,
					Idx:        reader.CurrentFileIndex() + 1,
					File:       targetPath,
					TotalBytes: totalBytesWritten,
					Error:      err,
				}
				return
			}
			outFile.Close()
			totalBytesWritten += bytesOut
			//fmt.Printf("%d %d/%d %s\n", shardId, reader.CurrentFileIndex()+1, len(fileNames), targetPath)
			progressCh <- XUpdMsg{
				Total:      len(fileNames),
				Shard:      shardId,
				Idx:        reader.CurrentFileIndex() + 1,
				File:       targetPath,
				TotalBytes: totalBytesWritten,
			}

		default:
			continue
		}
	}
}
