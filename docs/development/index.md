---
description: Build, test, lint, and contribute to klue — Makefile targets, tooling, and community links.
tags:
  - development
---

# Development

Requires [Go](https://go.dev) 1.26+. Run `make help` to list all available
targets.

## Common tasks

```bash
make build        # build the binary into ./bin
make run ARGS=... # run the CLI with arguments
make test         # run tests
make lint         # run golangci-lint
make fmt          # format the code
make ci           # tidy + vet + lint + race tests
make changelog    # regenerate CHANGELOG.md from git history
```

## Local CI checklist

Before opening a pull request:

- [ ] `make ci` passes locally
- [ ] `make docs-build` passes (validates documentation)
- [ ] Pre-commit hooks installed (`make pre-commit`)

## Documentation

=== "Serve locally"

    Live reload at [http://127.0.0.1:8000](http://127.0.0.1:8000):

    ```bash
    make docs-serve
    ```

=== "Build static site"

    Output is written to `./site`:

    ```bash
    make docs-build
    ```

The docs site uses [MkDocs Material](https://squidfunk.github.io/mkdocs-material/)
with Mermaid diagrams, admonitions, and git-based revision dates.

## Tooling

This project uses pinned versions of the following tools:

| Tool | Version | Purpose |
|------|---------|---------|
| [GoReleaser](https://goreleaser.com) | v2.16.0 | Build and publish releases |
| [git-cliff](https://git-cliff.org) | v2.12.0 | Changelog generation |
| [golangci-lint](https://golangci-lint.run) | v2.12.2 | Linting and formatting |
| [pre-commit](https://pre-commit.com) | 4.6.0 | Git hook management |
| [MkDocs Material](https://squidfunk.github.io/mkdocs-material/) | 9.6.14 | Documentation site |

Install the Go-based tooling and set up git hooks:

```bash
make tools        # install golangci-lint, goreleaser, and git-cliff
make pre-commit   # install and run pre-commit hooks
```

## Community

- [Contributing](https://github.com/gabor-boros/klue/blob/main/CONTRIBUTING.md)
- [Code of Conduct](https://github.com/gabor-boros/klue/blob/main/CODE_OF_CONDUCT.md)
- [Security policy](https://github.com/gabor-boros/klue/blob/main/SECURITY.md)
- [Changelog](https://github.com/gabor-boros/klue/blob/main/CHANGELOG.md)

## Releasing

See [Releasing](releasing.md) for the tag-based release workflow.
