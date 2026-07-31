<!--
SPDX-FileCopyrightText: 2026 Grove contributors

SPDX-License-Identifier: Apache-2.0
-->

# Contributing to Grove

Thanks for your interest in contributing. This document covers the
mechanics of getting a change merged; see the README for what Grove is and
how to run it locally.

## Prerequisites

- Go 1.26.5
- [go-task](https://taskfile.dev)
- Docker (only needed for `task docker:*`)

## Workflow

1. Fork the repo and create a branch off `main`.
2. Make your change. Keep commits focused — one logical change per commit.
3. Run `task check` before opening a PR; it runs the same fmt/vet/lint/test
   steps as CI.
4. Open a pull request against `main`.

PRs are merged via squash merge, so the **PR title becomes the permanent
commit message** on `main` — individual commit messages don't survive the
merge, but should still follow the same conventions below for reviewability.

## Commit messages

Commit messages and PR titles follow [Conventional Commits](https://www.conventionalcommits.org/):

```
<type>: <short summary>

[optional body]
```

Common types: `feat`, `fix`, `docs`, `test`, `refactor`, `chore`, `ci`, `build`.

## Developer Certificate of Origin (DCO)

Every commit must be signed off, certifying you wrote it or otherwise have
the right to submit it under the project's license (see the
[DCO text](https://developercertificate.org/)):

```sh
git commit -s -m "feat: add thing"
```

This adds a `Signed-off-by: Your Name <you@example.com>` trailer using your
configured Git identity (`git config user.name` / `user.email`).

## Signed commits

`main` requires cryptographically signed commits (GPG or SSH) — GitHub must
show them as "Verified". See GitHub's guide to
[signing commits](https://docs.github.com/en/authentication/managing-commit-signature-verification/signing-commits)
to set up a GPG or SSH signing key, then either pass `-S` per commit or set
`git config commit.gpgsign true` (or `commit.ssh.signingKey` /
`commit.gpgsign` with `gpg.format ssh`) to sign by default.

## What CI checks

Every PR is automatically checked for:

- `go build`, `go vet`, `go test -race`, `golangci-lint run`, and a Docker
  build
- DCO sign-off on every commit (the [DCO app](https://probot.github.io/apps/dco/))
- Every commit is signed (GPG/SSH) — enforced by branch protection
- The PR title and each commit message follow Conventional Commits

Run `task check` locally to catch the build/lint/test issues before pushing;
the rest (signing, DCO, message format) you get right by following the
sections above.

## Code style

- `gofmt`-formatted, `go vet` and `golangci-lint run` clean — `task check`
  covers all of this.
- Keep the package layout flat and behavior-driven: don't add new
  `internal/` packages or abstractions ahead of the code that needs them.

## License

By contributing, you agree that your contributions will be licensed under
the project's [Apache License 2.0](LICENSE).
