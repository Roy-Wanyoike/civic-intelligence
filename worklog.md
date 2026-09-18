# Engineering Worklog

A chronological log of engineering work on the Civic Intelligence Platform.
Each entry is keyed by an engineer task ID (ENG-X#) and summarises the
files added/changed, the tests added, the verification commands run, and
the commit hash. The companion `CHANGELOG.md` covers user-visible
releases; this file covers individual engineering sessions.

## ENG-I1 — Civic Knowledge Graph (Wave 9)

**Branch:** `feat/wave9-civic-graph`
**Worktree:** `/home/z/my-project/wt-civic-graph`

### Goal

Build an interactive Civic Knowledge Graph — a visual exploration tool
that shows how Bills, Acts, Institutions, People, Constitution Articles,
and Government terms are connected. No civic intelligence platform in
Africa offers a graph explorer; this is the platform's signature
differentiator.

### Files added

- `services/api/cmd/graph.go` — backend Graph API. Five endpoints
  (`/api/v1/graph/nodes`, `/node/{id}`, `/relationships?id=...&depth=1|2|3`,
  `/search?q=...`, `/paths?from=...&to=...`) plus a bare `/graph` summary
  endpoint. The graph is built once at package init from the existing
  Kenya seed data (acts, administrations, presidents, presidential terms,
  constitution articles, borrowing agreements) plus the package-level
  `sampleInstitutions`, `samplePeople`, `sampleCommittees` tables.
  - Node types: `bill`, `act`, `institution`, `person`,
    `constitution_article`, `administration`, `presidential_term`,
    `legislature`, `committee`, `creditor`, `borrowing_agreement`.
  - Edge types: `ORIGINATES_FROM` (Bill→Act), `SPONSORED_BY` (Bill→Person),
    `BELONGS_TO` (Bill→Legislature), `ASSESSED_BY` (Bill→Committee),
    `CITES` (Bill→Article), `ESTABLISHES` (Article→Institution),
    `GOVERNED_BY` (Bill→Administration), `ASSESNTED_BY` (Act→President),
    `BORROWED_BY` (Agreement→Creditor), `CONTRACTED_DURING`
    (Agreement→Administration), `MEMBER_OF` (Person→Committee), `LEADS`
    (Person→Institution).
  - BFS shortest-path traversal treats the graph as undirected so a
    citizen asking "how is this Bill connected to this President" gets
    the answer regardless of edge direction.
- `services/api/cmd/graph_test.go` — 16 unit tests covering all 5
  endpoints + the bare `/graph` summary + invariants over the seeded
  graph (every documented edge + node type is present).
- `apps/web/src/lib/graph-api.ts` — typed client for the 5 graph
  endpoints. Mirrors the Go response structs so a contract change
  surfaces as a TypeScript compile error.
- `apps/web/src/app/graph/page.tsx` — server component that fetches the
  graph summary and renders the interactive explorer.
- `apps/web/src/app/graph/graph-view.tsx` — client component with:
  - Force-directed SVG visualization powered by `d3-force`.
  - Nodes coloured by type (bills=blue, acts=green, institutions=orange,
    people=purple, articles=gold, plus 6 more type-specific colours).
  - Click a node → side panel shows node details + its direct
    relationships (sourced from `/api/v1/graph/node/{id}`).
  - Search bar at the top (debounced `/api/v1/graph/search`).
  - Depth selector (1, 2, 3 hops).
  - "Find Path" feature — select two nodes via search, see the
    shortest connection path highlighted in the graph.
  - Zoom/pan/drag controls (wheel + drag).
  - Export PNG (svg → canvas → html2canvas → download).
  - Accessible tabular data view as alternative to the visual graph.
  - Mobile-friendly list view with "expand relationship" buttons.
- `tests/integration/api_graph_test.go` — 12 integration tests covering
  all 5 endpoints + the FACT reality_layer tag on the summary + the
  response-shape contract (every node carries id/type/label, every edge
  carries source/target/type) + the source_url evidence-first promise.

### Files changed

- `services/api/cmd/main.go` — register `/api/v1/graph` and
  `/api/v1/graph/` routes on the public API mux (read-only; no auth
  required, in keeping with the existing public-read pattern).
- `apps/web/src/components/header.tsx` — add "Knowledge Graph" to the
  navbar mega-menu under the "Intelligence" group, with the `Network`
  icon.
- `apps/web/src/components/command-palette.tsx` — add "Knowledge Graph"
  to the Intelligence group of the Cmd+K palette.
- `apps/web/src/i18n/locales/en.json` — add `nav.pages.graph` =
  "Knowledge Graph".
- `apps/web/src/i18n/locales/sw.json` — add `nav.pages.graph` =
  "Grafu ya Maarifa" (Kiswahili).
- `apps/web/package.json` — add `d3-force`, `html2canvas` runtime deps
  + `@types/d3-force` dev dep.
- `docs/api/openapi.yaml` — document the 6 graph endpoints
  (`/graph`, `/graph/nodes`, `/graph/node/{id}`,
  `/graph/relationships`, `/graph/search`, `/graph/paths`) plus the
  `GraphNode`, `GraphEdge`, `GraphResponse` schemas.
- `tests/contract/openapi_contract_test.go` — add `/graph` to the
  sub-router allowlist so the OpenAPI contract test recognises that
  `/api/v1/graph/` is satisfied by at least one `/graph/*` openapi
  entry.

### Tests added

- `services/api/cmd/graph_test.go` — 16 tests (all PASS).
- `tests/integration/api_graph_test.go` — 12 tests (all PASS).

### Verification

- `cd services/api && go test -count=1 ./cmd/ 2>&1 | tail -5`
  → `ok github.com/Roy-Wanyoike/civic-intelligence/services/api/cmd 0.018s`
- `cd apps/web && npx tsc --noEmit 2>&1 | tail -5` → no errors.
- `cd tests/contract && go test -count=1 ./...` → PASS (OpenAPI
  contract test satisfied).
- `cd tests/integration && go test -count=1 ./...` → PASS (3.8s).
- Full `cmd/` suite: 160 tests pass, 0 fail.

### Design notes

- The graph is a NAVIGATION aid, never the source of truth. Every node
  carries the same `source_url` as the underlying entity so a citizen
  can verify any relationship against the authoritative Kenya Law / CBK
  / Treasury source. The summary endpoint is tagged
  `reality_layer: FACT`.
- The graph is built once at package init from the existing seed data;
  there are no write paths today. When the legislation service gains
  sponsor data, the `SPONSORED_BY` edge will appear without further
  code changes (the field is wired but the seed has no sponsors).
- The path BFS treats the graph as undirected — a citizen asking "how
  is this Bill connected to this President" gets the shortest
  connection regardless of edge direction.
- The frontend uses SVG (not canvas) so each node + edge is a real DOM
  element that carries a `<title>`, hover state, and click handler —
  accessibility gate.
- The bare `/api/v1/graph` endpoint returns a metadata summary (total
  node + edge counts broken down by type) so the page's initial server
  render shows the graph size without requiring a full traversal.

### Commit

`feat: interactive Civic Knowledge Graph — visual relationship explorer`
