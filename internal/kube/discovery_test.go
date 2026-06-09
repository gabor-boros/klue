package kube_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gabor-boros/klue/internal/kube"
)

const testKubeconfig = `apiVersion: v1
kind: Config
clusters:
- name: primary
  cluster:
    server: https://primary.example.com:6443
- name: secondary
  cluster:
    server: https://secondary.example.com:6443
contexts:
- name: primary
  context:
    cluster: primary
    user: tester
- name: secondary
  context:
    cluster: secondary
    user: tester
current-context: primary
users:
- name: tester
  user:
    token: test-token
`

func writeKubeconfig(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "kubeconfig")
	if err := os.WriteFile(path, []byte(testKubeconfig), 0o600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}

	return path
}

func TestRESTConfig_ExplicitKubeconfig(t *testing.T) {
	t.Parallel()

	cfg, err := kube.RESTConfig(kube.Options{Kubeconfig: writeKubeconfig(t)})
	if err != nil {
		t.Fatalf("RESTConfig() error = %v", err)
	}

	if cfg.Host != "https://primary.example.com:6443" {
		t.Errorf("Host = %q, want primary server", cfg.Host)
	}
}

func TestRESTConfig_ContextOverride(t *testing.T) {
	t.Parallel()

	cfg, err := kube.RESTConfig(kube.Options{Kubeconfig: writeKubeconfig(t), Context: "secondary"})
	if err != nil {
		t.Fatalf("RESTConfig() error = %v", err)
	}

	if cfg.Host != "https://secondary.example.com:6443" {
		t.Errorf("Host = %q, want secondary server", cfg.Host)
	}
}
