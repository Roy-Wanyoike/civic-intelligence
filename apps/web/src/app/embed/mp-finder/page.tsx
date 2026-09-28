import type { Metadata } from 'next';

// Embeddable widgets are always dynamic — they reflect live legislative
// state from the Go BFF, which re-discovers constituencies from the
// IEBC + parliament.go.ke. Static-caching /embed/mp-finder would freeze
// the MP roster at build time.
export const dynamic = 'force-dynamic';

export const metadata: Metadata = {
  title: 'MP Finder — Civic Intelligence',
  description: 'Embeddable constituency → MP scorecard lookup widget.',
  openGraph: null,
  twitter: null,
};

const API_BASE = process.env.API_BASE_URL ?? 'http://localhost:9000';

interface ConstituencyListItem {
  id: string;
  name: string;
  county: string;
  country: string;
  region: string;
  mp_name: string;
  mp_party: string;
  population: number;
  dashboard_url: string;
}

interface ConstituencyDetail {
  mp: {
    person_id: string;
    name: string;
    role: string;
    party: string;
    scorecard_url: string;
    source_url: string;
  };
}

async function searchConstituencies(
  q: string,
): Promise<{ items: ConstituencyListItem[]; source: 'api' | 'mock' }> {
  try {
    const url = `${API_BASE}/api/v1/constituencies?q=${encodeURIComponent(q)}&country=KE`;
    const resp = await fetch(url, { next: { revalidate: 60 } });
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const data = await resp.json();
    return { items: data.items ?? [], source: 'api' };
  } catch {
    // Mock fallback — case-insensitive substring match on the small
    // built-in roster so the widget always renders *something* useful
    // when the BFF is offline. Matches the platform's editorial rule:
    // a Constituency is a real civic entity, so the mock list is a
    // handful of well-known Nairobi constituencies with their actual
    // sitting MPs (data is public — parliament.go.ke is the source).
    const needle = q.toLowerCase();
    const items = MOCK_CONSTITUENCIES.filter(
      (c) =>
        c.name.toLowerCase().includes(needle) ||
        c.county.toLowerCase().includes(needle),
    );
    return { items, source: 'mock' };
  }
}

async function fetchConstituencyDetail(
  id: string,
): Promise<ConstituencyDetail | null> {
  try {
    const resp = await fetch(
      `${API_BASE}/api/v1/constituencies/${encodeURIComponent(id)}`,
      { next: { revalidate: 60 } },
    );
    if (!resp.ok) return null;
    return (await resp.json()) as ConstituencyDetail;
  } catch {
    return null;
  }
}

interface PageProps {
  searchParams: Promise<{ q?: string }>;
}

export default async function MPFinderEmbedPage({ searchParams }: PageProps) {
  const { q } = await searchParams;
  const query = (q ?? '').trim();

  return (
    <div className="civic-embed">
      <style>{CSS}</style>

      <div className="head">
        <span className="brand-dot" aria-hidden="true" />
        <span className="head-title">Find your MP</span>
      </div>

      {/* GET form so the widget works with zero client-side JS — the
          iframe simply reloads with ?q=<constituency> on submit. */}
      <form method="GET" action="/embed/mp-finder" className="form">
        <label htmlFor="q" className="label">
          Constituency name
        </label>
        <div className="input-row">
          <input
            id="q"
            name="q"
            type="search"
            defaultValue={query}
            placeholder="e.g. Westlands, Dagoretti North, Kibra"
            className="input"
            autoComplete="off"
          />
          <button type="submit" className="btn">
            Find
          </button>
        </div>
      </form>

      <div className="results">
        {query === '' ? (
          <p className="muted">
            Enter your constituency name to see your MP, their party, and a link to
            their scorecard.
          </p>
        ) : (
          <Results query={query} />
        )}
      </div>

      <a
        className="powered-by"
        href="https://civicintel.africa"
        target="_blank"
        rel="noopener noreferrer"
      >
        Powered by Civic Intelligence
      </a>
    </div>
  );
}

