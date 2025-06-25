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
}
type DownloadModel struct {
	ShardQueue    []int
	CurrentShard  int
	ShardMetadata map[int]*downloader.Metadata
	Status        map[int]*ShardStatus
	progressChan  <-chan downloader.ProgressUpdate
	Progress      progress.Model
	miniProgress  progress.Model
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
		ShardQueue:    shards,
		CurrentShard:  currentShard,
		ShardMetadata: metadata,
		Status:        status,
		progressChan:  progressChan,
		Progress:      p,
		miniProgress:  p2,
	}
}

func (m DownloadModel) Init() tea.Cmd {
	return m.downloadShardCmd(m.CurrentShard, m.ShardMetadata[m.CurrentShard])
}

func waitForUpdates(ch <-chan downloader.ProgressUpdate) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func (m DownloadModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "q", "esc", "ctrl+c":
			return m, tea.Quit
		}

	case downloader.ProgressUpdate:
		status := m.Status[msg.Shard]
		if msg.ChunkName == "all" {
			if len(m.ShardQueue) > 1 {
				m.ShardQueue = m.ShardQueue[1:]
				m.CurrentShard = m.ShardQueue[0]
				go downloader.Download(m.CurrentShard, m.ShardMetadata[m.CurrentShard])
				return m, waitForUpdates(m.progressChan)
			}
			return m, tea.Batch(
				func() tea.Msg { return cleanupMsg(true) },
				waitForUpdates(m.progressChan),
			)
		}
		if msg.Done {
			if !status.Downloaded[msg.ChunkName] {
				status.Downloaded[msg.ChunkName] = true
				status.DownloadedChunks++
			}
			delete(status.ActiveChunks, msg.ChunkName)
		} else if msg.ChunkName != "all" {
			status.ActiveChunks[msg.ChunkName] = msg.Percent
			if status.Downloaded[msg.ChunkName] {
				// If it was already completed before being registered, clean it up now
				delete(status.ActiveChunks, msg.ChunkName)
			}
		}
		return m, waitForUpdates(m.progressChan)
	case cleanupMsg:
		for _, shard := range m.Status {
			for c := range shard.ActiveChunks {
				if shard.Downloaded[c] {
					delete(shard.ActiveChunks, c)
					shard.DownloadedChunks++
				}
			}
		}
		return m, tea.Batch(
			func() tea.Msg { return cleanupMsg(true) },
			waitForUpdates(m.progressChan),
		)
	}
	return m, waitForUpdates(m.progressChan)
}

func (m DownloadModel) downloadShardCmd(shard int, metadata *downloader.Metadata) tea.Cmd {
	go downloader.Download(shard, metadata)
	return waitForUpdates(m.progressChan)
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
			}
			s += "\n"
		} else {
			s += "\n"
		}
	}
	return s
}
