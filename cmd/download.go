package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

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

	progressChan := make(chan downloader.ProgressUpdate, 1000)
	shardMetadata := make(map[int]*downloader.Metadata)
	downloader.EndpointURL = endpointURL
	downloader.ProgressChan = progressChan
	downloader.CheckSizes = sizeChecks
	if useTestnet {
		downloader.Network = "TESTNET"
	}

	if err := os.MkdirAll(outputDir, os.ModePerm); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	// Check if metadata.json already exists and, if so, load it
	metadataFilePath := outputDir + string(os.PathSeparator) + "metadata.json"
	if _, err := os.Stat(metadataFilePath); err == nil {
		metadataData, err := os.ReadFile(metadataFilePath)
		if err != nil {
			fmt.Printf("Failed to read existing metadata.json: %v\n", err)
			os.Exit(1)
		}
		err = json.Unmarshal(metadataData, &shardMetadata)
		if err != nil {
			fmt.Printf("Failed to parse existing metadata.json: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nResuming Snapshot Download\n")
		fmt.Printf("Snapshot Ages per shard: ")
		for _, s := range shardMetadata {
			// Convert timestamp (assumed unix seconds) to "X ago" format
			t := int64(s.Timestamp) / 1000
			now := time.Now().Unix()
			diff := now - t
			var rel string
			switch {
			case diff < 60:
				rel = fmt.Sprintf("%ds ago", diff)
			case diff < 3600:
				rel = fmt.Sprintf("%dm ago", diff/60)
			case diff < 86400:
				rel = fmt.Sprintf("%dh ago", diff/3600)
			default:
				rel = fmt.Sprintf("%dd ago", diff/86400)
			}
			fmt.Printf(" [%s]", rel)
		}
		fmt.Println()
	} else {
		for _, shard := range []int{0, 1, 2} {
			metadata, err := downloader.ShardMetadata(endpointURL, shard)
			if err != nil {
				fmt.Println(err)
				return
			}
			shardMetadata[shard] = metadata
		}
		// Serialize shardMetadata as JSON and save to outputDir/metadata.json
		metadataJson, err := json.MarshalIndent(shardMetadata, "", "  ")
		if err != nil {
			fmt.Printf("Failed to serialize shard metadata: %v\n", err)
			os.Exit(1)
		}
		err = os.WriteFile(metadataFilePath, metadataJson, 0644)
		if err != nil {
			fmt.Printf("Failed to write metadata.json: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("\nDownloading Latest Snapshot\n")
	}

	fmt.Printf("Download path: %s\n\n", downloader.OutputBasePath)

	go func() {
		for _, shard := range []int{0, 1, 2} {
			downloader.Download(shard, shardMetadata[shard])
		}
		progressChan <- downloader.ProgressUpdate{Quit: true}
	}()

	m := ui.NewDownloadModel(0, shardMetadata, progressChan, concurrentJobs)
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
