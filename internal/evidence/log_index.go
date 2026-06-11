package evidence

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gabor-boros/klue/pkg/resource"
)

// RankedLine is a log line with a relevance score for reporting.
type RankedLine struct {
	Text      string
	Score     int
	PatternID string
}

// LogSet holds log lines for one pod container.
type LogSet struct {
	entries []LogEntry
}

// Len returns the number of stored log entries.
func (s *LogSet) Len() int {
	if s == nil {
		return 0
	}
	return len(s.entries)
}

// Lines returns all log lines across entries in deterministic order.
func (s *LogSet) Lines() []string {
	if s == nil {
		return nil
	}

	var lines []string
	for _, entry := range s.entries {
		lines = append(lines, entry.Lines...)
	}
	return lines
}

// RelevantLines returns the top-ranked lines for display.
func (s *LogSet) RelevantLines(max int) []RankedLine {
	if s == nil || max <= 0 {
		return nil
	}

	lines := s.Lines()
	if len(lines) == 0 {
		return nil
	}

	ranked := rankLines(lines)
	if len(ranked) > max {
		ranked = ranked[:max]
	}
	return ranked
}

// BestSignal returns the strongest log signal in the set.
func (s *LogSet) BestSignal() (LogSignal, string, bool) {
	return BestLogSignal(s.Lines())
}

// SummaryMessage builds a one-line summary for evidence output.
func (s *LogSet) SummaryMessage(container string, previous bool) string {
	if s == nil {
		return ""
	}

	run := "current"
	if previous {
		run = "previous"
	}

	signal, line, ok := s.BestSignal()
	if ok {
		return fmt.Sprintf(`container %q (%s): %s — %s`, container, run, signal.Summary, truncateLine(line, 120))
	}

	lines := s.RelevantLines(1)
	if len(lines) > 0 {
		return fmt.Sprintf(`container %q (%s): %s`, container, run, truncateLine(lines[0].Text, 120))
	}

	for _, entry := range s.entries {
		if entry.FetchError != "" {
			return fmt.Sprintf(`container %q (%s): could not fetch logs: %s`, container, run, entry.FetchError)
		}
	}

	return ""
}

// RawExcerpt returns the top ranked lines joined for evidence raw payloads.
func (s *LogSet) RawExcerpt(maxLines int) string {
	ranked := s.RelevantLines(maxLines)
	if len(ranked) == 0 {
		return ""
	}

	lines := make([]string, len(ranked))
	for i, line := range ranked {
		lines[i] = truncateLine(line.Text, 200)
	}
	return strings.Join(lines, "\n")
}

// LogIndex groups fetched logs by pod and container.
type LogIndex struct {
	byKey map[string]*LogSet
}

// NewLogIndex builds a LogIndex from fetched log entries.
func NewLogIndex(entries []LogEntry) *LogIndex {
	index := &LogIndex{byKey: make(map[string]*LogSet)}
	for _, entry := range entries {
		key := logKey(entry.PodRef, entry.Container)
		set, ok := index.byKey[key]
		if !ok {
			set = &LogSet{}
			index.byKey[key] = set
		}
		set.entries = append(set.entries, entry)
	}
	return index
}

// ForPodContainer returns logs for the given pod container. The returned set is
// never nil.
func (i *LogIndex) ForPodContainer(pod resource.Reference, container string) *LogSet {
	if i == nil {
		return &LogSet{}
	}
	if set, ok := i.byKey[logKey(pod, container)]; ok {
		return set
	}
	return &LogSet{}
}

func logKey(pod resource.Reference, container string) string {
	return pod.Key() + "|" + container
}

func rankLines(lines []string) []RankedLine {
	ranked := make([]RankedLine, 0, len(lines))
	for idx, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		item := RankedLine{Text: line, Score: lineScore(line, idx)}
		if signal, ok := MatchLogLine(line); ok {
			item.PatternID = signal.ID
		}
		ranked = append(ranked, item)
	}

	sort.SliceStable(ranked, func(a, b int) bool {
		if ranked[a].Score != ranked[b].Score {
			return ranked[a].Score > ranked[b].Score
		}
		return ranked[a].Text < ranked[b].Text
	})

	return ranked
}

func truncateLine(line string, max int) string {
	if max <= 0 || len(line) <= max {
		return line
	}
	return line[:max-3] + "..."
}
