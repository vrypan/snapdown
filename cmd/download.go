package cmd

import (
	"fmt"

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
	Use:   "download",
	Short: "Download the current snapshot",
	Long:  ``,
	Run:   downloadRun,
}

func downloadRun(cmd *cobra.Command, args []string) {

	concurrentJobs, _ := cmd.Flags().GetInt("jobs")
	if concurrentJobs != 0 {
		downloader.Concurrency = 5
	}
	outputDir, _ := cmd.Flags().GetString("output")
	if outputDir != "" {
		downloader.OutputBasePath = outputDir
	}
	endpoint, _ := cmd.Flags().GetString("endpoint")
	if endpoint != "" {
		endpointURL = endpoint
	}

	progressChan := make(chan downloader.ProgressUpdate, 100)
	shardMetadata := make(map[int]*downloader.Metadata)
	downloader.EndpointURL = endpointURL
	downloader.ProgressChan = progressChan

	for shard := 0; shard < 3; shard++ {
		metadata, err := downloader.ShardMetadata(endpointURL, shard)
		if err != nil {
			fmt.Println(err)
			return
		}
		shardMetadata[shard] = metadata
	}
	fmt.Printf("\nSnapchain Snapshot Downloader\n")
	fmt.Printf("Download path: %s\n\n", downloader.OutputBasePath)

	m := ui.NewModel(0, shardMetadata, progressChan)
	p := tea.NewProgram(m)

	defer func() {
		if err := p.ReleaseTerminal(); err != nil {
			fmt.Println("failed to restore terminal:", err)
		}
	}()

	if err := p.Start(); err != nil {
		fmt.Println("error:", err)
	}
}
func init() {
	rootCmd.AddCommand(downloadCmd)
	downloadCmd.Flags().IntP("jobs", "j", 5, "Number of concurrent downloads.")
	downloadCmd.Flags().StringP("output", "o", "./snapshot", "Output directory")
	downloadCmd.Flags().String("endpoint", endpointURL, "Snapshot server URL")
}
