import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Developer API',
  description: 'Build on the Civic Intelligence Platform API.',
};

/**
 * Developer API page.
 *
 * Endpoints are grouped to mirror the navbar mega-menu so citizens can move
 * from a feature page to its backing API endpoint without context-switching:
 *
 *   Legislation   — Bills, Acts, Regulations, Policies, Terminology
 *   Government    — Constitution, Governments, Transitions, Institutions, People, Committees
 *   Finance       — Public Debt, Loans, Grants
 *   Intelligence  — Search, Feed, Trending, What Changed, Briefing, Q&A
 *   Trust         — Provenance, Evidence, Claims, Contradictions, Sources, Corrections, Subscriptions, Notifications
 *   Scenarios     — Phase 18 simulation endpoints
 *   Resources     — Health, Metrics, Sponsor, Operational
 *
 * The machine-readable contract is docs/api/openapi.yaml; this page is the
 * human-readable companion.
 */

interface Endpoint {
  method: 'GET' | 'POST' | 'DELETE';
  path: string;
  description: string;
  auth?: 'required' | 'ai:ask' | 'user:admin' | 'evidence:write';
}

interface EndpointGroup {
  title: string;
  description: string;
  endpoints: Endpoint[];
}

const groups: EndpointGroup[] = [
  {
    title: 'Legislation',
    description: 'Bills, Acts, regulations, policies, and parliamentary terms.',
    endpoints: [
      { method: 'GET', path: '/api/v1/bills', description: 'List Bills (filter by status, house, topic, q; paginated)' },
      { method: 'GET', path: '/api/v1/bills/{id}', description: 'Bill detail (sourced from the Kenya Law adapter)' },
      { method: 'GET', path: '/api/v1/bills/{id}/timeline', description: 'Verified timeline events for a Bill' },
      { method: 'GET', path: '/api/v1/bills/{id}/versions', description: 'Immutable Bill versions (newest first)' },
      { method: 'GET', path: '/api/v1/bills/{id}/summary', description: 'AI-generated plain-language summary (validated)' },
      { method: 'GET', path: '/api/v1/bills/{id}/changes', description: 'Change log between Bill versions' },
      { method: 'GET', path: '/api/v1/acts', description: 'Acts of Parliament (issue #116)' },
      { method: 'GET', path: '/api/v1/acts/{id}', description: 'Act detail' },
      { method: 'GET', path: '/api/v1/acts/{id}/audit', description: 'Full lifecycle audit (assent → publication → commencement → regulations → amendments → court → repeal). Missing records are NOT_VERIFIED, never inferred.' },
      { method: 'GET', path: '/api/v1/acts/{id}/events', description: 'Post-assent events (commencement, regulations, amendments, court challenges)' },
      { method: 'POST', path: '/api/v1/acts/{id}/follow', description: 'Follow a Law — creates a real persisted subscription (issue #193 flagship)', auth: 'required' },
      { method: 'GET', path: '/api/v1/acts/{id}/lineage', description: 'Full legal lineage — Bill → Parliamentary journey → Assent → Publication → Commencement → Regulations → Amendments → Court decisions → Current status' },
      { method: 'GET', path: '/api/v1/policies', description: 'Government policy documents' },
      { method: 'GET', path: '/api/v1/terminology', description: 'List all parliamentary terms with plain-language explanations' },
      { method: 'GET', path: '/api/v1/terminology/{term}', description: 'Plain-language explanation of a single parliamentary term' },
      { method: 'GET', path: '/api/v1/trending', description: 'Trending Bills (recently enacted + approaching final stage)' },
    ],
  },
  {
    title: 'Government',
    description: 'The Constitution, administrations, transitions, institutions, people, and committees.',
    endpoints: [
      { method: 'GET', path: '/api/v1/constitution', description: 'Constitution metadata (reality_layer: FACT — never reinterpreted)' },
      { method: 'GET', path: '/api/v1/governments', description: 'List administrations (most recent first)' },
      { method: 'GET', path: '/api/v1/governments/{id}', description: 'Administration detail + president + terms' },
      { method: 'GET', path: '/api/v1/governments/{id}/terms', description: 'Presidential terms for an administration' },
      { method: 'GET', path: '/api/v1/transitions', description: 'Government transitions (outgoing → incoming president)' },
      { method: 'GET', path: '/api/v1/institutions/{id}', description: 'Institution detail' },
      { method: 'GET', path: '/api/v1/people/{id}', description: 'Person detail (MP, senator, official)' },
      { method: 'GET', path: '/api/v1/committees/{id}', description: 'Committee detail' },
    ],
  },
  {
    title: 'Finance',
    description: 'Public debt, sovereign loans, and grants. The platform never attributes borrowing personally to a president.',
    endpoints: [
      { method: 'GET', path: '/api/v1/debt', description: 'National debt dashboard (latest CBK snapshot + debt service + debt-to-GDP)' },
      { method: 'GET', path: '/api/v1/debt/loans', description: 'Borrowing register — each agreement carries an attribution_warning field when its GovernmentAdministrationID does not match the administration in power on ContractDate (issue #221)' },
      { method: 'GET', path: '/api/v1/debt/timeline', description: 'Debt-stock observations (chronological; immutable per Spec §9)' },
      { method: 'GET', path: '/api/v1/debt/governments/{id}', description: 'Per-administration debt summary (carries NO_POLITICAL_PERFORMANCE_SCORE disclaimer)' },
      { method: 'GET', path: '/api/v1/loans', description: 'Sovereign loans tracker (list)' },
      { method: 'GET', path: '/api/v1/loans/{id}', description: 'Sovereign loan detail' },
      { method: 'GET', path: '/api/v1/grants', description: 'Grants received by the government (list)' },
      { method: 'GET', path: '/api/v1/grants/{id}', description: 'Grant detail' },
    ],
  },
  {
    title: 'Intelligence',
    description: 'Search, feeds, trend detection, briefings, and the AI Q&A gateway.',
    endpoints: [
      { method: 'GET', path: '/api/v1/search', description: 'Hybrid keyword + semantic search across Bills, Acts, regulations, Hansard, etc.' },
      { method: 'GET', path: '/api/v1/feed', description: 'Civic activity feed' },
      { method: 'GET', path: '/api/v1/what-changed', description: 'Proactive civic change feed (diffs since the caller\'s last visit)' },
      { method: 'GET', path: '/api/v1/briefing', description: 'Daily civic brief (optional date query param)' },
      { method: 'POST', path: '/api/v1/questions', description: 'Ask a civic question (non-streaming)', auth: 'ai:ask' },
      { method: 'POST', path: '/api/v1/questions/stream', description: 'Ask a civic question (SSE streaming)', auth: 'ai:ask' },
    ],
  },
  {
    title: 'Trust',
    description: 'Provenance, evidence, claims, contradictions, sources, corrections, subscriptions, and notifications.',
    endpoints: [
      { method: 'GET', path: '/api/v1/provenance/{entity_type}/{id}', description: 'Full evidence chain for an entity' },
      { method: 'GET', path: '/api/v1/evidence/{id}', description: 'Evidence detail' },
      { method: 'GET', path: '/api/v1/claims/{id}/evidence', description: 'Get a claim with its supporting evidence' },
      { method: 'GET', path: '/api/v1/contradictions', description: 'List active source conflicts' },
      { method: 'GET', path: '/api/v1/sources', description: 'List trust sources' },
      { method: 'GET', path: '/api/v1/sources/{id}', description: 'Source detail with health' },
      { method: 'POST', path: '/api/v1/corrections', description: 'Submit a correction request (anonymous submitters must provide submitted_email)' },
      { method: 'GET', path: '/api/v1/corrections', description: 'List corrections (admin scope)', auth: 'user:admin' },
      { method: 'GET', path: '/api/v1/corrections/{id}', description: 'Correction detail (admin scope)', auth: 'evidence:write' },
      { method: 'POST', path: '/api/v1/subscriptions', description: 'Follow an entity (bill, committee, topic, institution, person)', auth: 'required' },
      { method: 'GET', path: '/api/v1/subscriptions', description: 'List the caller\'s follows', auth: 'required' },
      { method: 'DELETE', path: '/api/v1/subscriptions/{id}', description: 'Unfollow', auth: 'required' },
      { method: 'GET', path: '/api/v1/notifications', description: 'List the caller\'s notifications (optional unread=true)', auth: 'required' },
      { method: 'POST', path: '/api/v1/notifications/{id}/read', description: 'Mark a notification as read', auth: 'required' },
    ],
  },
  {
    title: 'Scenarios',
    description: 'Phase 18 simulation infrastructure. Every response carries an explicit reality_layer tag (HYPOTHETICAL / MODELED / OBSERVED) so simulated outputs cannot be confused with observed civic facts.',
    endpoints: [
      { method: 'GET', path: '/api/v1/scenarios', description: 'List scenarios' },
      { method: 'POST', path: '/api/v1/scenarios', description: 'Create a scenario' },
      { method: 'GET', path: '/api/v1/scenarios/{id}', description: 'Scenario detail' },
      { method: 'POST', path: '/api/v1/scenarios/{id}?action=validate', description: 'Validate assumptions / constraints' },
      { method: 'POST', path: '/api/v1/scenarios/{id}?action=run', description: 'Run a scenario (deterministic / Monte Carlo) — supports seed for reproducibility' },
      { method: 'POST', path: '/api/v1/scenarios/{id}?action=replay', description: 'Replay the last run' },
      { method: 'GET', path: '/api/v1/scenarios/{id}/assumptions', description: 'Scenario assumptions' },
      { method: 'GET', path: '/api/v1/scenarios/{id}/evidence', description: 'Source evidence + baseline evidence + assumption evidence count' },
      { method: 'GET', path: '/api/v1/scenarios/{id}/results', description: 'Completed run results' },
      { method: 'GET', path: '/api/v1/scenarios/{id}/timeline', description: 'OBSERVED / ASSUMED / MODELED / UNKNOWN timeline' },
      { method: 'GET', path: '/api/v1/scenarios/{id}/methodology', description: 'Methodology + declared model limitations (Gate E)' },
      { method: 'POST', path: '/api/v1/scenarios/compare', description: 'Compare two or more scenarios side by side' },
    ],
  },
  {
    title: 'Resources',
    description: 'Health checks, metrics, sponsor endpoints, and operational routes.',
    endpoints: [
      { method: 'GET', path: '/api/v1/healthz', description: 'Liveness probe' },
      { method: 'GET', path: '/api/v1/readyz', description: 'Readiness probe' },
      { method: 'GET', path: '/metrics', description: 'Prometheus metrics (request counters, latency histograms, OIDC verification counters)' },
      { method: 'POST', path: '/api/v1/sponsor/mpesa', description: 'Initiate an M-Pesa sponsorship' },
      { method: 'POST', path: '/api/v1/sponsor/card', description: 'Initiate a card sponsorship (Stripe)' },
      { method: 'POST', path: '/api/v1/sponsor/mpesa/callback', description: 'M-Pesa STK push callback receiver' },
      { method: 'POST', path: '/api/v1/sponsor/card/webhook', description: 'Stripe webhook receiver' },
      { method: 'POST', path: '/api/v1/refresh', description: 'Trigger adapter re-discovery (called by cron)' },
    ],
  },
];

