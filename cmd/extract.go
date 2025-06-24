package cmd

import (
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/vrypan/snapsnapdown/downloader"
	"github.com/vrypan/snapsnapdown/ui"
)

var extractCmd = &cobra.Command{
	Use:     "extract",
	Aliases: []string{"x"},
	Short:   "EXPERIMENTAL: Extract a downloaded snapshot",
	//Run:     extractRun,
	Run: func(cmd *cobra.Command, args []string) {
		srcDir := args[0]
		dstDir := args[1]
		downloader.Extract(srcDir, dstDir)
	},
}

func init() {
	rootCmd.AddCommand(extractCmd)
}

func extractRun(cmd *cobra.Command, args []string) {
	progressCh := make(chan tea.Msg)

	go simulateExtraction(progressCh)

	p := tea.NewProgram(ui.NewExtractModel(progressCh))
	if err := p.Start(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}

func simulateExtraction(progressCh chan tea.Msg) {
	shards := []int{0, 1, 2}
	for _, shard := range shards {
		for chunk := 0; chunk < 10; chunk++ {
			chunkName := fmt.Sprintf("chunk_%04d.bin", chunk)
			progressCh <- ui.StartChunkMsg{Shard: shard, Chunk: chunkName}
			time.Sleep(200 * time.Millisecond)
			progressCh <- ui.FileExtractedMsg{Shard: shard, File: fmt.Sprintf("file_%d.txt", chunk)}
		}
		progressCh <- ui.ShardCompletedMsg{Shard: shard}
	}
	close(progressCh)
}
