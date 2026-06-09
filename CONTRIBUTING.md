# Contributing to klue

Thank you for your interest in contributing to klue. This guide covers the development workflow, commit conventions, and how to add new diagnostic rules.

## Code of Conduct

This project follows the [Contributor Covenant](CODE_OF_CONDUCT.md). By participating, you agree to uphold it.

## Getting Started

### Prerequisites

- [Go](https://go.dev) 1.26 or later
- [pre-commit](https://pre-commit.com) (optional but recommended)

### Setup

```bash
git clone https://github.com/gabor-boros/klue.git
cd klue
make tools        # install golangci-lint, goreleaser, and git-cliff
make pre-commit   # install and run git hooks
make build        # build ./bin/klue
```

Run the full local CI pipeline before opening a pull request:

```bash
make ci
```

## Pull Request Workflow

1. Fork the repository and create a feature branch from `main`.
2. Make your changes with tests where appropriate.
3. Run `make ci` and ensure it passes.
4. Open a pull request with a clear description of the problem and solution.
5. Link any related issues.

Small, focused pull requests are easier to review and merge.

## Commit Messages

This project uses [Conventional Commits](https://www.conventionalcommits.org/). Commit messages drive the changelog via [git-cliff](https://git-cliff.org/), so please follow the format:

```
<type>(<optional scope>): <description>
```

Common types:

| Type       | Purpose                                      |
| ---------- | -------------------------------------------- |
| `feat`     | New feature or diagnostic rule               |
| `fix`      | Bug fix                                      |
| `docs`     | Documentation only                           |
| `test`     | Tests only                                   |
| `chore`    | Maintenance (deps, tooling)                  |
| `refactor` | Code change that is not a fix or feature     |

Examples:

```
feat(pod): detect missing ConfigMap volume mounts
fix(service): correct selector mismatch evidence chain
docs: add cert-manager scenario to examples
```

Breaking changes use `!` after the type/scope: `feat(cli)!: rename why flags`.

Non-conventional messages (for example `wip`) are excluded from the changelog.

## Adding Diagnostic Rules

Diagnostic rules live under [`internal/rules/`](internal/rules/), grouped by Kubernetes resource kind. Each rule implements the `diagnose.Rule` interface and is registered in [`internal/rules/registry.go`](internal/rules/registry.go).

Typical steps:

1. Create or extend a file under the appropriate package (for example `internal/rules/pod/`).
2. Implement `Name()`, `Applies()`, and `Evaluate()` on your rule struct.
3. Reuse helpers from [`internal/rules/ruleutil/`](internal/rules/ruleutil/) where possible.
4. Register the rule in `registry.go` inside `All()`.
5. Add unit tests alongside existing `*_test.go` files in the same package.
6. Optionally add an example scenario under [`examples/scenarios/`](examples/scenarios/).

See existing rules such as [`internal/rules/pod/config_missing.go`](internal/rules/pod/config_missing.go) for patterns.

## Changelog

Before tagging a release, regenerate the changelog and commit the result:

```bash
make changelog
```

Release notes for GitHub are generated automatically from the same git-cliff configuration when a `v*` tag is pushed.

## Releasing

Releases are automated via GoReleaser and GitHub Actions. Maintainers tag `main` with a semantic version:

```bash
git tag v0.1.0
git push origin v0.1.0
```

Validate locally first:

```bash
make release-check
make snapshot
```

## Dependency updates

Dependency versions are managed by [Renovate](https://github.com/apps/renovate). Configuration lives in [`renovate.json`](renovate.json).

Maintainers: install the [Mend Renovate GitHub App](https://github.com/apps/renovate) on this repository if it is not already enabled. Renovate opens pull requests for Go modules, Python docs packages, GitHub Actions, pre-commit hooks, and pinned tooling in the Makefile and workflows.

Track pending updates in the **Dependency Dashboard** issue that Renovate creates. Merge dependency PRs after CI passes.

## Security

Do not open public issues for security vulnerabilities. See [SECURITY.md](SECURITY.md) for reporting instructions.

## Questions

Open a [feature request](https://github.com/gabor-boros/klue/issues/new?template=feature_request.yml) or [bug report](https://github.com/gabor-boros/klue/issues/new?template=bug_report.yml) issue if you are unsure where your change fits.
