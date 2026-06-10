# klue

Kubernetes troubleshooting that explains why, not just what.

## Documentation

Full documentation is published at [gabor-boros.github.io/klue](https://gabor-boros.github.io/klue/).

To build and preview locally: `make docs-serve`.

## Installation

```bash
go install github.com/gabor-boros/klue@latest
```

or

```bash
brew install --cask gabor-boros/brew/klue
```

Or download a prebuilt binary from the [releases page](https://github.com/gabor-boros/klue/releases).

## Usage

```bash
klue --help
klue version
klue why pod <name> -n <namespace>
klue why certificate <name> -n <namespace>
```

## Development

Requires [Go](https://go.dev) 1.26+. Run `make help` to list all available targets.

```bash
make build        # build the binary into ./bin
make run ARGS=... # run the CLI with arguments
make test         # run tests
make lint         # run golangci-lint
make fmt          # format the code
make ci           # tidy + vet + lint + race tests
make changelog    # regenerate CHANGELOG.md from git history
```

### Tooling

This project uses pinned versions of the following tools:

| Tool                                                          | Version  | Purpose                  |
| ------------------------------------------------------------- | -------- | ------------------------ |
| [GoReleaser](https://goreleaser.com)                          | v2.16.0  | Build and publish releases |
| [git-cliff](https://git-cliff.org)                            | v2.12.0  | Changelog generation     |
| [golangci-lint](https://golangci-lint.run)                    | v2.12.2  | Linting and formatting   |
| [pre-commit](https://pre-commit.com)                          | 4.6.0    | Git hook management      |

Install the Go-based tooling and set up git hooks:

```bash
make tools        # install golangci-lint, goreleaser, and git-cliff
make pre-commit   # install and run pre-commit hooks
```

### Releasing

Releases are automated via GoReleaser and GitHub Actions. Push a semantic
version tag to trigger a release:

```bash
git tag v0.1.0
git push origin v0.1.0
```

Validate the release configuration and build a local snapshot:

```bash
make release-check
make snapshot
```

## Community

- [Contributing](CONTRIBUTING.md)
- [Code of Conduct](CODE_OF_CONDUCT.md)
- [Security policy](SECURITY.md)
- [Changelog](CHANGELOG.md)

## License

klue is licensed under the [Apache License 2.0](LICENSE).
