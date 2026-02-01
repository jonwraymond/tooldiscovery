# Security Policy

## Supported Versions

| Version | Supported          |
| ------- | ------------------ |
| 0.2.x   | :white_check_mark: |

## Reporting a Vulnerability

If you discover a security vulnerability in tooldiscovery, please report it responsibly.

### How to Report

1. **Do NOT open a public GitHub issue** for security vulnerabilities.
2. **Email the maintainer** with details of the vulnerability:
   - Description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fixes (optional)
3. **Allow time for response** — we aim to respond within 48 hours and provide a fix timeline within 7 days.

### What to Expect

- **Acknowledgment**: We will acknowledge receipt within 48 hours.
- **Assessment**: We will assess severity and impact.
- **Fix Timeline**: For confirmed vulnerabilities, we will share an estimated fix timeline.
- **Disclosure**: We will coordinate public disclosure after a fix is available.
- **Credit**: With your permission, we will credit you in the advisory.

## Security Measures

### Input Validation
- Tool IDs are validated to prevent malformed namespaces and names
- Search and pagination inputs are sanitized and bounded

### Documentation Safety
- Tool example args are validated and capped for depth/size
- Documentation fields are truncated to prevent context pollution

### Dependencies
- Dependencies are scanned with `govulncheck`
- Security scanning runs in CI via `gosec`

## Scope

This policy applies to:
- `index` (registry, search orchestration, pagination)
- `search` (BM25 implementation)
- `semantic` (embedding strategies and indexing)
- `tooldoc` (progressive documentation store)
- `discovery` (facade and hybrid search)

## Out of Scope

- Vulnerabilities in upstream dependencies (report to maintainers)
- Example code intended for demonstration only
- Theoretical issues without demonstrated impact
