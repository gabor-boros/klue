---
description: Reference for klue CLI commands — root, version, and why.
tags:
  - usage
  - reference
---

# Commands

klue is invoked as `klue <command> [flags]`.

Persistent (global) flags apply to all subcommands. See [Flags](flags.md) for
the full reference.

## Command summary

`klue`
:   Root command. Running `klue` without a subcommand prints help.

`klue version`
:   Print build metadata (version, commit, build date, Go version, OS/arch).

`klue why <resource> <name>`
:   Diagnose why a Kubernetes resource is unhealthy.

## `klue`

```bash
klue --help
```

## `klue version`

**Arguments:** none

=== "Example"

    ```bash
    klue version
    ```

=== "Sample output"

    ```
    klue v0.1.0
    commit:  abc1234
    built:   2026-06-10T12:00:00Z
    go:      go1.26.4
    os/arch: linux/amd64
    ```

## `klue why`

Explain why a Kubernetes resource is unhealthy.

**Arguments:** `<resource> <name>` (exactly two)

- `<resource>` — kind, plural resource name, or alias (for example `pod`,
  `pods`, `deploy`, `certificate`)
- `<name>` — object name in the target namespace (or cluster-wide for
  cluster-scoped kinds)

```bash
klue why pod web-7fdc4f4d74-jj6hb -n default
klue why certificate my-cert -n cert-manager
```

See [why](why.md) for resource token resolution, examples, and the diagnosis
pipeline. Command-specific flags are listed in [Flags](flags.md#why-flags).
