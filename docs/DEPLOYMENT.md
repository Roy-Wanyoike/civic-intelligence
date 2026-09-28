# Deployment Guide — Civic Intelligence Platform

This guide covers the **production deployment** of all three services that
make up the platform.

| Service | Tech | Vercel-native? | Recommended host |
|---------|------|----------------|------------------|
| `apps/web` | Next.js 14 (App Router) | ✅ Yes | **Vercel** (default) |
| `services/ai` | Python 3.12 (FastAPI) | ✅ Yes | **Vercel** (default) |
| `services/api` | Go 1.23 (BFF) | ❌ No | **Railway / Render / Fly.io** (external) |

The Go BFF is the only piece Vercel can't host natively (Vercel's Go
runtime is serverless-only — the BFF uses long-running background
workers, signal-based graceful shutdown, and a 2346-line main.go that
doesn't fit Vercel's function model). The fix: deploy the BFF externally
and point the Next.js app at it via the `API_BASE_URL` env var.

---

## Architecture overview

```
┌──────────────────────────────────────────────────────────────────┐
│  Vercel                                                          │
│  ┌──────────────────────┐         ┌──────────────────────────┐    │
│  │  apps/web (Next.js)  │ ──────► │  services/ai (FastAPI)   │    │
│  │                      │  /api/  │                          │    │
│  │  API_BASE_URL ───────┼─  v1/   │  vercel.json → service   │    │
│  │  env var             │  ai     └──────────────────────────┘    │
│  │                      │                                         │
│  │                      │ ──►  (external)                         │
│  └──────────────────────┘         │                                │
└───────────────────────────────────┼──────────────────────────────┘
                                    │
                                    ▼
┌──────────────────────────────────────────────────────────────────┐
│  Railway / Render / Fly.io                                      │
│  ┌──────────────────────────────────────────────────────────┐    │
│  │  services/api (Go BFF)                                    │    │
│  │                                                          │    │
│  │  - Crawls kenyalaw.org / parliament.go.ke                │    │
│  │  - In-memory + Postgres-backed repos                    │    │
│  │  - Proxies AI calls to services/ai                       │    │
│  │  - OIDC JWT verification (Keycloak)                      │    │
│  │  - Prometheus metrics                                    │    │
│  └──────────────────────────────────────────────────────────┘    │
└──────────────────────────────────────────────────────────────────┘
```

---

## 1. Deploy the Go BFF (Railway / Render / Fly.io)

The BFF ships with a Dockerfile (`services/api/Dockerfile`) that handles
the `replace` directives in `go.mod`. Build context MUST be the repo
root (NOT `services/api/`).

### Option A — Railway (recommended, easiest)

1. Push the repo to GitHub (already done — `Roy-Wanyoike/civic-intelligence`).
2. Go to <https://railway.app/new> → **Deploy from GitHub repo**.
3. Select the repo. Railway auto-detects the Dockerfile.
4. Set the **Root Directory** to `/` (repo root) and the **Dockerfile Path**
   to `services/api/Dockerfile`.
