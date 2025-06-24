//go:build linux || darwin

package downloader

import (
	"golang.org/x/sys/unix"
)

func getOptimalBufferSize(path string) int {
	var stat unix.Statfs_t
	err := unix.Statfs(path, &stat)
	if err != nil {
		// Fallback
		return 100 * 1024
	}

	blockSize := int(stat.Bsize)
	bufferSize := blockSize * 128
	if bufferSize < 64*1024 {
		return 64 * 1024
	}
	if bufferSize > 4*1024*1024 {
		return 4 * 1024 * 1024
	}
	return bufferSize
}
