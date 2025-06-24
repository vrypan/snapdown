package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/vrypan/snapsnapdown/downloader"
)

type ExtractModel struct {
	MaxShard           int
	CurrentShard       int
	CurrentFile        string
	ShardChuncks       map[int]int
	ShardChunck        map[int]int
	ShardTotalBytesOut map[int]int64
	updatesCh          <-chan downloader.XUpdMsg
	progressBar        progress.Model
	spinner            spinner.Model
	error              error
}

func NewExtractModel(maxShard int, updates <-chan downloader.XUpdMsg) ExtractModel {
	p := progress.New(progress.WithSolidFill("#00ff00"))
	p.Full = '■'
	p.Empty = ' '
	p.Width = 80
	p.ShowPercentage = true
	spin := spinner.New()
	spin.Spinner = spinner.Dot
	return ExtractModel{
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

func (m ExtractModel) Init() tea.Cmd {
	return tea.Batch(
		m.spinner.Tick,      // Start the spinner updates
		listen(m.updatesCh), // Your existing update listener
	)
}

func (m ExtractModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	case downloader.XUpdMsg:
		if msg.Error != nil {
			m.error = msg.Error
			return m, tea.Quit
		}
		if msg.Quit {
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

		if msg.Error != nil {
			m.error = msg.Error
			return m, tea.Quit
		}

	case tea.QuitMsg:
		return m, tea.Quit
	}
	return m, listen(m.updatesCh)
}

func (m ExtractModel) View() string {
	s := ""
	for i := 0; i <= m.MaxShard; i++ {
		s += bold.Render(fmt.Sprintf("Shard %02d ", i))
		percent := 0.0
		if m.ShardChuncks[i] > 0 {
			percent = float64(m.ShardChunck[i]) / float64(m.ShardChuncks[i])
		}

		bar := m.progressBar.ViewAs(percent)
		s += fmt.Sprintf(" %04d/%04d chunks %s   [ %s ]\n", m.ShardChunck[i], m.ShardChuncks[i], bar, bytesHuman(m.ShardTotalBytesOut[i]))
	}
	s += fmt.Sprintf("\n          %sExtracting %s\n\n", m.spinner.View(), m.CurrentFile)

	if m.error != nil {
		s += fmt.Sprintf("\n\n%v\n\n", m.error)
	}
	return s
}

func bytesHuman(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
