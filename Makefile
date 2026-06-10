# klue - Makefile
#
# Run `make` or `make help` to list available targets.

# Use bash with strict flags for all recipes.
SHELL := bash
.SHELLFLAGS := -eu -o pipefail -c
.ONESHELL:
.DEFAULT_GOAL := help

# Pinned tooling versions (keep in sync with .github/workflows and configs).
GOLANGCI_LINT_VERSION ?= v2.12.2
GORELEASER_VERSION    ?= v2.16.0
GIT_CLIFF_VERSION     ?= 2.13.1
MKDOCS_MATERIAL_VERSION ?= 9.6.14

# Project metadata.
BINARY  := klue
PKG     := github.com/gabor-boros/klue
MODULE  := main

# Build directories.
DIST_DIR := dist
BIN_DIR  := bin

# Version metadata injected via -ldflags.
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
COMMIT  := $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
DATE    := $(shell date -u +%Y-%m-%dT%H:%M:%SZ)

LDFLAGS := -s -w \
	-X $(MODULE).version=$(VERSION) \
	-X $(MODULE).commit=$(COMMIT) \
	-X $(MODULE).date=$(DATE)

GO            ?= go
GOBIN         := $(shell $(GO) env GOPATH)/bin
GIT_CLIFF_BIN ?= $(shell command -v git-cliff 2>/dev/null || echo $(GOBIN)/git-cliff)

VENV_DOCS := .venv-docs
MKDOCS    := $(VENV_DOCS)/bin/mkdocs

.PHONY: help
help: ## Show this help.
	@grep -hE '^[a-zA-Z0-9_-]+:.*?## .*$$' $(MAKEFILE_LIST) \
		| sort \
		| awk 'BEGIN {FS = ":.*?## "}; {printf "  \033[36m%-16s\033[0m %s\n", $$1, $$2}'

## --- Build ---

.PHONY: build
build: ## Build the binary into ./bin.
	mkdir -p $(BIN_DIR)
	$(GO) build -trimpath -ldflags '$(LDFLAGS)' -o $(BIN_DIR)/$(BINARY) .

.PHONY: install
install: ## Install the binary into GOBIN.
	$(GO) install -trimpath -ldflags '$(LDFLAGS)' .

.PHONY: run
run: ## Run the CLI (use ARGS="..." to pass arguments).
	$(GO) run -ldflags '$(LDFLAGS)' . $(ARGS)

.PHONY: clean
clean: ## Remove build and test artifacts.
	rm -rf $(BIN_DIR) $(DIST_DIR) coverage.out coverage.html

## --- Quality ---

.PHONY: tidy
tidy: ## Tidy and verify go modules.
	$(GO) mod tidy
	$(GO) mod verify

.PHONY: fmt
fmt: ## Format code with golangci-lint formatters.
	golangci-lint fmt

.PHONY: vet
vet: ## Run go vet.
	$(GO) vet ./...

.PHONY: lint
lint: ## Run golangci-lint.
	golangci-lint run

## --- Tests ---

.PHONY: test
test: ## Run tests.
	$(GO) test ./...

.PHONY: test-race
test-race: ## Run tests with the race detector.
	$(GO) test -race ./...

.PHONY: cover
cover: ## Run tests and generate an HTML coverage report.
	$(GO) test -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report written to coverage.html"

## --- Release ---

.PHONY: snapshot
snapshot: ## Build a local snapshot release with GoReleaser.
	goreleaser release --snapshot --clean

.PHONY: release-check
release-check: ## Validate the GoReleaser configuration.
	goreleaser check

.PHONY: changelog
changelog: ## Generate CHANGELOG.md from git history.
	$(GIT_CLIFF_BIN) --config cliff.toml --output CHANGELOG.md

.PHONY: changelog-current
changelog-current: ## Print release notes for the latest tag.
	$(GIT_CLIFF_BIN) --config cliff.toml --current --strip header

## --- Tooling ---

.PHONY: tools
tools: ## Install pinned developer tooling (golangci-lint, goreleaser, git-cliff).
	$(GO) install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	$(GO) install github.com/goreleaser/goreleaser/v2@$(GORELEASER_VERSION)
	@set -eu; \
	if command -v git-cliff >/dev/null 2>&1 && git-cliff --version | grep -Eq " $(GIT_CLIFF_VERSION)([[:space:]]|$$)"; then \
		echo "git-cliff $(GIT_CLIFF_VERSION) already installed"; \
	else \
		os="$$(uname -s | tr '[:upper:]' '[:lower:]')"; \
		arch="$$(uname -m)"; \
		case "$$arch" in \
			x86_64) arch="x86_64" ;; \
			arm64|aarch64) arch="aarch64" ;; \
			*) echo "unsupported architecture: $$arch"; exit 1 ;; \
		esac; \
		case "$$os" in \
			linux) target="unknown-linux-gnu" ;; \
			darwin) target="apple-darwin" ;; \
			*) echo "unsupported os: $$os"; exit 1 ;; \
		esac; \
		archive="git-cliff-$(GIT_CLIFF_VERSION)-$$arch-$$target.tar.gz"; \
		url="https://github.com/orhun/git-cliff/releases/download/v$(GIT_CLIFF_VERSION)/$$archive"; \
		tmpdir="$$(mktemp -d)"; \
		curl -fsSL "$$url" -o "$$tmpdir/$$archive"; \
		tar -xzf "$$tmpdir/$$archive" -C "$$tmpdir"; \
		mkdir -p "$(GOBIN)"; \
		install "$$tmpdir/git-cliff-$(GIT_CLIFF_VERSION)/git-cliff" "$(GOBIN)/git-cliff"; \
		rm -rf "$$tmpdir"; \
	fi

.PHONY: pre-commit
pre-commit: ## Install and run pre-commit hooks against all files.
	pre-commit install
	pre-commit run --all-files

.PHONY: ci
ci: tidy vet lint test-race ## Run the full local CI pipeline.

## --- Documentation ---

.PHONY: docs-install docs-serve docs-build
docs-install: ## Install MkDocs Material into a local venv (.venv-docs).
	python3 -m venv $(VENV_DOCS)
	$(VENV_DOCS)/bin/pip install --upgrade pip
	$(VENV_DOCS)/bin/pip install -r docs/requirements-docs.txt

docs-serve: docs-install ## Serve docs locally at http://127.0.0.1:8000.
	$(MKDOCS) serve

docs-build: docs-install ## Build static site to ./site.
	$(MKDOCS) build --strict
