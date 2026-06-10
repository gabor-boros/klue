---
description: Install klue with go install or download a prebuilt binary from GitHub Releases.
tags:
  - getting-started
  - install
---

# Installation

=== "Go install"

    Requires [Go](https://go.dev) 1.26 or later.

    ```bash
    go install github.com/gabor-boros/klue@latest
    ```

    The binary is installed to your `GOBIN` directory (typically
    `$(go env GOPATH)/bin`). Make sure that directory is on your `PATH`.

=== "Homebrew"

    Requires [brew](https://brew.sh) installed.

    ```bash
    brew install --cask gabor-boros/brew/klue
    ```

=== "Prebuilt binary"

    Download a prebuilt binary for your platform from the
    [releases page](https://github.com/gabor-boros/klue/releases).

    Extract the archive and place the `klue` binary somewhere on your `PATH`.

## Verify the installation

```bash
klue version
```

You should see the release version, commit, build date, and platform details.

!!! warning "Command not found?"
    If your shell cannot find `klue`, add `$(go env GOPATH)/bin` to your `PATH`
    or move the binary to a directory that is already listed.

## Cluster access

klue needs read access to your Kubernetes API server. See
[Kubernetes access](kubernetes-access.md) for how kubeconfig, context, and
in-cluster credentials are resolved.

## What's next?

- [ ] Install and verify `klue version` works
- [ ] Confirm `kubectl` can reach your cluster (same credentials klue will use)
- [ ] Run your first diagnosis: [why](../usage/why.md)