async function Results({ query }: { query: string }) {
  const { items, source } = await searchConstituencies(query);

  if (items.length === 0) {
    return (
      <p className="muted">
        No constituency matched <code>{query}</code>. Try a different spelling
        or the county name.
      </p>
    );
  }

  // Show the first match as the canonical answer (the BFF sorts by
  // relevance server-side). Also surface the next few alternates so a
  // misspelled query can be self-corrected without a round-trip.
  const [first, ...rest] = items;
  const detail = await fetchConstituencyDetail(first.id);
  const scorecardUrl = detail?.mp?.scorecard_url ?? first.dashboard_url;

  return (
    <>
      <article className="match">
        <p className="match-constituency">
          {first.name}
          <span className="match-county"> · {first.county} County</span>
        </p>
        <p className="match-mp">
          <span className="mp-label">MP</span>{' '}
          <strong>{first.mp_name || detail?.mp?.name || 'Unknown'}</strong>
        </p>
        <p className="match-party">
          <span className="mp-label">Party</span>{' '}
          {first.mp_party || detail?.mp?.party || '—'}
        </p>
        <a
          className="cta"
          href={scorecardUrl}
          target="_blank"
          rel="noopener noreferrer"
        >
          View MP scorecard →
        </a>
      </article>

      {rest.length > 0 && (
        <details className="alternates">
          <summary>
            {rest.length} more match{rest.length === 1 ? '' : 'es'}
          </summary>
          <ul className="alt-list">
            {rest.slice(0, 5).map((c) => (
              <li key={c.id}>
                <a className="alt-link" href={c.dashboard_url} target="_blank" rel="noopener noreferrer">
                  {c.name} — {c.mp_name} ({c.mp_party})
                </a>
              </li>
            ))}
          </ul>
        </details>
      )}

      {source === 'mock' && (
        <p className="source-badge mock">
          Sample data — connect the Go API for the verified MP roster.
        </p>
      )}
    </>
  );
}

// A small, deterministic roster of well-known constituencies used as a
// fallback when the BFF is offline. Names + sitting MPs are public
// record (parliament.go.ke); the mock is a safe-by-default legible
// render so the widget never shows a stack trace inside someone else's
// page.
const MOCK_CONSTITUENCIES: ConstituencyListItem[] = [
  {
    id: 'const-westlands',
    name: 'Westlands',
    county: 'Nairobi',
    country: 'KE',
    region: 'Nairobi',
    mp_name: 'Tim Wanyonyi',
    mp_party: 'ODM',
    population: 0,
    dashboard_url: '/constituencies/const-westlands',
  },
  {
    id: 'const-dagoretti-north',
    name: 'Dagoretti North',
    county: 'Nairobi',
    country: 'KE',
    region: 'Nairobi',
    mp_name: 'Simba Arati',
    mp_party: 'ODM',
    population: 0,
    dashboard_url: '/constituencies/const-dagoretti-north',
  },
  {
    id: 'const-kibra',
    name: 'Kibra',
    county: 'Nairobi',
    country: 'KE',
    region: 'Nairobi',
    mp_name: 'Raphael Owino',
    mp_party: 'ODM',
    population: 0,
    dashboard_url: '/constituencies/const-kibra',
  },
  {
    id: 'const-langata',
    name: 'Langata',
    county: 'Nairobi',
    country: 'KE',
    region: 'Nairobi',
    mp_name: 'Phelix Odiwuor',
    mp_party: 'ODM',
    population: 0,
    dashboard_url: '/constituencies/const-langata',
  },
  {
    id: 'const-mathare',
    name: 'Mathare',
    county: 'Nairobi',
    country: 'KE',
    region: 'Nairobi',
    mp_name: 'Anthony Oluoch',
    mp_party: 'ODM',
    population: 0,
    dashboard_url: '/constituencies/const-mathare',
  },
];

