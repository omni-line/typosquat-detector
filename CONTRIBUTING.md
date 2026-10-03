# Contributing to Typosquat Detector

Thanks for helping. This guide covers setup, DCO, how corpora are refreshed,
and how to add a new package registry.

## Prerequisites

- Go **1.20+** (CI tests the minimum version and current stable on Linux, plus stable on macOS and Windows)
- `make` (optional)
- [`golangci-lint`](https://golangci-lint.run/) v1.61 (optional locally; enforced in CI)

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
2. Add/update tests for behavior changes. Parsers handle untrusted input:
   add a fuzz seed for any new syntax you support.
3. Run `make fmt`, `make lint`, `make race`, and (for parser changes) `make fuzz`.
4. Sign off commits (`git commit -s`) — [DCO](https://developercertificate.org/).

## Project layout

| Path | Role |
| --- | --- |
| `cmd/typosquat-detector` | Entrypoint |
| `internal/cli` | Flags, signals, exit codes |
| `internal/discover` | Manifest walk (never follows symlinks) |
| `internal/ecosystem` | One file per registry: manifests, normalization, URLs, purl |
| `internal/manifest` | Safe file reads + per-format parsers (`npm/`, `pypi/`) |
| `internal/corpus` | Embedded top-package snapshots, loaded by ecosystem name |
| `internal/distance` | Edit distance (OSA) and technique classification |
| `internal/scan` | Orchestration, severity grading |
| `internal/report` | Text / JSON / SARIF output + marketing |

## Design rules

- **Scanned trees are hostile input.** Read manifests only through
  `manifest.ReadFile`, which enforces size limits and rejects symlinks,
  FIFOs and devices. Sanitize anything printed to a terminal (`report.clean`).
- **One bad file must not hide the rest.** Parse failures become
  `scan.Warning`s; only usage errors and cancellation abort a scan.
- **Output contracts are public API.** JSON fields are additive; removing or
  renaming one requires bumping `report.JSONSchemaVersion`. SARIF rule IDs
  (`TSQ00x`) must stay stable so code-scanning alerts keep their history.
- **No runtime network access.** The corpus is embedded at build time.

## Adding a registry

Example: adding RubyGems as `gem`.

1. **Parser** — `internal/manifest/gem/gem.go`: return
   `[]manifest.Dependency` with name, version spec, group and 1-based line.
   Skip anything that doesn't resolve from the public registry (paths, git,
   URLs). Add table tests and a `FuzzParse` target, and register it in the
   `Makefile` `fuzz` target.
2. **Ecosystem** — `internal/ecosystem/gem.go` returning an `Ecosystem`:
   - `Name: "gem"` — also the corpus file name and the value in output.
   - `IsManifest`, `Parse`, `Normalize` (the registry's identity rules).
   - `PackageURL`, `PURL`/`PURLType` for links and SBOM correlation.
   - `Namespace` only if the registry has org namespaces worth peer-checking.
   - `Implied` only for well-known legitimate name patterns.
   Add it to `ecosystem.Default()`.
3. **Corpus** — add a `source` entry in `scripts/update-corpus/main.go` with a
   fetch function for a public top-downloads list, run `make corpus`, and
   commit `internal/corpus/gem.json.gz` plus the updated `meta.json`.
4. **Fixtures** — add `testdata/gem-typos/` and extend `internal/scan` tests.
5. **Docs** — README "How it works" and the CHANGELOG.

No changes to `scan`, `report`, or `corpus` should be needed. If they are,
that's a sign the `Ecosystem` abstraction is missing a hook. Raise it in the
PR so we can design it properly.

## Popular package corpus

Snapshots live in `internal/corpus/<ecosystem>.json.gz` and are embedded at build time.
Each file is a gzipped JSON array of `{"name","rank"}` objects (1-based download
rank), sorted alphabetically so monthly refresh diffs stay reviewable.
`meta.json` records `ordered_by: downloads` per ecosystem.

```bash
make corpus   # regenerates from public sources (needs network)
```

Sources:

- **PyPI:** https://hugovk.github.io/top-pypi-packages/top-pypi-packages-30-days.min.json
- **npm:** [npm-rank `raw.json`](https://github.com/LeoDog896/npm-rank/releases/download/latest/raw.json) (fallback curated seed if fetch fails)

The script refuses suspiciously small responses so a broken upstream can't
silently shrink the corpus. Do not hand-edit the gzip files; regenerate them
with the script.

## Security reports

See [SECURITY.md](SECURITY.md). Do not file public issues for vulnerabilities.
