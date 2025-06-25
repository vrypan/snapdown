package ui

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vrypan/snapsnapdown/downloader"
)

type Chunk struct {
	Shard           int
	Name            string
	BytesTotal      int64
	BytesDownloaded int64
}
type ShardStatus struct {
	TotalChunks      int
	DownloadedChunks int
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
	ActiveChunks  map[string]Chunk
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
		shrd.TotalChunks = len(metadata[i].Chunks)
		shrd.DownloadedChunks = 0
		status[i] = &shrd
	}
	return DownloadModel{
		CurrentShard:  currentShard,
		ShardMetadata: metadata,
		Status:        status,
		ActiveChunks:  make(map[string]Chunk),
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
		chunkId := fmt.Sprintf("%d-%s", msg.Shard, msg.ChunkName)
		m.ActiveChunks[chunkId] = Chunk{
			Shard:           msg.Shard,
			Name:            msg.ChunkName,
			BytesTotal:      msg.BytesTotal,
			BytesDownloaded: msg.BytesDownloaded,
		}

		if msg.BytesDownloaded == msg.BytesTotal {
			status.DownloadedChunks++
			status.BytesDownloaded += msg.BytesDownloaded
		}
	}
	return m, waitForUpdates(m.progressChan)
}

func (m DownloadModel) View() string {
	s := ""
	chunkKeys := make([]string, 0, len(m.ActiveChunks))
	for k := range m.ActiveChunks {
		chunkKeys = append(chunkKeys, k)
	}
	sort.Strings(chunkKeys)
	for i := 0; i < 3; i++ {
		s += bold.Render(fmt.Sprintf("Shard %02d ", i))
		percent := 0.0
		if m.Status[i].TotalChunks > 0 {
			percent = float64(m.Status[i].DownloadedChunks) / float64(m.Status[i].TotalChunks)
		}
		bar := m.Progress.ViewAs(percent)
		s += fmt.Sprintf(" %04d/%04d chunks %s   %s", m.Status[i].DownloadedChunks, m.Status[i].TotalChunks, bar, bytesHuman(m.Status[i].BytesDownloaded))
		details := ""
		for _, key := range chunkKeys {
			c := m.ActiveChunks[key]
			if c.Shard == i {
				percent := float32(c.BytesDownloaded) / float32(c.BytesTotal)
				details += fmt.Sprintf("          • %s %s   %08d/%08d\n", c.Name, m.miniProgress.ViewAs(float64(percent)), c.BytesDownloaded, c.BytesTotal)
				if c.BytesDownloaded == c.BytesTotal {
					delete(m.ActiveChunks, key)
				}

			}
		}
		if details != "" {
			s += "\n\n" + details
		}
		s += "\n"

	}
	for _, e := range m.Errors {
		s += fmt.Sprintf(" [!!!] %v", e)
	}
	return s
}
