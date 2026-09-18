# Security Policy

## Reporting a vulnerability

**Do NOT open a public GitHub issue for security vulnerabilities.**

Instead, email security@civicintelligence.dev (placeholder — replace with real address) with:
- A description of the vulnerability
- Steps to reproduce
- Affected versions
- Suggested fix (optional)

We will acknowledge receipt within 48 hours and provide an initial assessment within 7 days. We ask that you give us 90 days to address the issue before public disclosure.

## Threat model (summary)

The Civic Intelligence Platform handles public government information and citizen PII (email, preferences, follows). The threat model focuses on:

| Threat                                | Mitigation                                                                 |
| ------------------------------------- | -------------------------------------------------------------------------- |
| SSRF via crawler URLs                  | Crawler egress restricted to an allowlist; user-controlled URLs never directly drive crawler infrastructure; separate network namespace. |
| Prompt injection in AI                 | All AI output passes citation validation. System prompt explicitly forbids compliance with embedded instructions. |
| PII leakage via logs                   | Structured logging; PII fields redacted at the logger layer. No request bodies logged. |
| Supply-chain attacks via dependencies  | Dependabot weekly; pip-audit; npm audit --audit-level=moderate; Trivy filesystem + container scans; SLSA provenance on release artifacts (planned). |
| Privilege escalation via RBAC gaps     | Every protected route requires explicit permission; tests assert access matrix; deny-by-default. |
| Data exfiltration via search           | Rate limiting per IP and per user; bulk-export requires `researcher` role; query log audited. |
| Stale sessions                         | Sessions have `expires_at`; revocation supported; idle timeout enforced. |
| Token leakage in commits               | `gitleaks` pre-commit + CI scan; `.gitignore` covers `.env`, `*.pem`, `*.key`, `secrets/`. |

## Security controls

### Authentication & authorization
- **OIDC** via Keycloak (self-hosted) or any standards-compliant provider.
- **RBAC**: roles (`citizen`, `researcher`, `editor`, `admin`) + permissions (`bill:read`, `bill:follow`, `bill:question`, `ai:stream`, `search:query`, `briefing:read`, `editor:review`, `admin:users`, `admin:sources`).
- **Sessions**: server-side, with `expires_at`, `revoked_at`, IP, and user agent recorded.
- **API authorization**: every protected endpoint asserts the required permission; deny-by-default.

### Network
- TLS for all inter-service traffic (mTLS in prod via service mesh; planned).
- Crawler workers in a dedicated network namespace with egress allowlist (SSRF mitigation).
- All user-facing traffic behind HTTPS (HSTS, secure cookies).

### Input validation
- **Python**: Pydantic v2 models on every endpoint.
- **Go**: chi middleware + validator tags on every struct.
- **SQL**: parameterized queries everywhere — no string concatenation.
- **URLs**: schema + host validation; reject non-http(s); SSRF allowlist for crawler.

### Rate limiting
- Anonymous: 60 req/min per IP.
- Authenticated citizen: 300 req/min per user.
- AI question endpoints: 10 req/min per user (cost-protection).
- Search: 60 req/min per user.

### Secrets
- Never committed to git (`.gitignore` covers `.env`, `*.pem`, `*.key`, `secrets/`).
- In production: sealed secrets (Kubernetes) or Vault.
- Database credentials rotated quarterly.

### Audit logging
- `audit.log_entries` (append-only) records every write to canonical tables via triggers.
- Every API write records `actor_user_id`, `action`, `entity_type`, `entity_id`, `before`, `after`.
- Logs are retained for 1 year.

### Dependency & container scanning
- **Dependabot** for npm, pip, Go modules, Docker, GitHub Actions — weekly.
- **pip-audit** in CI for Python.
- **npm audit --audit-level=moderate** in CI for the frontend.
- **Trivy** filesystem + container scan in CI; CRITICAL/HIGH findings reported as SARIF.
- **CodeQL** SAST weekly + on PRs.

### AI-specific security
- **Prompt-injection defense**: the system prompt explicitly forbids acting on embedded instructions in retrieved evidence. The LLM is instructed to treat evidence as untrusted text.
- **Citation validation as a security control**: any AI response that asserts a fact without supporting evidence fails validation and is flagged to the user. This is the anti-hallucination gate.
- **Token budget enforcement**: the AI gateway refuses requests when the daily budget is exceeded.
- **No PII in prompts**: user identity is never included in prompts sent to LLM providers.

## Token rotation policy

- GitHub Personal Access Tokens: rotate every 90 days. Use fine-grained tokens with the minimum necessary scopes.
- OIDC client secrets: rotate every 180 days.
- Database credentials: rotate every 90 days.
- TLS certificates: rotate every 90 days via cert-manager.

## Incident response

1. **Detect**: alerting on `ai_failure_rate`, `citation_validation_failure_rate`, `workflow_failure_rate`, anomaly detection on traffic patterns.
2. **Triage**: on-call engineer acknowledges within 15 minutes during business hours, 60 minutes after hours.
3. **Contain**: rotate affected credentials; revoke affected sessions; isolate affected services.
4. **Eradicate**: deploy the fix; verify with regression tests + AI eval dataset.
5. **Recover**: restore service; verify no data loss; verify no canonical-state corruption.
6. **Postmortem**: blameless postmortem within 7 days; published for transparency.
