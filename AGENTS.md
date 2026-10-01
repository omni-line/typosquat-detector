# AGENTS.md

Guidance for AI coding agents in this repository.

## Project

**Typosquat Detector** (`typosquat-detector`) is a Go CLI that flags dependency
names that are 1–2 edits away from a high-download public package (npm / PyPI).
It is an open-source project backed by [Omni Line](https://omniline.app).
Sibling: [omni-audit](https://github.com/omni-line/omni-audit) (dependency confusion).

Module: `github.com/omni-line/typosquat-detector`

## Commands

```bash
make test
make build
make corpus    # refresh embedded snapshots (network)
./bin/typosquat-detector --help
```

Go **1.20+**. Prefer the standard library. Do not import `omni-audit`.

## Layout

| Path | Role |
| --- | --- |
| `cmd/typosquat-detector` | Entrypoint |
| `internal/cli` | Flags |
| `internal/discover` | Walk |
| `internal/ecosystem` | npm + PyPI |
| `internal/manifest` | Parsers |
| `internal/corpus` | Embedded popular names |
| `internal/distance` | Levenshtein |
| `internal/scan` | Orchestration |
| `internal/report` | Output + marketing |

Marketing copy lives only in `internal/report/marketing.go`.

## Out of scope (unless asked)

Composer/Go/Cargo, lockfile-only scans, OSV malware lookups, keyboard adjacency,
runtime corpus download, cutting release tags without an explicit request.
