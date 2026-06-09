// Package kube wraps access to a Kubernetes cluster for fetching the objects
// that make up the resource graph.
package kube

import (
	"fmt"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

const (
	defaultFetchConcurrency = 6
	defaultClientQPS        = 30
	defaultClientBurst      = 60
)

// DefaultFetchConcurrency returns the default parallel list concurrency.
func DefaultFetchConcurrency() int { return defaultFetchConcurrency }

// DefaultClientQPS returns the default Kubernetes API client QPS.
func DefaultClientQPS() float32 { return defaultClientQPS }

// DefaultClientBurst returns the default Kubernetes API client burst size.
func DefaultClientBurst() int { return defaultClientBurst }

// Client provides typed and dynamic access to a Kubernetes cluster. The dynamic
// client is optional: when nil, only typed resources are fetched.
type Client struct {
	clientset        kubernetes.Interface
	dynamic          dynamic.Interface
	fetchConcurrency int
}

// NewClient creates a Client by building a REST configuration from the given
// options and constructing typed and dynamic clients.
func NewClient(opts Options) (*Client, error) {
	cfg, err := RESTConfig(opts)
	if err != nil {
		return nil, fmt.Errorf("build kube config: %w", err)
	}
	fetchConcurrency := resolveFetchConcurrency(opts.FetchConcurrency)
	applyRateLimits(cfg, opts)

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create kube clientset: %w", err)
	}

	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create dynamic client: %w", err)
	}

	return &Client{
		clientset:        clientset,
		dynamic:          dyn,
		fetchConcurrency: fetchConcurrency,
	}, nil
}

// NewClientForInterface wraps an existing typed clientset. It is primarily used
// to inject a fake clientset in tests; the dynamic client is left unset.
func NewClientForInterface(clientset kubernetes.Interface) *Client {
	return &Client{
		clientset:        clientset,
		fetchConcurrency: defaultFetchConcurrency,
	}
}

// NewClientForInterfaces wraps existing typed and dynamic clients. It is used to
// inject fakes in tests that exercise the dynamic fetch path.
func NewClientForInterfaces(clientset kubernetes.Interface, dyn dynamic.Interface) *Client {
	return &Client{
		clientset:        clientset,
		dynamic:          dyn,
		fetchConcurrency: defaultFetchConcurrency,
	}
}

// Clientset returns the underlying typed clientset.
func (c *Client) Clientset() kubernetes.Interface {
	return c.clientset
}

// Dynamic returns the underlying dynamic client, which may be nil.
func (c *Client) Dynamic() dynamic.Interface {
	return c.dynamic
}

func resolveFetchConcurrency(requested int) int {
	if requested > 0 {
		return requested
	}

	return defaultFetchConcurrency
}

func applyRateLimits(cfg *rest.Config, opts Options) {
	if opts.QPS > 0 {
		cfg.QPS = opts.QPS
	} else if cfg.QPS <= 0 {
		cfg.QPS = defaultClientQPS
	}

	if opts.Burst > 0 {
		cfg.Burst = opts.Burst
	} else if cfg.Burst <= 0 {
		cfg.Burst = defaultClientBurst
	}
}
