package ui

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vrypan/snapsnapdown/downloader"
)

type Model struct {
	Shard            int
	TotalChunks      int
	DownloadedChunks int
	ActiveChunks     map[string]float64
	progressChan     <-chan downloader.ProgressUpdate
	Progress         progress.Model
	miniProgress     progress.Model
}

var bold = lipgloss.NewStyle().Bold(true)

func NewModel(shard int, totalChunks int, progressChan <-chan downloader.ProgressUpdate) Model {
	p := progress.New(progress.WithSolidFill("#00ff00"))
	p.Full = '■'
	p.Empty = ' '
	p.Width = 80
	p.ShowPercentage = true

	p2 := progress.New(progress.WithSolidFill("#999999"))
	p2.Full = '■'
	p2.Empty = ' '
	p2.Width = 80
	p2.ShowPercentage = true

	return Model{
		Shard:            shard,
		TotalChunks:      totalChunks,
		DownloadedChunks: 0,
		ActiveChunks:     make(map[string]float64),
		progressChan:     progressChan,
		Progress:         p,
		miniProgress:     p2,
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		waitForUpdates(m.progressChan),
	)
}

func waitForUpdates(ch <-chan downloader.ProgressUpdate) tea.Cmd {
	return func() tea.Msg {
		return downloader.ProgressUpdate(<-ch)
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}

	case downloader.ProgressUpdate:

		if msg.Percent <= 1.0 {
			m.ActiveChunks[msg.ChunkName] = msg.Percent
		} else {
			m.DownloadedChunks += 1
			delete(m.ActiveChunks, msg.ChunkName)
		}
		if m.DownloadedChunks == m.TotalChunks {
			return m, tea.Quit
		}
		return m, waitForUpdates(m.progressChan)
	}

	return m, nil
}

func (m Model) View() string {
	s := ""
	s += bold.Render(fmt.Sprintf("Shard %02d ", m.Shard))
	percent := float64(m.DownloadedChunks) / float64(m.TotalChunks)
	bar := m.Progress.ViewAs(percent)
	s += fmt.Sprintf(" %04d/%04d chunks %s", m.DownloadedChunks, m.TotalChunks, bar)

	if len(m.ActiveChunks) > 0 {
		s += "\n\n"
		keys := make([]string, 0, len(m.ActiveChunks))
		for key := range m.ActiveChunks {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		for _, key := range keys {
			v := m.ActiveChunks[key]

			//s += fmt.Sprintf("        :: %s  %s\n", key, m.miniProgress.ViewAs(v))
			s += fmt.Sprintf("          • %s %s\n", key, m.miniProgress.ViewAs(v))
		}
	} else {
		s += "\n"
	}
	return s
}
