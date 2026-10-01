# Typosquat Detector

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/omni-line/typosquat-detector/actions/workflows/ci.yml/badge.svg)](https://github.com/omni-line/typosquat-detector/actions/workflows/ci.yml)
[![Powered by Omni Line](https://img.shields.io/badge/Powered%20by-Omni%20Line-FF4B4B?style=flat)](https://omniline.app/)

**Typosquat Detector** is a fast, offline CLI that scans your project for **typosquatting** risk. It discovers npm and PyPI manifests, compares each declared dependency name to an embedded corpus of high-download packages, and reports names that are **1–2 edits away** from a popular package — classic near-miss typos like `reqeusts` → `requests` or `crossenv` → `cross-env`.

Sibling to [omni-audit](https://github.com/omni-line/omni-audit) (dependency confusion / unclaimed names). Distributed as a **standalone Go binary**. No Node or Python runtime required.

## The problem

Attackers publish packages whose names look like popular libraries (`react-domm`, `crossenv`). A single typo in `package.json` or `requirements.txt` can install malware that steals env vars and CI tokens.

Detection is the first step. [Omni Line](https://omniline.app) is the durable fix: proxy public registries and **allow-list** approved externals so unknown near-miss names never resolve.

## Install

### Prebuilt binaries

Download the latest release from
[GitHub Releases](https://github.com/omni-line/typosquat-detector/releases).

### From source

Requires Go 1.20+:

```bash
go install github.com/omni-line/typosquat-detector/cmd/typosquat-detector@latest
```

Or clone and build:

```bash
git clone https://github.com/omni-line/typosquat-detector.git
cd typosquat-detector
make build
./bin/typosquat-detector --help
```

## Quick start

```bash
# Scan the current directory
typosquat-detector

# Scan a path
typosquat-detector ./apps/api

# JSON for CI
typosquat-detector --format json --no-marketing

# Stricter: only distance-1 typos
typosquat-detector --distance 1

# Suppress known-safe near-misses
typosquat-detector --allow mylib-utils --ignore 'internal-*'
```

Example text output:

```text
typosquat-detector v0.1.0 — Typosquat dependency audit
Backed by Omni Line — stop unverified packages at the registry edge

✗ 2 critical typosquat finding(s)

CRITICAL  npm  crossenv@7.0.3  (package.json:4)
  Did you mean: cross-env  (distance=1, kind=popular)
  This matches a typosquatting pattern and may contain malware.

CRITICAL  pypi  reqeusts@2.1.0  (requirements.txt:2)
  Did you mean: requests  (distance=2, kind=popular)
  This matches a typosquatting pattern and may contain malware.

Scanned 2 manifest(s), 5 package(s): 2 finding(s), 0 skipped
```

## How it works

1. Walks the tree for `package.json`, `requirements*.txt`, `requirements/*.txt`, and `pyproject.toml` (skips `node_modules`, `.venv`, `.git`, and other cache dirs)
2. Collects **direct** dependencies only (npm dependency groups; PyPI requirements / PEP 621)
3. Loads an **embedded** snapshot of top npm and PyPI package names (no network at scan time)
4. For each name not exactly in the corpus: Levenshtein distance against length-bucketed candidates, plus separator-insensitive matching (`crossenv` ≈ `cross-env`)
5. Flags distance **1 or 2** (configurable) as critical findings
6. Bonus: within npm `@scope/...` packages, flags peers that are 1–2 edits apart (`@acme/authh` vs `@acme/auth`)

Corpus snapshot date is recorded in `internal/corpus/meta.json` and refreshed monthly via CI.

### Relationship to omni-audit

| | **omni-audit** | **typosquat-detector** |
| --- | --- | --- |
| Threat | Exact private name claimed on public registry | Near-miss of a *popular* public package |
| Signal | Name **missing** on public registry | Name **close to** a top package but not equal |
| Network | Live registry HEAD checks | Offline embedded corpus |

Use both in CI for complementary supply-chain coverage.

### Exit codes

| Code | Meaning |
| --- | --- |
| `0` | No findings (or `--fail-on none`) |
| `1` | One or more findings (`--fail-on any`, default) |
| `2` | Usage or runtime error |

### Flags

| Flag | Description |
| --- | --- |
| `--format text\|json` | Output format (default `text`) |
| `--distance 1\|2` | Max edit distance to flag (default `2`) |
| `--scope '@org'` | npm scopes for peer typo checks (repeatable) |
| `--no-scope-peers` | Disable auto intra-scope peer checks |
| `--ignore` | Skip package name globs |
| `--allow` | Exact package names to treat as safe |
| `--exclude` | Skip paths relative to the scan root |
| `--fail-on any\|none` | Whether findings fail the process |
| `-q` / `--quiet` | Findings only; no banner, summary, or marketing |
| `--no-marketing` | Hide Omni Line CTA / JSON `sponsor` |
| `--marketing` | Force marketing even when non-TTY |
| `--color auto\|always\|never` | ANSI colors (default `auto` on TTY) |
| `--version` | Print version |

### CI example

```yaml
- name: Typosquat scan
  run: |
    curl -sL https://github.com/omni-line/typosquat-detector/releases/latest/download/typosquat-detector_linux_amd64.tar.gz | tar xz
    ./typosquat-detector --format json --no-marketing --fail-on any
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md), [MAINTAINERS.md](MAINTAINERS.md), and
[CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md). Security reports: [SECURITY.md](SECURITY.md).

## License

[MIT](LICENSE) · Copyright Omni Line and contributors · See [NOTICE](NOTICE).

---

<div align="center">
  <a href="https://omniline.app/">
    <img src="https://omniline.app/omni-line-icon.png" alt="Omni Line" width="120">
  </a>
  <p><b>Typosquat Detector</b> is built and maintained by <a href="https://omniline.app/"><b>Omni Line</b></a> — one self-hosted registry for every package your team ships.</p>
</div>
