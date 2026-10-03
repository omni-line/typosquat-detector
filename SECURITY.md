# Security Policy

## Supported versions

| Version | Supported |
| --- | --- |
| latest `v*` release | Yes |

## Reporting a vulnerability

**Do not file a public GitHub issue.**

1. [GitHub Security Advisories](https://github.com/omni-line/typosquat-detector/security/advisories/new) (preferred)
2. Email: **security@omniline.app**

Findings about near-miss package names are heuristic risk signals, not proof of malware.

## Verifying release artifacts

Each release publishes:

- `checksums.txt` — SHA-256 of archives
- `checksums.txt.sigstore.json` — keyless cosign signature (GitHub OIDC)
- `*.sbom.json` — SPDX SBOM per archive (syft)
- GitHub attestations — SLSA build provenance (`gh attestation verify`)

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

gh attestation verify typosquat-detector_Linux_x86_64.tar.gz \
  --repo omni-line/typosquat-detector
```
