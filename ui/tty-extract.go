package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vrypan/snapdown/downloader"
)

type TtyExtract struct {
	MaxShard           int
	CurrentShard       int
	CurrentFile        string
	ShardChuncks       map[int]int
	ShardChunck        map[int]int
	ShardTotalBytesOut map[int]int64
	updatesCh          <-chan downloader.XUpdMsg
	progressBar        progress.Model
	spinner            spinner.Model
	Errors             []error
}

func NewTtyExtract(maxShard int, updates <-chan downloader.XUpdMsg) TtyExtract {
	p := progress.New(progress.WithSolidFill("#00ff00"))
	p.Full = '■'
	p.Empty = ' '
	p.Width = 80
	p.ShowPercentage = true
	spin := spinner.New()
	spin.Spinner = spinner.Dot
	return TtyExtract{
		MaxShard:           maxShard,
		CurrentShard:       0,
		updatesCh:          updates,
		ShardChuncks:       make(map[int]int, maxShard+1),
		ShardChunck:        make(map[int]int, maxShard+1),
		ShardTotalBytesOut: make(map[int]int64, maxShard+1),
		progressBar:        p,
		spinner:            spin,
	}
}

func listen(ch <-chan downloader.XUpdMsg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func (m TtyExtract) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,      // Start the spinner updates
		listen(m.updatesCh), // Your existing update listener
	)
}

func (m TtyExtract) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" {
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case downloader.XUpdMsg:
		if msg.Error != nil {
			m.Errors = append(m.Errors, msg.Error)
			//return m, tea.Quit
		}
		if msg.Quit {
			m.CurrentFile = ""
			return m, tea.Quit
		}
		if msg.TotalBytes > 0 {
			m.ShardTotalBytesOut[msg.Shard] = msg.TotalBytes
			m.CurrentFile = msg.File
		} else {
			m.CurrentShard = msg.Shard
			m.ShardChuncks[msg.Shard] = msg.Total
			m.ShardChunck[msg.Shard] = msg.Idx
		}

	case tea.QuitMsg:
		return m, tea.Quit
	}
	return m, listen(m.updatesCh)
}

func (m TtyExtract) View() string {
	var s string
	s = fmt.Sprintf("Shard  Chunks %-80s      Bytes Out\n", "")

	for i := 0; i <= m.MaxShard; i++ {
		s += bold.Render(fmt.Sprintf("%02d ", i))
		totalChunks := m.ShardChuncks[i]
		currentChunk := m.ShardChunck[i]
		totalBytes := m.ShardTotalBytesOut[i]

		var percent float64
		if totalChunks > 0 {
			percent = float64(currentChunk) / float64(totalChunks)
		} else {
			percent = 0.0
		}

		bar := m.progressBar.ViewAs(percent)
		s += fmt.Sprintf("    %04d/%04d %s   %s\n", currentChunk, totalChunks, bar, bytesHuman(totalBytes))
	}
	if m.CurrentFile != "" {
		s += fmt.Sprintf("\n%sExtracting %s\n\n", m.spinner.View(), m.CurrentFile)
	}
	if len(m.Errors) > 0 {
		s += "\n"
		for _, e := range m.Errors {
			s += fmt.Sprintf("[!] %v\n", e)
		}
	}
	return s
}
