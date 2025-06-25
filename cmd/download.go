package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/vrypan/snapsnapdown/downloader"
	"github.com/vrypan/snapsnapdown/ui"
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

func downloadRun(cmd *cobra.Command, args []string) {

	if len(args) != 1 {
		fmt.Println("Please set the output dir")
		cmd.Help()
		os.Exit(1)
	}
	downloader.OutputBasePath = args[0]

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

	for shard := 0; shard < 3; shard++ {
		metadata, err := downloader.ShardMetadata(endpointURL, shard)
		if err != nil {
			fmt.Println(err)
			return
		}
		shardMetadata[shard] = metadata
	}
	fmt.Printf("\nDownloading Snapshot\n")
	fmt.Printf("Download path: %s\n\n", downloader.OutputBasePath)

	go func() {
		for shard := 0; shard < 3; shard++ {
			downloader.Download(shard, shardMetadata[shard])
		}
		progressChan <- downloader.ProgressUpdate{Quit: true}
	}()

	m := ui.NewDownloadModel(0, shardMetadata, progressChan)
	p := tea.NewProgram(m)

	defer func() {
		if err := p.ReleaseTerminal(); err != nil {
			fmt.Println("failed to restore terminal:", err)
		}
	}()

	if err := p.Start(); err != nil {
		fmt.Println("error:", err)
	}
	if len(m.Errors) > 0 {
		os.Exit(1)
	}
}
func init() {
	rootCmd.AddCommand(downloadCmd)
	downloadCmd.Flags().IntP("jobs", "j", 5, "Number of concurrent downloads.")
	downloadCmd.Flags().String("endpoint", endpointURL, "Snapshot server URL")
	downloadCmd.Flags().Bool("size-checks", true, "If a chunk exists locally, check its size against the remote one.")
	downloadCmd.Flags().Bool("testnet", false, "Use the testnet")
}
