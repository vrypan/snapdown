//go:build windows

package downloader

func getOptimalBufferSize(path string) int {
	// On Windows, just use the default safe size
	return 100 * 1024 // 100 KB
}
