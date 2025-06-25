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
	MaxJobs       int
}

type cleanupMsg bool

var bold = lipgloss.NewStyle().Bold(true)

func NewDownloadModel(shard int, metadata map[int]*downloader.Metadata, progressChan <-chan downloader.ProgressUpdate, maxJobs int) DownloadModel {
	shards := []int{0, 1, 2}
	currentShard := shards[0]
	status := make(map[int]*ShardStatus, 3)
	for i := range shards {
		shrd := &ShardStatus{
			TotalChunks:      len(metadata[i].Chunks),
			DownloadedChunks: 0,
		}
		status[i] = shrd
	}
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
	return DownloadModel{
		CurrentShard:  currentShard,
		ShardMetadata: metadata,
		Status:        status,
		ActiveChunks:  make(map[string]Chunk),
		progressChan:  progressChan,
		Progress:      p,
		miniProgress:  p2,
		MaxJobs:       maxJobs,
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
		// Update chunk if it already exists, otherwise create new
		if existing, ok := m.ActiveChunks[chunkId]; ok {
			if msg.BytesDownloaded > existing.BytesDownloaded {
				existing.BytesDownloaded = msg.BytesDownloaded
				existing.BytesTotal = msg.BytesTotal
				m.ActiveChunks[chunkId] = existing
			}
		} else {
			m.ActiveChunks[chunkId] = Chunk{
				Shard:           msg.Shard,
				Name:            msg.ChunkName,
				BytesTotal:      msg.BytesTotal,
				BytesDownloaded: msg.BytesDownloaded,
			}
		}

		if msg.BytesDownloaded == msg.BytesTotal {
			status.DownloadedChunks++
			status.BytesDownloaded += msg.BytesDownloaded
			//delete(m.ActiveChunks, chunkId)
		}
	}
	return m, waitForUpdates(m.progressChan)
}

func (m DownloadModel) View() string {
	var s string

	// Collect and sort active chunk keys
	chunkKeys := make([]string, 0, len(m.ActiveChunks))
	for k := range m.ActiveChunks {
		chunkKeys = append(chunkKeys, k)
	}
	sort.Strings(chunkKeys)

	// Generate display for each shard
	for i := 0; i < 3; i++ {
		st := m.Status[i]
		s += bold.Render(fmt.Sprintf("Shard %02d ", i))
		var percent float64
		if st.TotalChunks > 0 {
			percent = float64(st.DownloadedChunks) / float64(st.TotalChunks)
		}
		bar := m.Progress.ViewAs(percent)
		s += fmt.Sprintf(" %04d/%04d chunks %s   %s", st.DownloadedChunks, st.TotalChunks, bar, bytesHuman(st.BytesDownloaded))

		// Gather details for active chunks in this shard
		details := ""
		for _, key := range chunkKeys {
			c := m.ActiveChunks[key]
			if c.Shard == i && c.BytesDownloaded < c.BytesTotal {
				// Avoid division by zero
				var chunkPercent float64
				if c.BytesTotal > 0 {
					chunkPercent = float64(c.BytesDownloaded) / float64(c.BytesTotal)
				}
				details += fmt.Sprintf("          • %s %s   %08d/%08d\n", c.Name, m.miniProgress.ViewAs(chunkPercent), c.BytesDownloaded, c.BytesTotal)

			}
		}
		if details != "" {
			s += "\n\n" + details
		}
		s += "\n"
	}

	// Show errors if any
	for _, e := range m.Errors {
		s += fmt.Sprintf(" [!!!] %v", e)
	}
	return s
}
