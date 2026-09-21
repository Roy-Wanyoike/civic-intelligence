# Deployment Guide

## Architecture Overview

The Civic Intelligence Platform is a three-tier polyglot application:

| Tier | Service | Stack | Deploy Target |
|------|---------|-------|---------------|
| Frontend | `apps/web/` | Next.js 14 + React 18 + TypeScript + Tailwind | Vercel |
| BFF / API | `services/api/` | Go 1.23 + stdlib `net/http` | Railway / Render / Fly.io |
| AI service | `services/ai/` | Python 3.12 + FastAPI + uvicorn | Railway / Render / Fly.io |
| Database | — | PostgreSQL 16 + pgvector | Supabase / Neon / RDS |

The frontend talks to the API and AI services via two Vercel rewrites
defined in `vercel.json`:

```text
/api/v1/ai/*   →  AI_SERVICE_URL/v1/*
/api/v1/*      →  API_SERVICE_URL/api/v1/*
```

The API service hosts the cron endpoint (`POST /api/v1/refresh`) that
re-discovers Bills, Hansard, Order Papers, Votes & Proceedings, committee
reports, and gazette notices from the 11 official Parliament of Kenya
sources + 1 Kenya Law gazette source.

---

## Vercel (Frontend)

The Next.js frontend deploys to Vercel.

### Vercel multi-service setup

This repo ships with a `vercel.json` that defines two services:

```json
{
  "services": {
    "web": { "root": "apps/web", "framework": "nextjs" },
    "ai":  { "root": "services/ai" }
  }
}
```

When you import the repo at the top level, Vercel will deploy BOTH
services. The `web` service is the Next.js frontend (includes the
API rewrites). The `ai` service is the Python FastAPI service.

If you prefer to deploy the frontend only, you can also import the
repo with the root directory set to `apps/web` (the older single-
service pattern). Either works.

### Setup

