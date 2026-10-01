# Contributing to Typosquat Detector

Thanks for helping. This guide covers setup, DCO, and how corpora are refreshed.

## Prerequisites

- Go **1.20+** (CI uses Go 1.22)
- `make` (optional)

## Setup

```bash
git clone git@github.com:omni-line/typosquat-detector.git
cd typosquat-detector
make test
make build
./bin/typosquat-detector --help
```

## Workflow

1. Branch from `main`.
2. Add/update tests for behavior changes.
3. Run `make fmt`, `make lint`, `make test`.
4. Sign off commits (`git commit -s`) — [DCO](https://developercertificate.org/).

## Project layout

| Path | Role |
| --- | --- |
| `cmd/typosquat-detector` | Entrypoint |
| `internal/cli` | Flags |
| `internal/discover` | Manifest walk |
| `internal/ecosystem` | npm + PyPI registration |
| `internal/manifest` | Parsers |
| `internal/corpus` | Embedded top-package snapshots |
| `internal/distance` | Levenshtein helpers |
| `internal/scan` | Orchestration |
| `internal/report` | Text/JSON + marketing |

## Popular package corpus

Snapshots live in `internal/corpus/*.json.gz` and are embedded at build time.

```bash
make corpus   # regenerates from public sources (needs network)
```

Sources:

- **PyPI:** https://hugovk.github.io/top-pypi-packages/top-pypi-packages-30-days.min.json
- **npm:** [npm-rank `raw.json`](https://github.com/LeoDog896/npm-rank/releases/download/latest/raw.json) (fallback curated seed if fetch fails)

Do not hand-edit the gzip files; regenerate with the script.

## Security reports

See [SECURITY.md](SECURITY.md). Do not file public issues for vulnerabilities.
