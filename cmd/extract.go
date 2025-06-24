package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"
	"github.com/vrypan/snapsnapdown/downloader"
	"github.com/vrypan/snapsnapdown/ui"
)

var extractCmd = &cobra.Command{
	Use:     "extract <source dir> <destination dir>",
	Aliases: []string{"x"},
	Short:   "Extract downloaded snapshot",
	Long: `If you downloaded the snapshot in ./snapshot you will probably want to run
  snapsnapdown extract ./snapshot .rocks
to extract the files in .rocks. Then you can start your node.

WARNING! Files in <destination dir> will be overwritten!
	`,
	//Run:     extractRun,
	Run: extractRun,
}

func init() {
	rootCmd.AddCommand(extractCmd)
}

func extractRun(cmd *cobra.Command, args []string) {
	srcDir := args[0]
	dstDir := args[1]
	progressCh := make(chan downloader.XUpdMsg, 1000)

	go func() {
		for i := 0; i < 3; i++ {
			downloader.Extract(srcDir, dstDir, i, progressCh)
		}
	}()

	p := tea.NewProgram(
		ui.NewExtractModel(
			2, progressCh,
		),
	)
	if err := p.Start(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
