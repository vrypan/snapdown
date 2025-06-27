package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/cobra"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vrypan/snapdown/downloader"
	"github.com/vrypan/snapdown/ui"
)

var (
	endpointURL = "https://pub-d352dd8819104a778e20d08888c5a661.r2.dev"
)

// downloadCmd represents the download command
var downloadCmd = &cobra.Command{
	Use:     "download <output directory>",
	Aliases: []string{"d"},
	Short:   "Download the current snapshot",
	Long:    ``,
	Run:     downloadRun,
}

// formatRelativeTime returns a user-friendly string for duration ago
func formatRelativeTime(timestampMs int64) string {
	t := timestampMs / 1000 // Convert ms to seconds
	now := time.Now().Unix()
	diff := now - t
	switch {
	case diff < 60:
		return fmt.Sprintf("%ds ago", diff)
	case diff < 3600:
		return fmt.Sprintf("%dm ago", diff/60)
	case diff < 86400:
		return fmt.Sprintf("%dh ago", diff/3600)
	default:
		return fmt.Sprintf("%dd ago", diff/86400)
	}
}

func mustMkdirAll(path string) {
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}
}

func mustReadFile(path string) []byte {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Printf("Failed to read %s: %v\n", path, err)
		os.Exit(1)
	}
	return data
}

func mustWriteFile(path string, data []byte) {
	if err := os.WriteFile(path, data, 0644); err != nil {
		fmt.Printf("Failed to write %s: %v\n", path, err)
		os.Exit(1)
	}
}

func mustUnmarshalMetadata(data []byte, meta *map[int]*downloader.Metadata) {
	if err := json.Unmarshal(data, meta); err != nil {
		fmt.Printf("Failed to parse metadata: %v\n", err)
		os.Exit(1)
	}
}

func mustMarshalMetadata(meta map[int]*downloader.Metadata) []byte {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		fmt.Printf("Failed to serialize shard metadata: %v\n", err)
		os.Exit(1)
	}
	return data
}

func fetchShardMetadata(endpoint string, shards []int) map[int]*downloader.Metadata {
	shardMetadata := make(map[int]*downloader.Metadata)
	for _, shard := range shards {
		metadata, err := downloader.ShardMetadata(endpoint, shard)
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		shardMetadata[shard] = metadata
	}
	return shardMetadata
}

func printShardAges(shardMetadata map[int]*downloader.Metadata) {
	fmt.Printf("Snapshot Ages per shard: ")
	for _, s := range shardMetadata {
		fmt.Printf(" [%s]", formatRelativeTime(int64(s.Timestamp)))
	}
	fmt.Println()
}

func downloadRun(cmd *cobra.Command, args []string) {
	if len(args) != 1 {
		fmt.Println("Please set the output dir")
		os.Exit(1)
	}
	outputDir := args[0]
	downloader.OutputBasePath = outputDir

	concurrentJobs, _ := cmd.Flags().GetInt("jobs")
	if concurrentJobs != 0 {
		downloader.Concurrency = 5
	}

	endpoint, _ := cmd.Flags().GetString("endpoint")
	if endpoint != "" {
		endpointURL = endpoint
	}
	sizeChecks, _ := cmd.Flags().GetBool("size-checks")
	useTestnet, _ := cmd.Flags().GetBool("testnet")
	notty, _ := cmd.Flags().GetBool("no-tty")

	progressChan := make(chan downloader.ProgressUpdate, 1000)
	shardMetadata := make(map[int]*downloader.Metadata)
	downloader.EndpointURL = endpointURL
	downloader.ProgressChan = progressChan
	downloader.CheckSizes = sizeChecks
	if useTestnet {
		downloader.Network = "TESTNET"
	}

	mustMkdirAll(outputDir)

	metadataFilePath := filepath.Join(outputDir, "metadata.json")
	shards := []int{0, 1, 2}

	// Load or fetch shard metadata
	if _, err := os.Stat(metadataFilePath); err == nil {
		// Existing metadata: resume
		metadataData := mustReadFile(metadataFilePath)
		mustUnmarshalMetadata(metadataData, &shardMetadata)

		fmt.Printf("\nResuming Snapshot Download\n")
		printShardAges(shardMetadata)
	} else {
		// Fresh: fetch metadata from remote
		shardMetadata = fetchShardMetadata(endpointURL, shards)
		metadataJson := mustMarshalMetadata(shardMetadata)
		mustWriteFile(metadataFilePath, metadataJson)

		fmt.Printf("\nDownloading Latest Snapshot\n")
	}

	fmt.Printf("Download path: %s\n\n", downloader.OutputBasePath)

	go func() {
		for _, shard := range shards {
			downloader.Download(shard, shardMetadata[shard])
		}
		progressChan <- downloader.ProgressUpdate{Quit: true}
	}()

	if notty {
		// Use plain text output
		nottyModel := ui.NewNoTTYDownload(shardMetadata, progressChan, concurrentJobs)
		nottyModel.Run()
		if len(nottyModel.Errors) > 0 {
			os.Exit(1)
		}
	} else {
		// Use fancy bubbletea interfcae
		m := ui.NewTtyDownload(0, shardMetadata, progressChan, concurrentJobs)
		p := tea.NewProgram(m)

		defer func() {
			if err := p.ReleaseTerminal(); err != nil {
				fmt.Println("failed to restore terminal:", err)
			}
		}()

		finalModel, err := p.Run()
		if err != nil {
			fmt.Println("error:", err)
		}
		downloadModel := finalModel.(ui.TtyDownload)

		if len(downloadModel.Errors) > 0 {
			for _, e := range downloadModel.Errors {
				fmt.Println(e)
			}
			os.Exit(1)
		}
	}
}

func init() {
	rootCmd.AddCommand(downloadCmd)
	downloadCmd.Flags().IntP("jobs", "j", 5, "Number of concurrent downloads.")
	downloadCmd.Flags().String("endpoint", endpointURL, "Snapshot server URL")
	downloadCmd.Flags().Bool("size-checks", true, "If a chunk exists locally, check its size against the remote one.")
	downloadCmd.Flags().Bool("testnet", false, "Use the testnet")
	downloadCmd.Flags().Bool("no-tty", false, "Plan text output")
}
