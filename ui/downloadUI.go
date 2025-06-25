package ui

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vrypan/snapsnapdown/downloader"
)

type ShardStatus struct {
	TotalChunks      int
	DownloadedChunks int
	ActiveChunks     map[string]float64
	Downloaded       map[string]bool
	BytesDownloaded  int64
	Done             bool
}
type DownloadModel struct {
	CurrentShard  int
	ShardMetadata map[int]*downloader.Metadata
	Status        map[int]*ShardStatus
	progressChan  <-chan downloader.ProgressUpdate
	Progress      progress.Model
	miniProgress  progress.Model
	Errors        []error
}

type cleanupMsg bool

var bold = lipgloss.NewStyle().Bold(true)

func NewDownloadModel(shard int, metadata map[int]*downloader.Metadata, progressChan <-chan downloader.ProgressUpdate) DownloadModel {
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

	shards := []int{0, 1, 2}
	currentShard := shards[0]
	status := make(map[int]*ShardStatus, 3)
	for i := range shards {
		shrd := ShardStatus{}
		shrd.ActiveChunks = make(map[string]float64)
		shrd.TotalChunks = len(metadata[i].Chunks)
		shrd.DownloadedChunks = 0
		shrd.Downloaded = make(map[string]bool)
		status[i] = &shrd
	}
	return DownloadModel{
		CurrentShard:  currentShard,
		ShardMetadata: metadata,
		Status:        status,
		progressChan:  progressChan,
		Progress:      p,
		miniProgress:  p2,
	}
}

func (m DownloadModel) Init() tea.Cmd {
	return waitForUpdates(m.progressChan)
}

func waitForUpdates(ch <-chan downloader.ProgressUpdate) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func (m DownloadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "q" || k == "esc" || k == "ctrl+c" {
			return m, tea.Quit
		}

	case downloader.ProgressUpdate:
		status := m.Status[msg.Shard]
		if msg.Quit {
			return m, tea.Quit
		}
		if msg.Error != nil {
			m.Errors = append(m.Errors, msg.Error)
			return m, waitForUpdates(m.progressChan)
		}
		if msg.Done {
			if !status.Downloaded[msg.ChunkName] {
				status.Downloaded[msg.ChunkName] = true
				status.DownloadedChunks++
			}
			delete(status.ActiveChunks, msg.ChunkName)
		} else {
			status.ActiveChunks[msg.ChunkName] = msg.Percent
		}
	}
	return m, waitForUpdates(m.progressChan)
}

func (m DownloadModel) View() string {
	s := ""
	for i := 0; i < 3; i++ {
		s += bold.Render(fmt.Sprintf("Shard %02d ", i))
		percent := 0.0
		if m.Status[i].TotalChunks > 0 {
			percent = float64(m.Status[i].DownloadedChunks) / float64(m.Status[i].TotalChunks)
		}
		bar := m.Progress.ViewAs(percent)
		s += fmt.Sprintf(" %04d/%04d chunks %s", m.Status[i].DownloadedChunks, m.Status[i].TotalChunks, bar)

		if len(m.Status[i].ActiveChunks) > 0 {
			s += "\n\n"
			keys := make([]string, 0, len(m.Status[i].ActiveChunks))
			for key := range m.Status[i].ActiveChunks {
				keys = append(keys, key)
			}
			sort.Strings(keys)
			for _, key := range keys {
				v := m.Status[i].ActiveChunks[key]
				s += fmt.Sprintf("          • %s %s\n", key, m.miniProgress.ViewAs(v))
				if v == 1.0 {
					// Clean up chunks that for some reason have not been deleted
					delete(m.Status[i].ActiveChunks, key)
				}
			}
			s += "\n"
		} else {
			s += "\n"
		}
	}
	for _, e := range m.Errors {
		s += fmt.Sprintf(" [!!!] %v", e)
	}
	return s
}