1. Go to [vercel.com/new](https://vercel.com/new)
2. Import `Roy-Wanyoike/civic-intelligence`
3. If you want both services: leave the root at the repo top level.
   If you want frontend only: set **Root Directory** to `apps/web`.
4. Vercel auto-detects Next.js for `web` and Python for `ai`.
5. Set environment variables (see below)
6. Deploy

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `AI_SERVICE_URL` | No* | `http://localhost:8000` | Python AI service URL |
| `API_SERVICE_URL` | No* | `http://localhost:9000` | Go API service URL |

*If not set, the frontend falls back to mock data. Set these to your backend URLs for real data.

### Cron Jobs (Hobby plan limitation)

The repo's `vercel.json` defines one cron:

```json
"crons": [
  { "path": "/api/v1/refresh", "schedule": "0 3 * * *" }
]
```

**Vercel Hobby accounts only allow cron jobs that run at most once per day.**
The `0 3 * * *` schedule runs once a day at 03:00 UTC — this is the
maximum frequency allowed on the Hobby plan. If you see this deploy
error:

> Hobby accounts are limited to daily cron jobs. This cron expression
> (0 * * * *) would run more than once per day.

…you are still on the old hourly schedule. Pull the latest `main` and
redeploy — the schedule was changed from `0 * * * *` (hourly) to
`0 3 * * *` (daily) in PR #254.

To run the refresh more frequently than once a day, either:
- Upgrade to Vercel Pro, or
- Set up an external scheduler (GitHub Actions `schedule:` cron, Railway
  cron, Render cron, or a cloud function) that POSTs to your deployed
  `/api/v1/refresh` endpoint.

### What deploys to Vercel

- Next.js frontend (73 pages)
- Static assets
- The `apps/web/` directory (or both services if you import at the repo root)

### What does NOT deploy to Vercel

- Go backend (deploy separately — Railway is the recommended target)
- Database (use Supabase, Neon, or RDS)

---

## Backend Services (Go API + Python AI)

Deploy the Go API and Python AI service separately. Recommended platforms:

### Option A: Railway (recommended — supports both Go + Python)
1. Go to [railway.app](https://railway.app)
2. Create two services:
   - **Go API**: root = `services/api/`, build = `go build -o api ./cmd/`, start = `./api`
   - **Python AI**: root = `services/ai/`, build = `pip install -r requirements.txt`, start = `uvicorn app.main:app --host 0.0.0.0 --port $PORT`
3. Set env vars on Railway:
   - `DATABASE_URL` — your Postgres connection string
   - `OPENAI_API_KEY` — for real AI
   - `DEV_MODE=false` — for production auth
4. Copy the Railway URLs back to Vercel env vars:
   - `AI_SERVICE_URL` = Railway Python service URL
   - `API_SERVICE_URL` = Railway Go service URL

### Option B: Render
Same approach — two web services on [render.com](https://render.com).

### Option C: Fly.io
Deploy as Docker containers using the existing Dockerfiles.

---

## Database

Use a managed PostgreSQL with pgvector:

### Supabase (recommended — free tier)
1. Go to [supabase.com](https://supabase.com)
2. Create a new project
3. Enable the `pgvector` extension (Supabase supports it natively)
4. Run migrations:
   ```bash
   psql $DATABASE_URL -f infrastructure/postgres/migrations/001_extensions.up.sql
   psql $DATABASE_URL -f infrastructure/postgres/migrations/002_schemas.up.sql
   # ... continue for all 17 migrations
   ```
5. Copy the connection string to your backend env vars

### Neon
Alternative — [neon.tech](https://neon.tech) also supports pgvector.

---

## M-Pesa + Stripe (Sponsor payments)

### M-Pesa (Daraja API)
Set these env vars on your Go API service:
- `MPESA_CONSUMER_KEY` — from [developer.safaricom.co.ke](https://developer.safaricom.co.ke)
- `MPESA_CONSUMER_SECRET`
- `MPESA_SHORTCODE` — your Paybill/Till number
- `MPESA_PASSKEY`
- `MPESA_CALLBACK_URL` — your Go API URL + `/api/v1/sponsor/mpesa/callback`

### Stripe
Set this env var on your Go API service:
- `STRIPE_SECRET_KEY` — from [dashboard.stripe.com](https://dashboard.stripe.com)

---

## Data Sources Crawled

The Go API's `/api/v1/refresh` endpoint (invoked daily at 03:00 UTC by the
Vercel cron) crawls the following 12 official Kenyan sources. The crawler
implementations live in `adapters/kenya/`:

| Source | URL | Document type |
|--------|-----|---------------|
| NA Bills | `parliament.go.ke/the-national-assembly/house-business/bills` | bill |
| Senate Bills | `parliament.go.ke/the-senate/senate-bills` | bill |
| NA Bill Tracker | `parliament.go.ke/the-national-assembly/house-business/bill-tracker` | bill_tracker |
| NA Hansard | `parliament.go.ke/the-national-assembly/house-business/hansard` | hansard |
| Senate Hansard | `parliament.go.ke/the-senate/Hansard` (capital H) | hansard |
| NA Order Paper | `parliament.go.ke/the-national-assembly/house-business/order-paper` | order_paper |
| Senate Order Paper | `parliament.go.ke/the-senate/house-business/order-paper` | order_paper |
| NA Votes & Proceedings | `parliament.go.ke/the-national-assembly/house-business/votes-proceeding` (singular) | votes_proceedings |
| Senate Votes & Proceedings | `parliament.go.ke/the-senate/house-business/votes-proceeding` | votes_proceedings |
| NA Committees | `parliament.go.ke/the-national-assembly/committees` | committee_report |
| Senate Committees | `parliament.go.ke/the-senate/committees/senate-committees` | committee_report |
| Kenya Gazette | `new.kenyalaw.org/kenya_law/gazette/` | gazette_notice |

All crawls go through the `PoliteClient` (1 request/sec/host) so the
crawler is a good citizen on the source sites. The `User-Agent` header
includes the project URL so source administrators can reach the
platform operators if the crawler misbehaves.

---

## Country Subdomains (Future Feature)

Yes — `ke.civicintelligence.com`, `ug.civicintelligence.com`, etc. is absolutely possible.

### Architecture

```text
ke.civicintelligence.com  →  Kenya data (Bills, Acts, terminology from Kenya adapter)
ug.civicintelligence.com  →  Uganda data
tz.civicintelligence.com  →  Tanzania data
gh.civicintelligence.com  →  Ghana data
ng.civicintelligence.com  →  Nigeria data
za.civicintelligence.com  →  South Africa data
rw.civicintelligence.com  →  Rwanda data
zm.civicintelligence.com  →  Zambia data
sn.civicintelligence.com  →  Senegal data
eg.civicintelligence.com  →  Egypt data
ma.civicintelligence.com  →  Morocco data
cd.civicintelligence.com  →  DR Congo data
et.civicintelligence.com  →  Ethiopia data
mw.civicintelligence.com  →  Malawi data
```

### Implementation plan

1. **Next.js Middleware** detects the subdomain
2. Sets `country` context (KE, UG, TZ, GH, NG, ZA, RW, ZM, SN, EG, MA, CD, ET, MW)
3. All API calls include the country parameter
4. Go API routes to the correct country adapter
5. Frontend renders country-specific data (Bills, stages, terminology)

```typescript
// apps/web/src/middleware.ts (future)
export function middleware(request: NextRequest) {
  const host = request.headers.get('host') ?? '';
  const subdomain = host.split('.')[0]; // 'ke', 'ug', etc.
  const countryCode = VALID_COUNTRIES[subdomain] ?? 'KE';
  // Set country in headers/cookies for downstream API calls
}
```

6. **DNS**: Add wildcard CNAME `*.civicintelligence.com` → Vercel
7. **Vercel**: Supports wildcard domains natively

This is a Phase 12 feature — the architecture already supports it via the country adapter pattern. The platform currently supports 14 countries (KE, UG, TZ, GH, NG, ZA, RW, ZM, SN, EG, MA, CD, ET, MW).

---

## Quick Deploy Checklist

- [ ] Vercel: Import repo (root = repo top, OR root = `apps/web` for frontend-only)
- [ ] Vercel: Set `AI_SERVICE_URL` and `API_SERVICE_URL` env vars
- [ ] Vercel: Confirm the cron schedule in `vercel.json` is `0 3 * * *` (daily, Hobby-compatible)
- [ ] Railway/Render: Deploy Go API from `services/api/`
- [ ] Railway/Render: Deploy Python AI from `services/ai/`
- [ ] Supabase: Create Postgres + pgvector, run 17 migrations
- [ ] Set `DATABASE_URL` on Railway/Render
- [ ] Set `OPENAI_API_KEY` on Railway/Render (Python service)
- [ ] Set `DEV_MODE=false` on Railway/Render (Go service)
- [ ] Optional: Set `MPESA_*` and `STRIPE_SECRET_KEY` for sponsor payments
- [ ] Point your domain to Vercel
- [ ] Verify the cron by manually POSTing to `/api/v1/refresh` after the first deploy

---

## Railway Deployment (Backend — Go API + Python AI)

Railway is the recommended platform for the Go API and Python AI service.

### Step-by-step

1. **Go to [railway.app](https://railway.app)** and sign in with GitHub
2. **New Project** → **Deploy from GitHub repo**
3. Select `Roy-Wanyoike/civic-intelligence`
4. **Create Service 1 — Go API:**
   - **Root Directory:** `services/api`
   - **Build Command:** `go build -o api ./cmd/`
   - **Start Command:** `./api`
   - **Port:** `9000` (Railway auto-detects, or set `PORT=9000`)
   - **Environment Variables:**
     - `DATABASE_URL` — your Postgres connection string (from Supabase)
     - `OPENAI_API_KEY` — for real AI (optional)
     - `DEV_MODE=false` — for production auth
     - `AI_SERVICE_URL` — URL of your Python AI service (from step 5)
5. **Create Service 2 — Python AI:**
   - **Root Directory:** `services/ai`
   - **Build Command:** `pip install -r requirements.txt`
   - **Start Command:** `uvicorn app.main:app --host 0.0.0.0 --port $PORT`
   - **Environment Variables:**
     - `OPENAI_API_KEY` — your OpenAI API key
     - `AI_MODEL_GATEWAY_DEFAULT_PROVIDER=openai` — use real AI
6. **Copy the Railway URLs** back to your Vercel project:
   - `API_SERVICE_URL` = Railway Go API URL (e.g., `https://civic-api.up.railway.app`)
   - `AI_SERVICE_URL` = Railway Python AI URL (e.g., `https://civic-ai.up.railway.app`)
7. **Set up daily refresh:**
   - Vercel Cron is configured in `vercel.json` (runs daily at 03:00 UTC)
   - It calls `/api/v1/refresh` which proxies to the Go API
   - The Go API re-discovers Bills, Hansard, Order Papers, V&P, committee
     reports, and gazette notices from the 12 official Kenyan sources
     (see "Data Sources Crawled" above)

### Alternative: Render

Same approach — two web services on [render.com](https://render.com):
- Go API: https://render.com/docs/deploy-go
- Python AI: https://render.com/docs/deploy-fastapi

### Alternative: Fly.io

Deploy as Docker containers using the existing Dockerfiles:
- Go: `infrastructure/docker/Dockerfile.go`
- Python: `infrastructure/docker/Dockerfile.python-ai`

