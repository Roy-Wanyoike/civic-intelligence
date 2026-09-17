# Deployment Guide

## Vercel (Frontend)

The Next.js frontend deploys to Vercel.

### Setup

1. Go to [vercel.com/new](https://vercel.com/new)
2. Import `Roy-Wanyoike/civic-intelligence`
3. **Important:** Set **Root Directory** to `apps/web`
4. Vercel auto-detects Next.js
5. Set environment variables (see below)
6. Deploy

### Environment Variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `AI_SERVICE_URL` | No* | `http://localhost:8000` | Python AI service URL |
| `API_SERVICE_URL` | No* | `http://localhost:9000` | Go API service URL |

*If not set, the frontend falls back to mock data. Set these to your backend URLs for real data.

### What deploys to Vercel

- Next.js frontend (36 pages)
- Static assets
- The `apps/web/` directory only

### What does NOT deploy to Vercel

- Go backend (deploy separately)
- Python AI service (deploy separately)
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
```

### Implementation plan

1. **Next.js Middleware** detects the subdomain
2. Sets `country` context (KE, UG, TZ, GH, NG, ZA)
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

This is a Phase 12 feature — the architecture already supports it via the country adapter pattern.

---

## Quick Deploy Checklist

- [ ] Vercel: Import repo, set Root Directory to `apps/web`
- [ ] Vercel: Set `AI_SERVICE_URL` and `API_SERVICE_URL` env vars
- [ ] Railway/Render: Deploy Go API from `services/api/`
- [ ] Railway/Render: Deploy Python AI from `services/ai/`
- [ ] Supabase: Create Postgres + pgvector, run 17 migrations
- [ ] Set `DATABASE_URL` on Railway/Render
- [ ] Set `OPENAI_API_KEY` on Railway/Render (Python service)
- [ ] Set `DEV_MODE=false` on Railway/Render (Go service)
- [ ] Optional: Set `MPESA_*` and `STRIPE_SECRET_KEY` for sponsor payments
- [ ] Point your domain to Vercel
