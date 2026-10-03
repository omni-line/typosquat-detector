# Typosquat Detector

[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/omni-line/typosquat-detector/actions/workflows/ci.yml/badge.svg)](https://github.com/omni-line/typosquat-detector/actions/workflows/ci.yml)
[![Powered by Omni Line](https://img.shields.io/badge/Powered%20by-Omni%20Line-FF4B4B?style=flat)](https://omniline.app/)

**Typosquat Detector** is a fast, offline CLI that scans your project for **typosquatting** risk. It discovers npm, PyPI, Composer, Go, Cargo, Maven, RubyGems, Docker, and Conan manifests, compares each declared dependency name to an embedded corpus of high-download packages, and reports names that are **1–2 edits away** from a popular package — classic near-miss typos like `reqeusts` → `requests` or `crossenv` → `cross-env`.

Sibling to [omni-audit](https://github.com/omni-line/omni-audit) (dependency confusion / unclaimed names). Distributed as a **standalone Go binary**. No Node or Python runtime required.

## The problem

Attackers publish packages whose names look like popular libraries (`react-domm`, `crossenv`). A single typo in `package.json` or `requirements.txt` can install malware that steals env vars and CI tokens.

Detection is the first step. [Omni Line](https://omniline.app) is the durable fix: proxy public registries and **allow-list** approved externals so unknown near-miss names never resolve.

## Install

### Prebuilt binaries

Download the latest release from
[GitHub Releases](https://github.com/omni-line/typosquat-detector/releases).
Every release ships `checksums.txt` (cosign-signed), per-archive SBOMs, and
GitHub attestations — see [Verify a release](#verify-a-release).

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

# Scan a path (directory or single manifest)
typosquat-detector ./apps/api

# JSON for scripts, SARIF for GitHub code scanning / SAST dashboards
typosquat-detector --format json --no-marketing
typosquat-detector --format sarif > typosquat.sarif

# Only fail CI on the highest-confidence findings
typosquat-detector --fail-on critical

# Suppress known-safe near-misses
typosquat-detector --allow mylib-utils --ignore 'internal-*'
```

Example text output:

```text
✗ 4 typosquat findings: 3 critical, 1 high

CRITICAL  npm  crossenv@7.0.3  npm-typos/package.json:4 (dependencies)
  did you mean  cross-env (#625 on npm)
  why           differs only by '-', '_' or '.' separators · 1 edit from a popular npm package
  compare       https://www.npmjs.com/package/crossenv
                https://www.npmjs.com/package/cross-env
  fix           use "cross-env" if that was intended; if "crossenv" is genuinely yours, add --allow crossenv

CRITICAL  pypi  reqeusts@2.1.0  pypi-typos/requirements.txt:2 (requirements)
  did you mean  requests (#7 on pypi)
  why           two adjacent characters swapped · 1 edit from a popular pypi package
  compare       https://pypi.org/project/reqeusts/
                https://pypi.org/project/requests/
  fix           use "requests" if that was intended; if "reqeusts" is genuinely yours, add --allow reqeusts

HIGH      npm  @acme/authh@1.0.0  npm-typos/package.json:8 (dependencies)
  did you mean  @acme/auth
  why           one extra character · 1 edit from another package in the same namespace
  ...

Scanned 3 manifests, 10 packages · 4 findings · 0 skipped · in 47ms
Corpus 2026-10-01 (npm 5,247, pypi 10,000 top packages) · max distance 2
```

`-q` prints one grep-friendly line per finding:

```text
CRITICAL npm crossenv@7.0.3 npm-typos/package.json:4 (dependencies) -> cross-env (#625 on npm) (separator, distance 1)
```

## How it works

1. Walks the tree for supported manifests (`package.json`, Composer, `go.mod`, `Cargo.toml`, `pom.xml`, `Gemfile`, Dockerfile/Compose, `conanfile.txt`, PyPI requirements/`pyproject.toml`). Skips `node_modules`, `.venv`, `.git`, and other cache dirs. Symlinks and non-regular files are never followed or read.
2. Collects **direct** dependencies only (registry-resolvable names; skips path/git/local sources).
3. Loads an **embedded** snapshot of top package names per ecosystem (no network at scan time). npm uses ~25k names; others use ~10k. Snapshots include download ranks.
4. For each name not in the corpus, computes the edit distance to length-bucketed candidates. Adjacent swaps (`reqeusts`) and separator changes (`crossenv`) count as a single edit. Short names (under 5 characters) are only flagged at distance 1 to keep noise down.
5. Within namespaced packages (npm `@scope/...`, Composer `vendor/...`, Maven `groupId`), flags peers that are 1–2 edits apart. Sibling families such as `@acme/ui` / `@acme/ui-kit` are not flagged.
6. If a manifest can't be read or parsed, that becomes a warning and the scan continues. Use `--strict` to fail on warnings.

Corpus snapshot date is recorded in `internal/corpus/meta.json`, shown in every report, and refreshed monthly via CI.

### Severity

| Severity | Meaning |
| --- | --- |
| `critical` | 1 edit from a popular package — the classic typosquat |
| `high` | 2 edits from a popular package, or 1 edit from a package in the same namespace |
| `medium` | 2 edits from a package in the same namespace |

Each finding also reports a `technique` (`separator`, `transposition`, `extra-character`, `missing-character`, `substitution`, `multiple-edits`), a [package URL](https://github.com/package-url/purl-spec), and registry links for both names so reviewers can compare them side by side.

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
| `0` | No findings at or above `--fail-on` (or `--fail-on none`) |
| `1` | One or more findings at or above `--fail-on` (default: any) |
| `2` | Usage or runtime error, interrupted, or warnings with `--strict` |

### Flags

| Flag | Description |
| --- | --- |
| `--format text\|json\|sarif` | Output format (default `text`) |
| `--distance 1\|2` | Max edit distance to flag (default `2`) |
| `--fail-on any\|critical\|high\|medium\|none` | Minimum severity that exits 1 (default `any`) |
| `--strict` | Exit 2 when a manifest cannot be read or parsed |
| `--scope '@org'` | npm scopes for peer typo checks (repeatable) |
| `--no-scope-peers` | Disable auto intra-scope peer checks |
| `--ignore` | Skip package name globs |
| `--allow` | Exact package names to treat as safe |
| `--exclude` | Skip paths relative to the scan root |
| `-q` / `--quiet` | One line per finding; no banner, summary, or marketing |
| `--no-marketing` | Hide Omni Line CTA / JSON `sponsor` |
| `--marketing` | Force marketing even when non-TTY |
| `--color auto\|always\|never` | ANSI colors (default `auto` on TTY) |
| `--version` | Print version |

### JSON output

`--format json` emits a document with `schema_version` (bumped only on breaking changes), `version`, `scan` (root, max distance, duration, corpus metadata), `findings`, `warnings`, and `stats` (including `by_severity`). New fields may be added without a schema bump, so consumers should ignore unknown keys.

### CI examples

GitHub code scanning (alerts show up in the Security tab and on pull requests):

```yaml
- name: Typosquat scan
  run: |
    VERSION=0.4.0  # pin a release
    BASE=https://github.com/omni-line/typosquat-detector/releases/download/v${VERSION}
    curl -sSfLO "$BASE/typosquat-detector_Linux_x86_64.tar.gz"
    curl -sSfLO "$BASE/checksums.txt"
    curl -sSfLO "$BASE/checksums.txt.sigstore.json"
    cosign verify-blob \
      --certificate-identity "https://github.com/omni-line/typosquat-detector/.github/workflows/release.yml@refs/tags/v${VERSION}" \
      --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
      --bundle checksums.txt.sigstore.json \
      checksums.txt
    sha256sum --check --ignore-missing checksums.txt
    tar xzf typosquat-detector_Linux_x86_64.tar.gz typosquat-detector
    ./typosquat-detector --format sarif --fail-on none > typosquat.sarif
- uses: github/codeql-action/upload-sarif@v3
  with:
    sarif_file: typosquat.sarif
```

Plain gate:

```yaml
- run: ./typosquat-detector --no-marketing --fail-on critical
```

## Verify a release

Releases include a cosign-signed `checksums.txt`, SPDX SBOMs (`*.sbom.json`), and
GitHub attestations (SLSA provenance).

```bash
VERSION=v0.4.0
BASE=https://github.com/omni-line/typosquat-detector/releases/download/${VERSION}

curl -sSfLO "$BASE/checksums.txt"
curl -sSfLO "$BASE/checksums.txt.sigstore.json"
cosign verify-blob \
  --certificate-identity "https://github.com/omni-line/typosquat-detector/.github/workflows/release.yml@refs/tags/${VERSION}" \
  --certificate-oidc-issuer "https://token.actions.githubusercontent.com" \
  --bundle checksums.txt.sigstore.json \
  checksums.txt

curl -sSfLO "$BASE/typosquat-detector_Linux_x86_64.tar.gz"
sha256sum --check --ignore-missing checksums.txt

gh attestation verify typosquat-detector_Linux_x86_64.tar.gz \
  --repo omni-line/typosquat-detector
```

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) (including how to add a new registry), [MAINTAINERS.md](MAINTAINERS.md), and
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