const CSS = `
.civic-embed {
  --bg: #ffffff;
  --ink: #0b1f17;
  --muted: #52606b;
  --border: #c8d3cc;
  --leaf: #15784a;
  --leaf-soft: rgba(21, 120, 74, 0.12);
  --acacia: #d4a017;
  --shadow: rgba(11, 31, 23, 0.04);
  font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto,
    'Helvetica Neue', Arial, 'Noto Sans', sans-serif;
  background: var(--bg);
  color: var(--ink);
  padding: 14px 16px;
  border: 1px solid var(--border);
  border-radius: 10px;
  box-shadow: 0 1px 2px var(--shadow);
  max-width: 100%;
  box-sizing: border-box;
  font-size: 13px;
  line-height: 1.45;
}
.civic-embed * { box-sizing: border-box; }
.civic-embed .head {
  display: flex;
  align-items: center;
  gap: 6px;
  margin-bottom: 8px;
}
.civic-embed .brand-dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--leaf);
}
.civic-embed .head-title {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--muted);
}
.civic-embed .form { margin: 0 0 12px; }
.civic-embed .label {
  display: block;
  font-size: 11px;
  color: var(--muted);
  margin-bottom: 4px;
}
.civic-embed .input-row { display: flex; gap: 6px; }
.civic-embed .input {
  flex: 1 1 auto;
  min-width: 0;
  padding: 7px 10px;
  border: 1px solid var(--border);
  border-radius: 6px;
  font-size: 13px;
  color: var(--ink);
  background: #fff;
  outline: none;
}
.civic-embed .input:focus { border-color: var(--leaf); box-shadow: 0 0 0 3px var(--leaf-soft); }
.civic-embed .btn {
  flex: 0 0 auto;
  padding: 7px 14px;
  border: none;
  border-radius: 6px;
  background: var(--leaf);
  color: #fff;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}
.civic-embed .btn:hover { filter: brightness(1.05); }
.civic-embed .results { min-height: 60px; }
.civic-embed .muted { color: var(--muted); margin: 0; font-size: 12px; }
.civic-embed code {
  background: rgba(82, 96, 107, 0.08);
  padding: 1px 4px;
  border-radius: 3px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
}
.civic-embed .match {
  background: var(--leaf-soft);
  border-radius: 8px;
  padding: 10px 12px;
  margin: 0 0 8px;
}
.civic-embed .match-constituency {
  margin: 0 0 6px;
  font-size: 12px;
  color: var(--muted);
}
.civic-embed .match-county { color: var(--muted); }
.civic-embed .match-mp,
.civic-embed .match-party {
  margin: 0 0 4px;
  font-size: 13px;
}
.civic-embed .mp-label {
  display: inline-block;
  width: 48px;
  font-size: 10px;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: var(--muted);
}
.civic-embed .cta {
  display: inline-block;
  margin-top: 6px;
  padding: 6px 10px;
  border-radius: 6px;
  background: var(--leaf);
  color: #fff;
  text-decoration: none;
  font-size: 12px;
  font-weight: 600;
}
.civic-embed .cta:hover { filter: brightness(1.05); }
.civic-embed .alternates { margin: 6px 0 0; font-size: 11px; color: var(--muted); }
.civic-embed .alternates summary {
  cursor: pointer;
  color: var(--leaf);
  font-size: 11px;
  padding: 2px 0;
}
.civic-embed .alt-list { list-style: none; margin: 6px 0 0; padding: 0; }
.civic-embed .alt-list li { margin: 0 0 2px; }
.civic-embed .alt-link {
  color: var(--leaf);
  text-decoration: none;
  font-size: 11px;
}
.civic-embed .alt-link:hover { text-decoration: underline; }
.civic-embed .source-badge.mock {
  display: block;
  margin: 8px 0 0;
  padding: 4px 8px;
  background: rgba(212, 160, 23, 0.1);
  color: var(--acacia);
  border-radius: 4px;
  font-size: 10px;
}
.civic-embed .powered-by {
  display: block;
  margin-top: 12px;
  padding-top: 8px;
  border-top: 1px solid var(--border);
  font-size: 10px;
  color: var(--muted);
  text-decoration: none;
  text-align: right;
}
.civic-embed .powered-by:hover { color: var(--leaf); }
`;
