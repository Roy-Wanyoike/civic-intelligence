import type { Metadata } from 'next';
import { mockBills, mockTimeline } from '@/lib/mock-data';
import type { Bill, BillEvent } from '@/lib/types';

// Embeddable widgets are always dynamic — they reflect live legislative
// state from the Go BFF, which re-discovers Bills from kenyalaw.org on a
// cron. Static-caching /embed/bill-tracker?bill_id=… would freeze the
// Bill's current stage at build time and lie to embedders.
export const dynamic = 'force-dynamic';

export const metadata: Metadata = {
  title: 'Bill Tracker — Civic Intelligence',
  description: 'Embeddable Bill stage timeline widget.',
  // Embeddable widgets are loaded inside third-party <iframe>s, so we
  // opt out of OpenGraph / Twitter card metadata — those only matter
  // when the page is the top-level document, which it never is.
  openGraph: null,
  twitter: null,
};

const API_BASE = process.env.API_BASE_URL ?? 'http://localhost:9000';

/**
 * The canonical Kenyan Bill pipeline, ordered earliest → latest. The
 * `current_stage` field on a Bill is one of these (or a variant
 * spelling the Go BFF emits, e.g. "Committee Stage" vs "Committee
 * Consideration"); we map both spellings onto the same ordered list
 * so the horizontal tracker can show "you are here" relative to the
 * full pipeline. (The list intentionally omits "Withdrawn" and
 * "Negatived" — those are terminal failure states, not stages.)
 */
const STAGES: Array<{ key: string; label: string; short: string }> = [
  { key: 'first_reading', label: 'First Reading', short: '1st' },
  { key: 'second_reading', label: 'Second Reading', short: '2nd' },
  { key: 'committee', label: 'Committee Stage', short: 'Cmte' },
  { key: 'report', label: 'Report Stage', short: 'Rpt' },
  { key: 'third_reading', label: 'Third Reading', short: '3rd' },
  { key: 'assent', label: 'Assent', short: 'Assent' },
  { key: 'publication', label: 'Publication', short: 'Pub' },
  { key: 'commencement', label: 'Commencement', short: 'Cmce' },
];

/**
 * Map a Bill's free-text `current_stage` (as returned by the Go BFF)
 * onto one of the canonical STAGES keys. Returns the 0-based index
 * into STAGES (or `null` if we can't recognise the stage) so the
 * tracker can mark every stage before that index as ✅ done and every
 * stage after as ⏳ pending.
 */
function stageIndex(currentStage: string | null | undefined): number | null {
  if (!currentStage) return null;
  const s = currentStage.toLowerCase();
  if (s.includes('first') || s.includes('1st')) return 0;
  if (s.includes('second') || s.includes('2nd')) return 1;
  if (s.includes('committee')) return 2;
  if (s.includes('report')) return 3;
  if (s.includes('third') || s.includes('3rd')) return 4;
  if (s.includes('assent')) return 5;
  if (s.includes('publication')) return 6;
  if (s.includes('commencement') || s.includes('operation')) return 7;
  // Enacted / in force — treat as the final stage.
  if (s.includes('enacted') || s.includes('in_force') || s.includes('commenced')) return 7;
  return null;
}

async function fetchBill(billId: string): Promise<{ bill: Bill | null; source: 'api' | 'mock' }> {
  try {
    const resp = await fetch(
      `${API_BASE}/api/v1/bills/${encodeURIComponent(billId)}`,
      { next: { revalidate: 60 } },
    );
    if (!resp.ok) return { bill: null, source: 'api' };
    return { bill: (await resp.json()) as Bill, source: 'api' };
  } catch {
    const bill = mockBills.find((b) => b.id === billId) ?? null;
    return { bill, source: 'mock' };
  }
}

async function fetchTimeline(billId: string): Promise<{ events: BillEvent[]; source: 'api' | 'mock' }> {
  try {
    const resp = await fetch(
      `${API_BASE}/api/v1/bills/${encodeURIComponent(billId)}/timeline`,
      { next: { revalidate: 60 } },
    );
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const data = await resp.json();
    return { events: data.events ?? [], source: 'api' };
  } catch {
    return {
      events: mockTimeline.filter((e) => e.bill_id === billId),
      source: 'mock',
    };
  }
}

interface PageProps {
  searchParams: Promise<{ bill_id?: string }>;
}

