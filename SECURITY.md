# Security Policy

## Supported Versions

Security fixes are provided for the latest release and the `main` branch.

| Version | Supported          |
| ------- | ------------------ |
| latest  | :white_check_mark: |
| main    | :white_check_mark: |
| older   | :x:                |

## Reporting a Vulnerability

**Please do not report security vulnerabilities through public GitHub issues.**

Preferred: use [GitHub private vulnerability reporting](https://github.com/gabor-boros/klue/security/advisories/new) on this repository.

If private reporting is unavailable, email **gabor.brs@gmail.com** with:

- A description of the vulnerability and its potential impact
- Steps to reproduce, including klue version and Kubernetes version if relevant
- Any proof-of-concept or exploit details you can share safely

You should receive an acknowledgment within **72 hours**. We will work with you to understand the issue and coordinate a fix and disclosure timeline based on severity.

## Out of Scope

The following are generally **not** considered security vulnerabilities in klue:

- Misconfigurations in your Kubernetes cluster or manifests
- Bugs in upstream Kubernetes or third-party operators/controllers
- Issues that require cluster-admin credentials the reporter already possesses
- Denial-of-service against a local CLI through extremely large API responses (report as a regular bug unless impact is severe)

Thank you for helping keep klue and its users safe.
