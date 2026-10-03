# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.3.0] - 2026-10-03

### Added

- Corpus snapshots store a 1-based download `rank` per package (`ordered_by: downloads` in `meta.json`)
- Findings expose `target_rank` for the primary suggestion (text, JSON, SARIF); omitted when the corpus has no ranks
- Text output annotates suggestions as `cross-env (#625 on npm)`

### Changed

- Suggestion ties break by ascending download rank, then name
- Corpus files are `[{name, rank}, ...]` sorted alphabetically for reviewable monthly diffs (legacy name-only arrays still load)

## [0.2.0] - 2026-10-02

### Added

- `--format sarif` (SARIF 2.1.0) for GitHub code scanning and SAST dashboards, with stable rule IDs `TSQ001`–`TSQ004`
- Graded severities (`critical`, `high`, `medium`) and `--fail-on critical|high|medium`
- Findings now include `technique`, `message`, `purl`, `registry_url` and `suggestion_url`
- JSON output: `schema_version`, `scan` metadata (root, max distance, duration, corpus snapshot), `warnings`, `stats.by_severity`
- Richer text output: location with dependency group, explanation, registry links, fix hint, corpus date, scan duration
- `--strict` to exit 2 when a manifest cannot be read or parsed
- PyPI: `[build-system].requires`, PEP 735 `[dependency-groups]`, and Poetry dependency tables
- `--version` shows commit and build date; `go install` builds report the module version
- Fuzz tests for all parsers and the distance kernel; govulncheck, race detector, and macOS/Windows in CI; Dependabot

### Changed

- Adjacent-character swaps count as one edit (`reqeusts` → `requests` is now distance 1)
- Names shorter than 5 characters are only flagged at distance 1, which cuts noise such as `@acme/ui` vs `@acme/api`
- npm names compare case-insensitively against a normalized corpus
- Ecosystems are self-contained (`internal/ecosystem/<name>.go`) and corpora load by name, so adding a registry needs no changes to scan or report code
- `meta.json` uses a per-ecosystem `ecosystems` map
- Findings sort by severity, then ecosystem, manifest and line
- About 7x faster on real monorepos (memoized lookups and bounded distance computation)

### Fixed

- **Security:** symlinked, FIFO and device manifests are no longer read; a FIFO named `package.json` previously hung the scan forever
- **Security:** SIGINT/SIGTERM were captured but ignored, so the process could not be stopped; they now cancel the scan, and a second signal exits immediately
- **Security:** terminal sanitization now neutralizes C1 controls, DEL, bidi overrides and zero-width characters
- A typo repeated in several manifests was reported only once; every location is now reported
- One malformed manifest aborted the whole scan; it is now a warning
- `--allow` / `--ignore` did not suppress scope-peer findings
- `pyproject.toml`: a quote inside a `#` comment broke array parsing and hid dependencies; line numbers were relative to the section instead of the file
- `requirements.txt`: inline `# comments` and `\` continuations leaked into versions
- `@types/<pkg>` for popular packages and sibling families like `@scope/swagger` / `@scope/swagger-ui` are no longer flagged
- Banner showed `vv0.1.0` for `make build` binaries
- `update-corpus`: HTTP timeouts and size limits added; alphabetical truncation could drop popular names
- README download URL did not match the release archive name

## [0.1.0] - 2026-10-01

### Added

- Initial typosquat-detector CLI: npm (`package.json`) and PyPI (`requirements*.txt`, `pyproject.toml`)
- Embedded top-package corpus with Levenshtein distance 1–2 alerts
- Separator-insensitive matching (`crossenv` vs `cross-env`)
- Intra-scope peer checks for npm `@org/...` packages
- Text/JSON output, colors, Omni Line marketing CTA