export default async function BillTrackerEmbedPage({ searchParams }: PageProps) {
  const { bill_id: billId } = await searchParams;

  // No bill_id query param — render a friendly "how to use" state
  // instead of a 400. Embedders will see this on first paint before
  // they wire up the bill_id attribute on their iframe.
  if (!billId) {
    return (
      <Widget>
        <Head title="Bill Tracker" />
        <p className="muted">
          Add a <code>?bill_id=…</code> query parameter to track a specific Bill.
        </p>
        <p className="muted">
          Example:{' '}
          <a
            href="/embed/bill-tracker?bill_id=00000000-0000-0000-0000-000000000001"
            className="link"
          >
            /embed/bill-tracker?bill_id=00000000-0000-0000-0000-000000000001
          </a>
        </p>
        <PoweredBy />
      </Widget>
    );
  }

  const { bill, source: billSource } = await fetchBill(billId);
  const fallbackBill = mockBills.find((b) => b.id === billId) ?? null;
  const rendered = bill ?? fallbackBill;

  if (!rendered) {
    return (
      <Widget>
        <Head title="Bill Tracker" />
        <p className="muted">
          Bill <code>{billId}</code> was not found.
        </p>
        <PoweredBy />
      </Widget>
    );
  }

  // Only fetch the timeline if the live BFF gave us a Bill OR the mock
  // fallback has events for this ID — otherwise we'd 500 on every call.
  const { events } = await fetchTimeline(rendered.id);

  const currentIdx = stageIndex(rendered.current_stage);
  const lastEvent = events[0] ?? null;
  const stageCount = currentIdx === null ? 0 : currentIdx + 1;

  return (
    <Widget>
      <Head title="Bill Tracker" />

      <h2 className="bill-title">{rendered.title}</h2>
      <p className="bill-meta">
        {rendered.identifier} · {rendered.house_name ?? rendered.house ?? 'Parliament'}
        {rendered.current_stage ? ` · ${rendered.current_stage}` : ''}
      </p>

      <ol className="timeline" aria-label={`Bill stages — currently at ${rendered.current_stage ?? 'unknown'}`}>
        {STAGES.map((stage, i) => {
          const done = currentIdx !== null && i < currentIdx;
          const current = currentIdx === i;
          return (
            <li
              key={stage.key}
              className={`stage${done ? ' done' : ''}${current ? ' current' : ''}`}
              aria-current={current ? 'step' : undefined}
            >
              <span className="dot" aria-hidden="true" />
              <span className="stage-label">{stage.label}</span>
            </li>
          );
        })}
      </ol>

      <p className="progress">
        {currentIdx === null
          ? 'Stage could not be determined from available sources.'
          : `${stageCount} of ${STAGES.length} stages completed.`}
      </p>

      {lastEvent && (
        <p className="last-event">
          Latest:{' '}
          <time dateTime={lastEvent.date ?? undefined}>
            {lastEvent.date ? new Date(lastEvent.date).toLocaleDateString('en-KE', {
              day: 'numeric',
              month: 'short',
              year: 'numeric',
            }) : '—'}
          </time>{' '}
          — {lastEvent.description}
        </p>
      )}

      <a
        className="cta"
        href={rendered.source_url ?? `/bills/${encodeURIComponent(rendered.id)}`}
        target="_blank"
        rel="noopener noreferrer"
      >
        View full Bill →
      </a>

      {billSource === 'mock' && (
        <p className="source-badge mock">Sample data — connect the Go API for verified events.</p>
      )}

      <PoweredBy />
    </Widget>
  );
}

// ----- Shared widget chrome (kept local so each embed page is a
// single self-contained file with no cross-widget imports). -----

function Widget({ children }: { children: React.ReactNode }) {
  return (
    <div className="civic-embed">
      <style>{CSS}</style>
      {children}
    </div>
  );
}

function Head({ title }: { title: string }) {
  return (
    <div className="head">
      <span className="brand-dot" aria-hidden="true" />
      <span className="head-title">{title}</span>
    </div>
  );
}

function PoweredBy() {
  return (
    <a
      className="powered-by"
      href="https://civicintel.africa"
      target="_blank"
      rel="noopener noreferrer"
    >
      Powered by Civic Intelligence
    </a>
  );
}

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
.civic-embed .bill-title {
  margin: 0 0 4px;
  font-size: 15px;
  font-weight: 600;
  color: var(--ink);
}
.civic-embed .bill-meta {
  margin: 0 0 10px;
  font-size: 11px;
  color: var(--muted);
}
.civic-embed .timeline {
  list-style: none;
  margin: 0 0 8px;
  padding: 0;
  display: flex;
  align-items: stretch;
  gap: 0;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
}
.civic-embed .stage {
  flex: 1 1 0;
  min-width: 44px;
  position: relative;
  padding-top: 14px;
  text-align: center;
}
.civic-embed .stage::before {
  content: '';
  position: absolute;
  top: 4px;
  left: 50%;
  width: 100%;
  height: 2px;
  background: var(--border);
  transform: translateX(-50%);
  z-index: 0;
}
.civic-embed .stage:first-child::before { left: 50%; width: 50%; }
.civic-embed .stage:last-child::before { width: 50%; }
.civic-embed .stage .dot {
  position: absolute;
  top: 0;
  left: 50%;
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--bg);
  border: 2px solid var(--border);
  transform: translateX(-50%);
  z-index: 1;
}
.civic-embed .stage.done .dot {
  background: var(--leaf);
  border-color: var(--leaf);
}
.civic-embed .stage.current .dot {
  background: var(--leaf);
  border-color: var(--leaf);
  box-shadow: 0 0 0 4px var(--leaf-soft);
}
.civic-embed .stage.done::before,
.civic-embed .stage.current::before {
  background: var(--leaf);
}
.civic-embed .stage-label {
  display: block;
  font-size: 10px;
  color: var(--muted);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.civic-embed .stage.current .stage-label {
  color: var(--ink);
  font-weight: 600;
}
.civic-embed .progress {
  margin: 0 0 6px;
  font-size: 11px;
  color: var(--muted);
}
.civic-embed .last-event {
  margin: 0 0 10px;
  font-size: 11px;
  color: var(--ink);
}
.civic-embed .cta {
  display: inline-block;
  padding: 6px 10px;
  border-radius: 6px;
  background: var(--leaf);
  color: #fff;
  text-decoration: none;
  font-size: 12px;
  font-weight: 600;
}
.civic-embed .cta:hover { filter: brightness(1.05); }
.civic-embed .muted { color: var(--muted); margin: 0 0 6px; font-size: 12px; }
.civic-embed .link { color: var(--leaf); text-decoration: none; }
.civic-embed .link:hover { text-decoration: underline; }
.civic-embed code {
  background: rgba(82, 96, 107, 0.08);
  padding: 1px 4px;
  border-radius: 3px;
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 11px;
}
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
