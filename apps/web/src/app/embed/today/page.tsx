import type { Metadata } from 'next';
import type { CalendarEvent } from '@/lib/api';

// Embeddable widgets are always dynamic — they reflect the current
// calendar date. Static-caching /embed/today would freeze "today" at
// build time and lie to embedders about what's on the agenda.
export const dynamic = 'force-dynamic';

export const metadata: Metadata = {
  title: 'Today in Parliament — Civic Intelligence',
  description: 'Embeddable today-in-parliament calendar widget.',
  openGraph: null,
  twitter: null,
};

const API_BASE = process.env.API_BASE_URL ?? 'http://localhost:9000';

async function fetchToday(): Promise<{ events: CalendarEvent[]; source: 'api' | 'mock' }> {
  try {
    const resp = await fetch(`${API_BASE}/api/v1/calendar/today?country=KE`, {
      next: { revalidate: 60 },
    });
    if (!resp.ok) throw new Error(`HTTP ${resp.status}`);
    const data = await resp.json();
    return { events: data.items ?? [], source: 'api' };
  } catch {
    // The mock / sample calendar is a deterministic slice so the
    // widget always renders something legible when the BFF is offline
    // — never a crash, never a stack trace inside someone else's page.
    return { events: MOCK_TODAY, source: 'mock' };
  }
}

interface PageProps {
  searchParams: Promise<{ country?: string }>;
}

export default async function TodayEmbedPage({ searchParams }: PageProps) {
  const { country } = await searchParams;
  const { events, source } = await fetchToday();

  const today = new Date();
  const todayLabel = today.toLocaleDateString('en-KE', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    year: 'numeric',
  });

  return (
    <div className="civic-embed">
      <style>{CSS}</style>

      <div className="head">
        <span className="brand-dot" aria-hidden="true" />
        <span className="head-title">Today in Parliament · {country ?? 'Kenya'}</span>
      </div>

      <h2 className="date">{todayLabel}</h2>

      {events.length === 0 ? (
        <p className="empty">
          No parliament events scheduled today. Check{' '}
          <a href="/calendar" className="link" target="_blank" rel="noopener noreferrer">
            the full calendar
          </a>{' '}
          for upcoming sittings.
        </p>
      ) : (
        <ul className="events" aria-label="Today's parliament events">
          {events.map((ev) => (
            <li key={ev.id} className="event">
              <span className={`event-type type-${ev.type}`} aria-hidden="true">
                {TYPE_LABEL[ev.type] ?? ev.type}
              </span>
              <div className="event-body">
                <p className="event-title">{ev.title}</p>
                {ev.institution && (
                  <p className="event-meta">{ev.institution}</p>
                )}
                {ev.source_url && (
                  <a
                    className="event-source"
                    href={ev.source_url}
                    target="_blank"
                    rel="noopener noreferrer"
                  >
                    Source →
                  </a>
                )}
              </div>
            </li>
          ))}
        </ul>
      )}

      {source === 'mock' && (
        <p className="source-badge mock">Sample schedule — connect the Go API for live events.</p>
      )}

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

// Type labels — short, sentence-case, aligned with the calendar-view's
// filter chips so an embedder who already knows the platform's event
// taxonomy recognises the same vocabulary.
const TYPE_LABEL: Record<string, string> = {
  parliament_session: 'Plenary',
  committee_meeting: 'Committee',
  bill_reading: 'Reading',
  public_participation: 'Public Input',
  gazette_publication: 'Gazette',
  court_hearing: 'Court',
  budget_presentation: 'Budget',
};

// A small, deterministic slice of "today's events" used only as a
// fallback when the Go BFF is unreachable (dev without the API
// running, or a transient outage). Mirrors the Go BFF's in-memory
// sample calendar shape so the widget renders the same way against
// either source.
const MOCK_TODAY: CalendarEvent[] = [
  {
    id: 'today-1',
    title: 'National Assembly — Morning Sitting',
    date: new Date().toISOString().slice(0, 10),
    type: 'parliament_session',
    description: 'Order Paper includes Second Reading of the Housing Bill, 2024.',
    source_url: 'https://parliament.go.ke/order-paper',
    country: 'KE',
    institution: 'National Assembly',
  },
  {
    id: 'today-2',
    title: 'Departmental Committee on Lands — Public Hearing',
    date: new Date().toISOString().slice(0, 10),
    type: 'committee_meeting',
    description: 'Clause-by-clause consideration of the Housing Bill, 2024.',
    source_url: 'https://parliament.go.ke/committees/lands',
    country: 'KE',
    institution: 'Departmental Committee on Lands',
  },
  {
    id: 'today-3',
    title: 'Senate — Afternoon Sitting',
    date: new Date().toISOString().slice(0, 10),
    type: 'parliament_session',
    description: 'Continued debate on the Data Protection (Amendment) Bill, 2024.',
    source_url: 'https://senate.go.ke/order-paper',
    country: 'KE',
    institution: 'Senate',
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
  margin-bottom: 6px;
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
.civic-embed .date {
  margin: 0 0 10px;
  font-size: 15px;
  font-weight: 600;
  color: var(--ink);
}
.civic-embed .events {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.civic-embed .event {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  padding: 8px 10px;
  background: var(--leaf-soft);
  border-radius: 6px;
}
.civic-embed .event-type {
  display: inline-block;
  flex: 0 0 auto;
  padding: 2px 6px;
  border-radius: 4px;
  background: var(--leaf);
  color: #fff;
  font-size: 10px;
  font-weight: 600;
  line-height: 1.2;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  margin-top: 1px;
}
.civic-embed .event-body { flex: 1 1 auto; min-width: 0; }
.civic-embed .event-title {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  color: var(--ink);
  word-wrap: break-word;
}
.civic-embed .event-meta {
  margin: 2px 0 0;
  font-size: 11px;
  color: var(--muted);
}
.civic-embed .event-source {
  display: inline-block;
  margin-top: 4px;
  font-size: 10px;
  color: var(--leaf);
  text-decoration: none;
}
.civic-embed .event-source:hover { text-decoration: underline; }
.civic-embed .empty {
  margin: 0 0 12px;
  font-size: 12px;
  color: var(--muted);
}
.civic-embed .link { color: var(--leaf); text-decoration: none; }
.civic-embed .link:hover { text-decoration: underline; }
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
