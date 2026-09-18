# Engineering Worklog

Each entry is a self-contained record of what an engineer did in a single
sitting, organised by task ID. The file is append-only — never rewrite an
existing entry; if you need to correct one, add a follow-up entry that
references the original.

Convention: `ENG-<letter><number>` matches the task IDs in the issue tracker.
Wave 11 introduced the `K` series for "creative feature" tasks.

---

## ENG-K2 — MP Scorecard + Constituency Dashboard

**Branch:** `feat/wave11-creative-2`
**Worktree:** `/home/z/my-project/wt-creative-2`
**Commit:** `feat: MP Scorecard (factual, no rankings) + Constituency Dashboard`

### What was built

Two creative features that surface the platform's "facts only, no rankings"
editorial posture in concrete UI:

1. **MP Scorecard** — `/people/[id]/scorecard`
   - Backend: `services/api/cmd/scorecard.go` (≈ 400 lines)
     - `GET /api/v1/people/{id}/scorecard` returns the MP's factual record:
       attendance rate (0.0–1.0 float), raw counts for bills sponsored,
       questions asked, votes recorded, statements made, plus lists of bills
       sponsored, committee memberships, and a recent-activity timeline.
     - Every metric carries a `source_url` so a citizen can verify each datum
       against the official parliamentary record (Hansard, parliament.go.ke,
       Votes and Proceedings).
     - **NEVER** calculates a composite "performance score", "MP rating",
       "grade", or "rank". The test suite enforces this invariant
       (`TestScorecard_ReturnsFactualRecord` asserts the absence of
       `score`, `rating`, `performance_score`, `grade`, `rank`,
       `approval_rating` keys).
     - 5 sample MPs seeded (`samplePeople`): Kimani Ichung'wah,
       Opiyo Wandayi, Aaron Cheruiyot, Esther Passaris, Millie Odhiambo.
   - Frontend: `apps/web/src/app/people/[id]/scorecard/page.tsx`
     - Header (name, photo placeholder, role, constituency, party)
     - 5 metric cards (attendance, bills, questions, votes, statements)
       each linking to its source URL
     - Bills sponsored list (links to bill detail + official source)
     - Committee memberships list
     - Recent activity timeline (questions, statements, votes, bill
       publications, committee meetings)
     - DISCLAIMER banner rendered verbatim from the API
     - Print-friendly (`PrintButton` + `print:` Tailwind variants)
     - Share button (Web Share API with clipboard fallback)
   - The existing `/people` list page was rewritten to consume
     `/api/v1/people` and link to the scorecards.

2. **Constituency Dashboard** — `/constituencies/[id]`
   - Backend: `services/api/cmd/constituencies.go` (≈ 540 lines)
     - `GET /api/v1/constituencies?country=KE` — list (with `?q=` search)
     - `GET /api/v1/constituencies/{id}` — full dashboard payload:
       MP info + scorecard link, Bills affecting the area (tagged by
       topic/sector), budget allocated, local projects (from gazette
       notices), public participation opportunities, recent civic events,
       government context (administration + parliamentary term + county
       government + governor), public debt context (with the canonical
       "sovereign debt is national — never personally attributed" note).
     - 10 sample Kenyan constituencies seeded (`sampleConstituencies`):
       Nairobi, Mombasa, Kisumu, Nakuru, Eldoret, Meru, Nyeri, Kakamega,
       Garissa, Turkana.
     - Sovereign debt is **NEVER** attributed personally to any MP,
       governor, or president. The `debt_context.region_note` always
       says "national" — `TestConstituencies_DebtContextNeverAttributesToPerson`
       enforces this.
   - Frontend:
     - `apps/web/src/app/constituencies/page.tsx` — list with URL-driven
       search (debounced via `useTransition` so the URL stays shareable)
     - `apps/web/src/app/constituencies/[id]/page.tsx` — dashboard:
       SVG outline of Kenya with the constituency highlighted
       (`KenyaMapPlaceholder` — explicitly a placeholder, NOT a precise
       district map, with `<title>`/`<desc>` for screen readers)
       - MP info card with link to scorecard
       - Bills Affecting This Area (with topic tags)
       - Budget & Spending section
       - Local Projects section (status colour-coded)
       - Public Participation section (status colour-coded)
       - Recent Events timeline
       - Government Context section
       - Public Debt Context (amber banner with the "national, not personal"
         note and link to `/debt`)
       - Print-friendly

