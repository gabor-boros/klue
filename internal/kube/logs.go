package kube

import (
	"bufio"
	"context"
	"strings"
	"sync"

	corev1 "k8s.io/api/core/v1"

	"github.com/gabor-boros/klue/internal/evidence"
)

// FetchPodLogs retrieves container logs for each candidate. Failures are
// recorded on the entry and do not abort the remaining fetches.
func (c *Client) FetchPodLogs(ctx context.Context, candidates []evidence.LogCandidate, tailLines int64) []evidence.LogEntry {
	if len(candidates) == 0 {
		return nil
	}

	entries := make([]evidence.LogEntry, len(candidates))
	var wg sync.WaitGroup
	sem := make(chan struct{}, c.fetchConcurrency)

	for i := range candidates {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				entries[idx] = evidence.LogEntry{
					PodRef:     candidates[idx].PodRef,
					Container:  candidates[idx].Container,
					Previous:   candidates[idx].Previous,
					FetchError: ctx.Err().Error(),
				}
				return
			}
			defer func() { <-sem }()

			entries[idx] = c.fetchPodLog(ctx, candidates[idx], tailLines)
		}(i)
	}

	wg.Wait()
	return entries
}

func (c *Client) fetchPodLog(ctx context.Context, candidate evidence.LogCandidate, tailLines int64) evidence.LogEntry {
	entry := evidence.LogEntry{
		PodRef:    candidate.PodRef,
		Container: candidate.Container,
		Previous:  candidate.Previous,
	}

	tail := tailLines
	req := c.clientset.CoreV1().Pods(candidate.Namespace).GetLogs(candidate.PodName, &corev1.PodLogOptions{
		Container: candidate.Container,
		Previous:  candidate.Previous,
		TailLines: &tail,
	})

	stream, err := req.Stream(ctx)
	if err != nil {
		entry.FetchError = err.Error()
		return entry
	}
	defer stream.Close()

	scanner := bufio.NewScanner(stream)
	const maxLineLength = 64 * 1024
	scanner.Buffer(make([]byte, 0, 64*1024), maxLineLength)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		entry.Lines = append(entry.Lines, line)
	}
	if err := scanner.Err(); err != nil {
		entry.FetchError = err.Error()
	}

	return entry
}

// JoinLogLines joins ranked log lines for evidence raw payloads.
func JoinLogLines(lines []string) string {
	return strings.Join(lines, "\n")
}
