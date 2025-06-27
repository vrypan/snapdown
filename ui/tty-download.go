package ui

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/vrypan/snapdown/downloader"
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
type TtyDownload struct {
	CurrentShard      int
	ShardMetadata     map[int]*downloader.Metadata
	Status            map[int]*ShardStatus
	progressChan      <-chan downloader.ProgressUpdate
	Progress          progress.Model
	miniProgress      progress.Model
	Errors            []error
	ActiveChunks      map[string]Chunk
	MaxJobs           int
	RecentlyCompleted map[string]time.Time
}

type cleanupMsg bool

var bold = lipgloss.NewStyle().Bold(true)

func NewTtyDownload(shard int, metadata map[int]*downloader.Metadata, progressChan <-chan downloader.ProgressUpdate, maxJobs int) TtyDownload {
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
	return TtyDownload{
		CurrentShard:      currentShard,
		ShardMetadata:     metadata,
		Status:            status,
		ActiveChunks:      make(map[string]Chunk, maxJobs*2),
		RecentlyCompleted: make(map[string]time.Time, maxJobs*4),
		progressChan:      progressChan,
		Progress:          p,
		miniProgress:      p2,
		MaxJobs:           maxJobs,
	}
}

func (m TtyDownload) Init() tea.Cmd {
	return tea.Batch(
		waitForUpdates(m.progressChan),
		tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return cleanupMsg(true)
		}),
	)
}

func waitForUpdates(ch <-chan downloader.ProgressUpdate) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func (m TtyDownload) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if k := msg.String(); k == "ctrl+c" {
			return m, tea.Quit
		}
	case cleanupMsg:
		now := time.Now()
		for chunkId, completedAt := range m.RecentlyCompleted {
			if now.Sub(completedAt) > 5*time.Second {
				delete(m.RecentlyCompleted, chunkId)
			}
		}
		return m, tea.Tick(time.Second, func(t time.Time) tea.Msg {
			return cleanupMsg(true)
		})
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
		if _, ok := m.RecentlyCompleted[chunkId]; ok {
			// Already recently completed, safe to ignore further updates
			return m, waitForUpdates(m.progressChan)
		}
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

			// Move to recently completed
			m.RecentlyCompleted[chunkId] = time.Now()
			delete(m.ActiveChunks, chunkId)
		}
	}
	return m, waitForUpdates(m.progressChan)
}

func (m TtyDownload) View() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Shard  Chunks %-80s      Bytes In\n", ""))

	// Collect and sort active chunk keys
	chunkKeys := make([]string, 0, len(m.ActiveChunks))
	for k := range m.ActiveChunks {
		chunkKeys = append(chunkKeys, k)
	}
	sort.Strings(chunkKeys)

	// Generate display for each shard
	for i := 0; i < 3; i++ {
		st := m.Status[i]
		b.WriteString(bold.Render(fmt.Sprintf("%02d ", i)))
		var percent float64
		if st.TotalChunks > 0 {
			percent = float64(st.DownloadedChunks) / float64(st.TotalChunks)
		}
		bar := m.Progress.ViewAs(percent)
		b.WriteString(fmt.Sprintf("    %04d/%04d %s   %s", st.DownloadedChunks, st.TotalChunks, bar, bytesHuman(st.BytesDownloaded)))

		// Gather details for active chunks in this shard
		var details strings.Builder
		for _, key := range chunkKeys {
			c := m.ActiveChunks[key]
			if c.Shard == i && c.BytesDownloaded < c.BytesTotal {
				// Avoid division by zero
				var chunkPercent float64
				if c.BytesTotal > 0 {
					chunkPercent = float64(c.BytesDownloaded) / float64(c.BytesTotal)
				}
				details.WriteString(fmt.Sprintf("> %s %s   %08d/%08d\n", c.Name, m.miniProgress.ViewAs(chunkPercent), c.BytesDownloaded, c.BytesTotal))
			}
		}
		detailStr := details.String()
		if detailStr != "" {
			b.WriteString("\n\n")
			b.WriteString(detailStr)
			b.WriteString("\n")
		} else {
			b.WriteString("\n")
		}
	}

	for _, e := range m.Errors {
		b.WriteString(fmt.Sprintf("[!] %v\n", e))
	}

	return b.String()
}

func (m TtyDownload) debugInfo() string {
	var mm runtime.MemStats
	var s string
	runtime.ReadMemStats(&mm)
	s += fmt.Sprintf("\nMemory: Alloc = %v MiB | TotalAlloc = %v MiB | Sys = %v MiB | NumGC = %v\n",
		mm.Alloc/1024/1024, mm.TotalAlloc/1024/1024, mm.Sys/1024/1024, mm.NumGC)
	s += fmt.Sprintf("len(RecentlyCompleted)=%d | len(ActiveChunks)=%d", len(m.RecentlyCompleted), len(m.ActiveChunks))
	return s
}
