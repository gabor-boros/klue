package kube

import (
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// Options configures how the Kubernetes connection is established.
type Options struct {
	// Kubeconfig is an explicit path to a kubeconfig file. When empty, the
	// default loading rules (including the KUBECONFIG environment variable and
	// the default path) are used.
	Kubeconfig string

	// Context selects a named context from the kubeconfig. When empty, the
	// current context is used.
	Context string

	// FetchConcurrency is the maximum number of Kubernetes list operations run
	// in parallel while building a diagnosis snapshot. Values <= 0 use the
	// package default.
	FetchConcurrency int

	// QPS is the client-go REST client rate limit in queries per second. Values
	// <= 0 use the package default.
	QPS float32

	// Burst is the client-go REST client burst size. Values <= 0 use the package
	// default.
	Burst int
}

// RESTConfig builds a Kubernetes REST configuration from the given options. It
// loads the configuration using the standard kubeconfig precedence and falls
// back to the in-cluster configuration when no kubeconfig is available.
func RESTConfig(opts Options) (*rest.Config, error) {
	loadingRules := clientcmd.NewDefaultClientConfigLoadingRules()
	if opts.Kubeconfig != "" {
		loadingRules.ExplicitPath = opts.Kubeconfig
	}

	overrides := &clientcmd.ConfigOverrides{
		CurrentContext: opts.Context,
	}

	clientConfig := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)

	cfg, err := clientConfig.ClientConfig()
	if err == nil {
		return cfg, nil
	}

	// When no usable kubeconfig is found, try the in-cluster configuration.
	if clientcmd.IsEmptyConfig(err) {
		if inCluster, inErr := rest.InClusterConfig(); inErr == nil {
			return inCluster, nil
		}
	}

	return nil, err
}