const methodColors: Record<Endpoint['method'], string> = {
  GET: 'bg-civic-leaf/15 text-civic-leaf',
  POST: 'bg-civic-forest/15 text-civic-forest',
  DELETE: 'bg-red-100 text-red-700',
};

const authBadge: Record<NonNullable<Endpoint['auth']>, string> = {
  required: 'Auth required',
  'ai:ask': 'Scope: ai:ask',
  'user:admin': 'Scope: user:admin',
  'evidence:write': 'Scope: evidence:write',
};

export default function DevelopersPage() {
  const totalEndpoints = groups.reduce((n, g) => n + g.endpoints.length, 0);

  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <h1 className="font-serif text-3xl font-semibold text-civic-forest">Developer API</h1>
      <p className="mt-2 text-sm text-civic-stone">
        Access verified Kenyan civic data through our REST API. {totalEndpoints}+ endpoints
        across {groups.length} feature areas.
      </p>

      <div className="mt-6 rounded-lg border border-civic-border bg-civic-mist/40 p-4 text-sm text-civic-ink">
        <p>
          <span className="font-semibold">Base URL:</span>{' '}
          <code className="rounded bg-civic-paper px-1.5 py-0.5 font-mono text-civic-forest">
            http://localhost:9000/api/v1
          </code>{' '}
          (local dev BFF)
        </p>
        <p className="mt-2">
          <span className="font-semibold">Auth:</span> Bearer JWT (OIDC / Keycloak).
          Public-read endpoints accept anonymous calls; write endpoints return
          401 if the caller is anonymous.
        </p>
        <p className="mt-2">
          <span className="font-semibold">Spec:</span>{' '}
          <code className="rounded bg-civic-paper px-1.5 py-0.5 font-mono text-civic-forest">
            docs/api/openapi.yaml
          </code>{' '}
          is the machine-readable contract.
        </p>
      </div>

      <div className="mt-8 space-y-8">
        {groups.map((group) => (
          <section key={group.title}>
            <h2 className="font-serif text-xl font-semibold text-civic-forest">
              {group.title}
            </h2>
            <p className="mt-1 text-sm text-civic-stone">{group.description}</p>
            <div className="mt-3 overflow-hidden rounded-lg border border-civic-border bg-civic-paper">
              <table className="w-full text-left text-sm">
                <thead className="bg-civic-mist/60 text-xs uppercase tracking-wider text-civic-stone">
                  <tr>
                    <th scope="col" className="px-4 py-2 font-semibold">Method</th>
                    <th scope="col" className="px-4 py-2 font-semibold">Path</th>
                    <th scope="col" className="px-4 py-2 font-semibold">Description</th>
                    <th scope="col" className="px-4 py-2 font-semibold">Auth</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-civic-border">
                  {group.endpoints.map((ep) => (
                    <tr key={`${ep.method} ${ep.path}`} className="align-top">
                      <td className="px-4 py-2">
                        <span
                          className={`inline-block rounded px-2 py-0.5 text-xs font-bold ${methodColors[ep.method]}`}
                        >
                          {ep.method}
                        </span>
                      </td>
                      <td className="px-4 py-2 font-mono text-xs text-civic-ink">
                        {ep.path}
                      </td>
                      <td className="px-4 py-2 text-civic-ink">{ep.description}</td>
                      <td className="px-4 py-2 text-xs text-civic-stone">
                        {ep.auth ? authBadge[ep.auth] : 'Public'}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          </section>
        ))}
      </div>

      <div className="mt-8 rounded-lg border border-civic-border bg-civic-paper p-6">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">Example calls</h2>
        <pre className="mt-3 overflow-x-auto rounded bg-civic-ink p-4 text-xs text-civic-paper">
{`# List in-progress Bills
curl -s 'http://localhost:9000/api/v1/bills?status=in_progress' | jq

# Full lifecycle audit for an Act
curl -s http://localhost:9000/api/v1/acts/act-public-finance-management-2012/audit | jq

# National debt dashboard
curl -s http://localhost:9000/api/v1/debt | jq

# Run a hypothetical scenario (seed for reproducibility)
curl -s -X POST 'http://localhost:9000/api/v1/scenarios/scn-simple-deterministic?action=run&seed=42' | jq

# Follow a Law (requires auth)
curl -s -X POST -H "Authorization: Bearer $JWT" \\
  http://localhost:9000/api/v1/acts/act-public-finance-management-2012/follow | jq`}
        </pre>
      </div>

      <EmbeddableWidgets />

      <div className="mt-4 rounded-lg border border-dashed border-civic-border p-6 text-center text-sm text-civic-stone">
        API keys, rate limiting (300 req/min), and SDKs coming soon.
      </div>
    </div>
  );
}

/**
 * Embeddable Widgets — issue #291.
 *
 * Three chrome-less, iframe-able widgets for third-party civic-tech
 * sites: newsrooms embedding a Bill tracker on an article page, a
 * civil-society org embedding an MP finder on a voter-guide page,
 * or a school embedding today's parliament agenda on a resources page.
 *
 * Each widget is a server-rendered page under /embed/* that:
 *   - renders with NO navbar/footer (the root layout skips the chrome
 *     for /embed/* routes via the `x-civic-embed` middleware marker),
 *   - uses minimal inline CSS scoped to `.civic-embed *` so the host
 *     page's stylesheet can't leak in or out,
 *   - falls back to mock data when the Go BFF is unreachable so the
 *     widget never renders a stack trace inside someone else's page,
 *   - carries a "Powered by Civic Intelligence" link at the bottom.
 *
 * CSP `frame-ancestors *` is set on /embed/* in next.config.mjs so
 * browsers permit embedding from any origin.
 */
function EmbeddableWidgets() {
  const WIDGET_BASE = 'https://civicintel.africa';
  const widgets = [
    {
      name: 'MP Finder',
      path: '/embed/mp-finder',
      width: 400,
      height: 320,
      description:
        'Constituency name → MP name + party + scorecard link. A simple GET form (works without JS).',
      snippet: `<iframe
  src="${WIDGET_BASE}/embed/mp-finder"
  width="400"
  height="320"
  style="border:1px solid #c8d3cc;border-radius:10px"
  loading="lazy"
  title="Civic Intelligence — MP Finder"
></iframe>`,
    },
    {
      name: 'Bill Tracker',
      path: '/embed/bill-tracker',
      width: 480,
      height: 240,
      description:
        "Shows a Bill's current stage on an 8-step horizontal timeline (First Reading → Commencement). Pass ?bill_id=…",
      snippet: `<iframe
  src="${WIDGET_BASE}/embed/bill-tracker?bill_id=00000000-0000-0000-0000-000000000001"
  width="480"
  height="240"
  style="border:1px solid #c8d3cc;border-radius:10px"
  loading="lazy"
  title="Civic Intelligence — Bill Tracker"
></iframe>`,
    },
    {
      name: "Today in Parliament",
      path: '/embed/today',
      width: 400,
      height: 360,
      description:
        "Today's parliament calendar events in a compact list. Refreshes on each load (no caching).",
      snippet: `<iframe
  src="${WIDGET_BASE}/embed/today"
  width="400"
  height="360"
  style="border:1px solid #c8d3cc;border-radius:10px"
  loading="lazy"
  title="Civic Intelligence — Today in Parliament"
></iframe>`,
    },
  ];

  return (
    <section className="mt-8">
      <h2 className="font-serif text-xl font-semibold text-civic-forest">
        Embeddable widgets
      </h2>
      <p className="mt-1 text-sm text-civic-stone">
        Drop these chrome-less iframe widgets into any third-party site —
        newsroom articles, civil-society voter guides, school resource pages.
        Each widget renders with no navbar/footer and minimal inline CSS, so
        the host page&rsquo;s styles can&rsquo;t leak in or out.{' '}
        <code className="rounded bg-civic-paper px-1.5 py-0.5 font-mono text-xs text-civic-forest">
          frame-ancestors *
        </code>{' '}
        is set on <code className="font-mono text-xs">/embed/*</code> routes so
        any origin may embed them.
      </p>

      <div className="mt-4 space-y-6">
        {widgets.map((w) => (
          <div
            key={w.path}
            className="rounded-lg border border-civic-border bg-civic-paper p-5"
          >
            <div className="flex flex-wrap items-baseline justify-between gap-2">
              <h3 className="font-serif text-base font-semibold text-civic-forest">
                {w.name}
              </h3>
              <code className="font-mono text-xs text-civic-stone">{w.path}</code>
            </div>
            <p className="mt-2 text-sm text-civic-ink">{w.description}</p>

            <div className="mt-3 grid gap-4 sm:grid-cols-2">
              <div>
                <p className="mb-1 text-xs uppercase tracking-wider text-civic-stone">
                  Live preview
                </p>
                <iframe
                  src={w.path}
                  width={w.width}
                  height={w.height}
                  style={{ border: '1px solid #c8d3cc', borderRadius: 10 }}
                  loading="lazy"
                  title={`Civic Intelligence — ${w.name}`}
                />
              </div>
              <div>
                <p className="mb-1 text-xs uppercase tracking-wider text-civic-stone">
                  Copy-paste
                </p>
                <pre className="overflow-x-auto rounded bg-civic-ink p-3 text-xs text-civic-paper">
                  {w.snippet}
                </pre>
              </div>
            </div>
          </div>
        ))}
      </div>

      <p className="mt-4 text-xs text-civic-stone">
        Tip: widgets fetch from the same Go BFF as the rest of the platform, so
        they reflect live data. If the BFF is unreachable, each widget falls back
        to a small built-in sample so the iframe never renders a stack trace on
        your page. Replace{' '}
        <code className="rounded bg-civic-paper px-1 py-0.5 font-mono text-xs text-civic-forest">
          civicintel.africa
        </code>{' '}
        with your dev/staging origin to test locally.
      </p>
    </section>
  );
}
