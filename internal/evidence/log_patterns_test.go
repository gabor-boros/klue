package evidence_test

import (
	"testing"

	"github.com/gabor-boros/klue/internal/evidence"
)

func TestMatchLogLine(t *testing.T) {
	t.Parallel()

	tests := []struct {
		line      string
		wantID    string
		wantMatch bool
	}{
		{line: "panic: runtime error: index out of range", wantID: "panic", wantMatch: true},
		{line: "startup failed: schema migration failed", wantID: "migration-failed", wantMatch: true},
		{line: "dial tcp 10.0.0.1:5432: connection refused", wantID: "connection-refused", wantMatch: true},
		{line: "lookup postgres.default.svc: no such host", wantID: "no-such-host", wantMatch: true},
		{line: "request failed: context deadline exceeded", wantID: "timeout", wantMatch: true},
		{line: "tls handshake failed: x509: certificate signed by unknown authority", wantID: "tls-certificate", wantMatch: true},
		{line: "permission denied while opening /var/run/secrets/token", wantID: "permission-denied", wantMatch: true},
		{line: "open /etc/config/app.yaml: no such file or directory", wantID: "config-missing", wantMatch: true},
		{line: "write /tmp/state: read-only file system", wantID: "read-only-filesystem", wantMatch: true},
		{line: "write /var/lib/app/chunk: no space left on device", wantID: "disk-full", wantMatch: true},
		{line: "FATAL: password authentication failed for user app", wantID: "database-connection", wantMatch: true},
		{line: "failed to pull artifact from registry.example.com/internal/service:v2", wantID: "image-or-artifact-pull", wantMatch: true},
		{line: "readiness probe failed: HTTP probe failed with statuscode: 503", wantID: "probe-listen-failure", wantMatch: true},
		{line: "api returned 429 too many requests", wantID: "rate-limited", wantMatch: true},
		{line: "upstream service unavailable (503)", wantID: "dependency-unavailable", wantMatch: true},
		{line: "listen tcp :8080: bind: address already in use", wantID: "bind-address", wantMatch: true},
		{line: "starting server", wantMatch: false},
	}

	for _, tt := range tests {
		signal, ok := evidence.MatchLogLine(tt.line)
		if ok != tt.wantMatch {
			t.Fatalf("MatchLogLine(%q) ok = %v, want %v", tt.line, ok, tt.wantMatch)
		}
		if tt.wantMatch && signal.ID != tt.wantID {
			t.Fatalf("MatchLogLine(%q) id = %q, want %q", tt.line, signal.ID, tt.wantID)
		}
	}
}

func TestMatchLogLinePrefersSpecificPatternOrder(t *testing.T) {
	t.Parallel()

	line := "request failed: 429 too many requests, upstream service unavailable"
	signal, ok := evidence.MatchLogLine(line)
	if !ok {
		t.Fatalf("MatchLogLine(%q) = no match, want match", line)
	}
	if signal.ID != "rate-limited" {
		t.Fatalf("signal.ID = %q, want rate-limited", signal.ID)
	}
}

func TestBestLogSignalPrefersPatternOverGenericError(t *testing.T) {
	t.Parallel()

	lines := []string{
		"info: starting",
		"error: something went wrong",
		"panic: boom",
	}

	signal, line, ok := evidence.BestLogSignal(lines)
	if !ok {
		t.Fatal("BestLogSignal() = false, want match")
	}
	if signal.ID != "panic" {
		t.Fatalf("signal.ID = %q, want panic", signal.ID)
	}
	if line != "panic: boom" {
		t.Fatalf("line = %q, want panic line", line)
	}
}
