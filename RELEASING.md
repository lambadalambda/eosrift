# Releasing EosRift

This repo is pre-alpha; we are not cutting regular releases yet. This document captures what
“release-ready” means for EosRift and provides a repeatable checklist when we decide to tag.

## Versioning

- Tags use SemVer: `vMAJOR.MINOR.PATCH`
- Pre-1.0 releases are expected to be `v0.x.y` while the CLI/config/protocol are still settling.

## “v1.0-ready” definition (high level)

Before tagging `v1.0.0`, we should be confident in:

- **CLI stability:** flags, help text, and output formatting are treated as API.
- **Config stability:** `eosrift.yml` keys and precedence are stable and documented.
- **Protocol stability:** control-plane JSON schema has a migration story for changes.
- **Security defaults:** safe-by-default deployment posture is documented and tested.
- **Operational basics:** logs/metrics/health checks work and are documented.

## Release checklist (any tag)

Engineering:

- [ ] `./scripts/go test ./...` is green.
- [ ] `docker compose -f docker-compose.test.yml up --build --exit-code-from test --abort-on-container-exit` is green.
- [ ] Run a small smoke load test:
  - [ ] `EOSRIFT_LOAD_REQUESTS=200 EOSRIFT_LOAD_CONCURRENCY=20 docker compose -f docker-compose.loadtest.yml up --build --exit-code-from loadtest --abort-on-container-exit`
- [ ] Docs reflect reality (`README.md`, `deploy/PRODUCTION.md`, `ARCHITECTURE.md`, `SECURITY.md`, `PLAN.md`).

Release hygiene:

- [ ] Move `CHANGELOG.md` entries from `[Unreleased]` into a new version section.
- [ ] Ensure the version is injected in release builds (`-ldflags -X eosrift.com/eosrift/internal/cli.version=...`).
- [ ] Create and push a signed tag: `git tag -s vX.Y.Z -m "vX.Y.Z"` (optional but recommended).
- [ ] Push: `git push --tags`.

Verification (post-tag):

- [ ] GitHub Actions release workflow uploads artifacts.
- [ ] Client install script works against the new release (`scripts/install.sh --version vX.Y.Z`).
- [ ] Docker image workflow publishes multi-arch images (if enabled for that tag).

## Dry-run release builds (no tag)

To validate the release pipeline end-to-end without creating a tag:

- Run the **Release** workflow via GitHub Actions → “Run workflow”.
- Set `version` to something like `v0.1.0-dryrun` (this is embedded in the binaries).
- Download the `dist-<version>` workflow artifact and verify:
  - `checksums.txt` exists and has a `.sig` + `.pem` (cosign keyless signing)
  - the tarballs contain the expected binaries (`eosrift`, `eosrift-server`)
