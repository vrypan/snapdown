package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/vrypan/snapsnapdown/downloader"
	"github.com/vrypan/snapsnapdown/ui"
)

var shards []int

var extractCmd = &cobra.Command{
	Use:     "extract <source dir> <destination dir>",
	Aliases: []string{"x"},
	Short:   "Extract downloaded snapshot",
	Long: `
If you downloaded the snapshot in ./snapshot you will probably
want to run:
  snapsnapdown extract ./snapshot .rocks
to extract the files in .rocks. Then you can start your node.

WARNING! Files in <destination dir> will be overwritten!
	`,
	//Run:     extractRun,
	Run: extractRun,
}

func init() {
	rootCmd.AddCommand(extractCmd)
	extractCmd.Flags().IntSliceVar(&shards, "shards", []int{0, 1, 2}, "List of shard indices (e.g. --shard=0,1,2)")
}

func extractRun(cmd *cobra.Command, args []string) {
	if len(args) < 2 {
		cmd.Help()
		os.Exit(1)
	}
	if !downloader.HasTarInPath() {
		fmt.Println("Error: 'tar' not found in PATH.")
		os.Exit(1)
	}
	srcDir := args[0]
	dstDir := args[1]
	progressCh := make(chan downloader.XUpdMsg, 1000)

	fmt.Printf("\nExtracting Snapshot")
	fmt.Printf(" [%s] -> [%s]\n\n", srcDir, dstDir)

	go func() {
		for _, shard := range shards {
			downloader.ExtractWithNativeTar(srcDir, dstDir, shard, progressCh)
		}
		progressCh <- downloader.XUpdMsg{Quit: true}
	}()

	p := tea.NewProgram(
		ui.NewExtractModel(
			2, progressCh,
		),
	)
	if _, err := p.Run(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
