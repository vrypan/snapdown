package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

// Messages
type StartChunkMsg struct {
	Shard int
	Chunk string
}

type FileExtractedMsg struct {
	Shard int
	File  string
}

type ShardCompletedMsg struct {
	Shard int
}

// Shard progress tracker
type ShardProgress struct {
	TotalChunks     int
	ExtractedChunks int
	CurrentChunk    string
	FilesExtracted  []string
	Completed       bool
}

type ExtractModel struct {
	Shards     map[int]*ShardProgress
	ShardOrder []int
	progressCh <-chan tea.Msg
}

func NewExtractModel(progressCh <-chan tea.Msg) ExtractModel {
	return ExtractModel{
		Shards:     make(map[int]*ShardProgress),
		ShardOrder: []int{},
		progressCh: progressCh,
	}
}

func listenProgress(ch <-chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		return <-ch
	}
}

func (m ExtractModel) Init() tea.Cmd {
	return listenProgress(m.progressCh)
}

func (m ExtractModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case StartChunkMsg:
		shard, exists := m.Shards[msg.Shard]
		if !exists {
			m.Shards[msg.Shard] = &ShardProgress{TotalChunks: 10} // Set total chunks here
			m.ShardOrder = append(m.ShardOrder, msg.Shard)
			shard = m.Shards[msg.Shard]
		}
		shard.CurrentChunk = msg.Chunk
		shard.ExtractedChunks++

	case FileExtractedMsg:
		shard := m.Shards[msg.Shard]
		shard.FilesExtracted = append(shard.FilesExtracted, msg.File)

	case ShardCompletedMsg:
		shard := m.Shards[msg.Shard]
		shard.Completed = true
		shard.CurrentChunk = ""

	case tea.QuitMsg:
		return m, tea.Quit
	}

	return m, listenProgress(m.progressCh)
}

func (m ExtractModel) View() string {
	var b strings.Builder
	for _, shardID := range m.ShardOrder {
		shard := m.Shards[shardID]
		progress := float64(shard.ExtractedChunks) / float64(shard.TotalChunks)
		b.WriteString(fmt.Sprintf("Shard %02d: %d/%d chunks extracted %s\n", shardID, shard.ExtractedChunks, shard.TotalChunks, renderProgressBar(progress)))
		if shard.CurrentChunk != "" {
			b.WriteString(fmt.Sprintf("  Extracting: %s\n", shard.CurrentChunk))
		}
		if len(shard.FilesExtracted) > 0 {
			b.WriteString("  Recent files:\n")
			files := shard.FilesExtracted
			if len(files) > 5 {
				files = files[len(files)-5:]
			}
			for _, file := range files {
				b.WriteString(fmt.Sprintf("    - %s\n", file))
			}
		}
		if shard.Completed {
			b.WriteString("  ✅ Shard Completed\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("Press ctrl+c to quit.\n")
	return b.String()
}

func renderProgressBar(p float64) string {
	width := 30
	filled := int(p * float64(width))
	if filled > width {
		filled = width
	}
	return "[" + strings.Repeat("■", filled) + strings.Repeat(" ", width-filled) + "]"
}
