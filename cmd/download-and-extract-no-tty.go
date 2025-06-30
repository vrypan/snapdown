package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/vrypan/snapdown/downloader"
	"github.com/vrypan/snapdown/ui"
)

// dxCmd represents the download command
var dxCmd = &cobra.Command{
	Use:     "dx <download dir> <export dir>",
	Aliases: []string{""},
	Short:   "Download and Extract the current snapshot",
	Long: `This is equivalent to
  snapdown download <download dir> --no-tty && \
  snapdown extract <download dir> <export dir> --no-tty`,
	Run: dxRun,
}

func dxRun(cmd *cobra.Command, args []string) {
	if len(args) != 2 {
		fmt.Println("Please set download dir and output dir.")
		os.Exit(1)
	}

	downloadDir := args[0]
	outputDir := args[1]
	downloader.OutputBasePath = downloadDir

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

	progressChan := make(chan downloader.ProgressUpdate, 1000)
	shardMetadata := make(map[int]*downloader.Metadata)
	downloader.EndpointURL = endpointURL
	downloader.ProgressChan = progressChan
	downloader.CheckSizes = sizeChecks
	if useTestnet {
		downloader.Network = "TESTNET"
	}

	mustMkdirAll(downloadDir)

	metadataFilePath := filepath.Join(downloadDir, "metadata.json")
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

	nottyModel := ui.NewNoTTYDownload(shardMetadata, progressChan, concurrentJobs)
	nottyModel.Run()
	if len(nottyModel.Errors) > 0 {
		os.Exit(1)
	}

	if len(args) < 2 {
		fmt.Println("Provide input and output dirs")
		os.Exit(1)
	}

	if !downloader.HasTarInPath() {
		fmt.Println("Error: 'tar' not found in PATH.")
		os.Exit(1)
	}

	// Extract

	progressCh := make(chan downloader.XUpdMsg, 1000)
	fmt.Printf("\nExtracting Snapshot [%s] -> [%s]\n\n", downloadDir, outputDir)
	go func() {
		for _, shard := range shards {
			downloader.ExtractWithNativeTar(downloadDir, outputDir, shard, progressCh)
		}
		progressCh <- downloader.XUpdMsg{Quit: true}
	}()
	var maxShard = len(shards) - 1
	runNoTtyExtraction(maxShard, progressCh)

}

func init() {
	rootCmd.AddCommand(dxCmd)
	dxCmd.Flags().IntP("jobs", "j", 5, "Number of concurrent downloads.")
	dxCmd.Flags().String("endpoint", endpointURL, "Snapshot server URL")
	dxCmd.Flags().Bool("size-checks", true, "If a chunk exists locally, check its size against the remote one.")
	dxCmd.Flags().Bool("testnet", false, "Use the testnet")
}
