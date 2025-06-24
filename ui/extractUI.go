package ui

import (
	"fmt"

	"github.com/charmbracelet/bubbles/progress"
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
	error              error
}

func NewExtractModel(maxShard int, updates <-chan downloader.XUpdMsg) ExtractModel {
	p := progress.New(progress.WithSolidFill("#00ff00"))
	p.Full = '■'
	p.Empty = ' '
	p.Width = 80
	p.ShowPercentage = true
	return ExtractModel{
		MaxShard:           maxShard,
		CurrentShard:       0,
		updatesCh:          updates,
		ShardChuncks:       make(map[int]int, maxShard+1),
		ShardChunck:        make(map[int]int, maxShard+1),
		ShardTotalBytesOut: make(map[int]int64, maxShard+1),
		progressBar:        p,
	}
}

func listen(ch <-chan downloader.XUpdMsg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func (m ExtractModel) Init() tea.Cmd {
	return listen(m.updatesCh)
}

func (m ExtractModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}
	case downloader.XUpdMsg:
		m.CurrentShard = msg.Shard
		m.CurrentFile = msg.File
		m.ShardChuncks[msg.Shard] = msg.Total
		m.ShardChunck[msg.Shard] = msg.Idx
		m.ShardTotalBytesOut[msg.Shard] = msg.TotalBytes
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
	s += "\nx " + m.CurrentFile

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
