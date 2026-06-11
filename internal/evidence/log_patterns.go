package evidence

import (
	"regexp"
	"strings"
)

// LogSignal describes a failure pattern detected in container logs.
type LogSignal struct {
	ID              string
	Summary         string
	ConfidenceBoost float64
}

type logPattern struct {
	id      string
	summary string
	boost   float64
	re      *regexp.Regexp
}

var logPatterns = []logPattern{
	{id: "oom-killed", summary: "container may have been killed due to memory pressure (OOM)", boost: 0.15, re: regexp.MustCompile(`(?i)(out of memory|\boom\b|oomkilled|killed.*memory)`)},
	{id: "panic", summary: "application panic or runtime error in logs", boost: 0.2, re: regexp.MustCompile(`(?i)(panic:|runtime error:|fatal error:)`)},
	{id: "migration-failed", summary: "database or schema migration failed during startup", boost: 0.2, re: regexp.MustCompile(`(?i)(migration(s)? (failed|error)|failed to (apply|run) migration|schema migration failed)`)},
	{id: "connection-refused", summary: "connection refused to an upstream dependency", boost: 0.15, re: regexp.MustCompile(`(?i)connection refused`)},
	{id: "no-such-host", summary: "hostname or DNS resolution failure", boost: 0.15, re: regexp.MustCompile(`(?i)(no such host|enotfound|name or service not known)`)},
	{id: "timeout", summary: "request or dependency timeout while serving traffic", boost: 0.15, re: regexp.MustCompile(`(?i)(context deadline exceeded|i/o timeout|request timed out|timeout while|dial tcp.*i/o timeout)`)},
	{id: "tls-certificate", summary: "TLS certificate validation or handshake failure", boost: 0.15, re: regexp.MustCompile(`(?i)(x509:|certificate (has expired|is not yet valid)|unknown authority|tls: handshake failure|tls: failed to verify certificate)`)},
	{id: "permission-denied", summary: "permission denied or authorization failure", boost: 0.1, re: regexp.MustCompile(`(?i)(permission denied|\b403\b|\b401\b|unauthorized)`)},
	{id: "config-missing", summary: "missing configuration file or path", boost: 0.15, re: regexp.MustCompile(`(?i)(no such file or directory|config.*not found|failed to read.*config)`)},
	{id: "read-only-filesystem", summary: "application attempted to write to a read-only filesystem", boost: 0.15, re: regexp.MustCompile(`(?i)(read-only file system|cannot (create|write|open).*(read-only|readonly))`)},
	{id: "disk-full", summary: "disk pressure or storage quota exhaustion while writing data", boost: 0.15, re: regexp.MustCompile(`(?i)(no space left on device|disk quota exceeded|enospc)`)},
	{id: "database-connection", summary: "database connection or authentication failure", boost: 0.15, re: regexp.MustCompile(`(?i)(database is (unavailable|down)|password authentication failed|too many connections|could not connect to (server|database)|connection to .* database failed)`)},
	{id: "image-or-artifact-pull", summary: "application failed to pull an image or required artifact", boost: 0.1, re: regexp.MustCompile(`(?i)(failed to pull (artifact|image|module|package)|error pulling image|manifest unknown|failed to fetch (artifact|module|package))`)},
	{id: "probe-listen-failure", summary: "application startup indicates listener or health endpoint is not ready", boost: 0.1, re: regexp.MustCompile(`(?i)(server failed to start|failed to start.*(http|grpc|server)|health(check)? endpoint.*(failed|error)|readiness probe failed|liveness probe failed)`)},
	{id: "rate-limited", summary: "upstream dependency is rate limiting requests", boost: 0.1, re: regexp.MustCompile(`(?i)(\b429\b|too many requests|rate limit(?:ed)?|throttl(?:e|ing))`)},
	{id: "dependency-unavailable", summary: "upstream dependency is unavailable", boost: 0.1, re: regexp.MustCompile(`(?i)(\b503\b|service unavailable|upstream (service )?unavailable|bad gateway|gateway timeout)`)},
	{id: "bind-address", summary: "port bind failure or address already in use", boost: 0.1, re: regexp.MustCompile(`(?i)(address already in use|bind:.*address|failed to bind)`)},
}

var errorKeywords = regexp.MustCompile(`(?i)\b(error|fatal|exception|panic|failed)\b`)

// MatchLogLine returns the highest-priority pattern matched by line, if any.
func MatchLogLine(line string) (LogSignal, bool) {
	for _, pattern := range logPatterns {
		if pattern.re.MatchString(line) {
			return LogSignal{
				ID:              pattern.id,
				Summary:         pattern.summary,
				ConfidenceBoost: pattern.boost,
			}, true
		}
	}
	return LogSignal{}, false
}

// BestLogSignal scans lines and returns the strongest signal found.
func BestLogSignal(lines []string) (LogSignal, string, bool) {
	bestScore := -1
	var best LogSignal
	var bestLine string

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		score := lineScore(line, i)
		if score <= bestScore {
			continue
		}

		if signal, ok := MatchLogLine(line); ok {
			bestScore = score
			best = signal
			bestLine = line
			continue
		}

		if errorKeywords.MatchString(line) && bestScore < score {
			bestScore = score
			best = LogSignal{ID: "error-keyword", Summary: "error-level message in container logs", ConfidenceBoost: 0.05}
			bestLine = line
		}
	}

	if bestScore < 0 {
		return LogSignal{}, "", false
	}
	return best, bestLine, true
}

func lineScore(line string, index int) int {
	score := index + 1
	if signal, ok := MatchLogLine(line); ok {
		score += 1000 + int(signal.ConfidenceBoost*100)
	} else if errorKeywords.MatchString(line) {
		score += 100
	}
	return score
}