5. Add the env vars (see [§3 below](#3-required-env-vars)).
6. Railway assigns a public URL like `civic-intelligence-api.up.railway.app`.
   Copy it — you'll need it for the Vercel deployment.

### Option B — Render

1. Go to <https://dashboard.render.com> → **New +** → **Web Service**.
2. Connect the GitHub repo.
3. Set **Root Directory** to the repo root.
4. Set **Runtime** to **Docker** and **Dockerfile Path** to
   `services/api/Dockerfile`.
5. Add the env vars.
6. Render assigns `https://civic-intelligence-api.onrender.com`.

### Option C — Fly.io

```bash
# From the repo root:
flyctl launch --dockerfile services/api/Dockerfile --name civic-intelligence-api
# Edit fly.toml to set env vars, then:
flyctl deploy
flyctl apps open
```

### Verify the BFF

After deployment, hit the health endpoint:

```bash
curl https://<your-bff-host>/api/v1/healthz
# Expected: {"status":"ok",...}
```

---

## 2. Deploy the Web App + AI Service (Vercel)

The web app and AI service are Vercel-native — `vercel.json` already
wires them up. The only thing missing is the **`API_BASE_URL`** env var
that tells the Next.js app where the Go BFF lives.

1. Go to <https://vercel.com> → your `civic-intelligence` project →
   **Settings → Environment Variables**.
2. Add the following:

   | Name | Value | Environments |
   |------|-------|--------------|
   | `API_BASE_URL` | `https://<your-bff-host>` (e.g. `https://civic-intelligence-api.up.railway.app`) | Production, Preview, Development |
   | `API_SERVICE_URL` | same as `API_BASE_URL` | Production, Preview, Development |
   | `AI_SERVICE_URL` | (leave unset — Vercel's `vercel.json` rewrites route `/api/v1/ai/*` to the Python service automatically) | — |
   | `NEXT_PUBLIC_SITE_URL` | `https://civicintelligence.vercel.app` (or your custom domain) | Production |
   | `QWEN_API_KEY` | your Qwen API key (for the AI service) | Production, Preview |

3. **Redeploy** the project (Vercel → Deployments → Redeploy).

### Verify the web app

```bash
curl https://civicintelligence.vercel.app/bills
# Should render with the "✅ Live data" badge, not "Showing sample data".
```

If you still see "Showing sample data", the BFF env var is wrong or the
BFF is unreachable. Check the BFF health endpoint first, then the
`API_BASE_URL` value on Vercel.

---

## 3. Required env vars

### Go BFF (Railway / Render / Fly.io)

| Var | Default | Notes |
|-----|---------|-------|
| `API_SERVICE_ADDR` | `:9000` | Listen address. Change to `:8080` if the platform expects that port (Railway auto-detects; Render reads the `PORT` env var). |
| `DEV_MODE` | `false` | **Set `false` in production.** When `true`, JWT signatures are NOT verified — useful for local dev only. |
| `AI_SERVICE_URL` | `http://localhost:8000` | URL of the Python AI service. On Vercel, set to `https://civicintelligence.vercel.app/api/v1/ai` (the vercel.json rewrite). |
| `OIDC_ISSUER` | `http://localhost:8081/realms/civic` | Keycloak issuer URL. Required if `DEV_MODE=false`. |
| `OIDC_AUDIENCE` | `civic-intelligence` | Expected `aud` claim in JWTs. |
| `OIDC_JWKS_URL` | `…/protocol/openid-connect/certs` | JWKS endpoint for signature verification. |
| `DATABASE_URL` | (unset) | Postgres connection string. When unset, the BFF uses in-memory repos with seed data. |

### Next.js web app (Vercel)

| Var | Default | Notes |
|-----|---------|-------|
| `API_BASE_URL` | `http://localhost:9000` | **MUST be set on Vercel.** The base URL of the Go BFF (e.g. `https://civic-intelligence-api.up.railway.app`). |
| `API_SERVICE_URL` | `http://localhost:9000` | Used by `next.config.mjs` rewrites for relative `/api/v1/*` URLs. Set to the same value as `API_BASE_URL`. |
| `AI_SERVICE_URL` | `http://localhost:8000` | Used by `next.config.mjs` rewrites for `/api/v1/ai/*`. Leave unset on Vercel — `vercel.json` already routes those to the Python service. |
| `NEXT_PUBLIC_SITE_URL` | `https://civicintelligence.vercel.app` | Used for Open Graph + canonical URLs. |
| `QWEN_API_KEY` | (unset) | Required for AI Q&A features. Without it, the AI service returns a graceful "AI features unavailable" response. |

---

## 4. Local development

For end-to-end local testing (BFF + AI + web all running locally), use
`docker compose`:

```bash
# From the repo root:
docker compose up           # starts the Go BFF + Python AI service
# In another terminal:
cd apps/web && pnpm dev     # Next.js dev server (port 3000)
```

The Next.js dev server reads `API_BASE_URL` from `.env.local` (default
`http://localhost:9000`) and the Docker Compose BFF listens on
`localhost:9000`. They should connect automatically.

Verify the wiring:

```bash
# BFF is alive:
curl http://localhost:9000/api/v1/healthz

# Web app can reach the BFF:
curl http://localhost:3000/api/v1/bills
# Should return live JSON, not the mock fallback.
```

---

## 5. Operational notes

### Logs

- **Vercel**: Vercel dashboard → your project → **Logs**. Filter by
  environment + route. The Next.js error boundary (`app/error.tsx`)
  logs uncaught SSR errors to `console.error` — they appear here.
- **Railway / Render / Fly.io**: each platform has a logs tab. The BFF
  logs to stdout in structured JSON (see `packages/observability`).

### Health checks

Both Docker images ship with `HEALTHCHECK` directives:

- BFF: `GET /api/v1/healthz` (registered in `services/api/cmd/main.go`)
- AI service: `GET /healthz` (FastAPI route in `services/ai/app/main.py`)

Use these for uptime monitoring (e.g. UptimeRobot, Better Stack).

### When the BFF is down

The Next.js app is designed to **degrade gracefully** when the BFF is
unreachable:

- **Bills list / Bill detail**: fall back to curated mock Bills with a
  prominent amber "Showing sample data" banner.
- **Acts / lineage / audit / events**: render an amber "Try again" card
  with a retry link.
- **Search**: shows an amber "search service is unavailable" banner.
- **Homepage**: hides the "What Changed" + "Live Parliament" sections;
  Bills carousel falls back to mock Bills.

If the user sees a generic "Application error: a server-side exception
has occurred" page instead of one of these graceful states, it's a bug
in an error boundary — file an issue with the digest hash.

### Scaling

- The BFF is stateless (repos are in-memory or Postgres-backed) — scale
  horizontally by adding instances.
- The AI service is stateless — same.
- The Next.js app is serverless on Vercel — scales automatically.

For production with high traffic, switch the BFF from in-memory repos
to Postgres-backed ones (set `DATABASE_URL` and the repos auto-swap
behind the `legislation.Wire*` boundaries — no HTTP contract change).