### Supporting changes

- **Navbar**: added "Constituencies" under the "Government" mega-menu group
  (`apps/web/src/components/header.tsx`). Imports `MapPin` from lucide-react.
- **i18n**: added `pages.constituencies` to both English and Kiswahili locale
  files. EN: "Constituencies" · SW: "Jimbo".
- **Frontend API client**: `apps/web/src/lib/people-api.ts` (typed wrappers
  for `listPeople`, `getMPScorecard`, `listConstituencies`, `getConstituency`).
- **Routing**: wired both endpoints in `services/api/cmd/main.go`:
  - `/api/v1/people` (list — no trailing slash) AND `/api/v1/people/` (per
    person + `/scorecard` sub-resource) both dispatch through `handlePeople`.
  - `/api/v1/constituencies` and `/api/v1/constituencies/` registered.

### Pre-existing OIDC verifier build fix

The base branch had a broken OIDC verifier (`services/api/internal/oidc/verifier.go`):
- `tok.Headers` and `tok.Claims(key, &jc)` were called on `*jose.JSONWebSignature`,
  but go-jose/v3.0.3 exposes signatures via `tok.Signatures[i].Header` and
  verification via `tok.Verify(key) ([]byte, error)`. The file even had a
  `FIXME: verify with go build when Go available` comment — confirming it
  had never been compiled.
- Fixed with a minimum-change patch: read the header from `tok.Signatures[0]`,
  call `tok.Verify(key)` then `json.Unmarshal(payload, &jc)`. This unblocks
  the `cmd` package tests (which transitively import `internal/oidc`).

### Test counts

- `services/api`: 216 `=== RUN` lines, 0 failures, 0 build errors. The 11 new
  tests cover the scorecard + constituencies surface:
  - `TestScorecard_ReturnsFactualRecord`
  - `TestScorecard_Returns404ForUnknownPerson`
  - `TestScorecard_AllFiveSamplePeopleHaveScorecards`
  - `TestScorecard_DisclaimerWordingIsStable`
  - `TestPeopleList_ReturnsSamplePeopleWithScorecardLinks`
  - `TestConstituencies_ListReturnsTen`
  - `TestConstituencies_ListSearchFilters`
  - `TestConstituencies_DetailReturnsFullDashboard`
  - `TestConstituencies_DetailReturns404ForUnknown`
  - `TestConstituencies_DebtContextNeverAttributesToPerson`
  - `TestConstituencies_ListIsSorted`
- `apps/web`: `npx tsc --noEmit` exits 0 — no TypeScript errors.

### Files added / modified

**Added (8 files):**
- `services/api/cmd/scorecard.go`
- `services/api/cmd/scorecard_test.go`
- `services/api/cmd/constituencies.go`
- `services/api/cmd/constituencies_test.go`
- `apps/web/src/lib/people-api.ts`
- `apps/web/src/app/people/[id]/scorecard/page.tsx`
- `apps/web/src/app/people/[id]/scorecard/share-button.tsx`
- `apps/web/src/app/constituencies/page.tsx`
- `apps/web/src/app/constituencies/search.tsx`
- `apps/web/src/app/constituencies/[id]/page.tsx`
- `apps/web/src/app/constituencies/[id]/map.tsx`

**Modified (5 files):**
- `services/api/cmd/main.go` — wired people list + scorecard + constituencies routes
- `services/api/internal/oidc/verifier.go` — minimum-change fix for the broken go-jose v3 API
- `apps/web/src/app/people/page.tsx` — replaced hardcoded list with API fetch
- `apps/web/src/components/header.tsx` — added Constituencies nav item under Government group
- `apps/web/src/i18n/locales/en.json` — added `pages.constituencies`
- `apps/web/src/i18n/locales/sw.json` — added `pages.constituencies` (Kiswahili: "Jimbo")

### Editorial posture

The MP Scorecard and Constituency Dashboard are the platform's most explicit
embodiment of the "facts only, no rankings" rule (see also ADR-0005: AI
cannot mutate truth, and `docs/NO_FAKE_COMPLETION.md`). Every numeric metric
has a source URL; no field aggregates raw counts into a composite score; no
view ranks MPs or constituencies; sovereign debt is never attributed
personally to any official. The disclaimers are canonical constants in the
Go source (`scorecardDisclaimer`, `constituencyDisclaimer`) and the test
suite asserts their exact wording — any drift triggers a test failure.
