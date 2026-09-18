import Link from 'next/link';
import { notFound } from 'next/navigation';
import {
  ArrowLeft,
  ArrowRight,
  CalendarClock,
  CircleDollarSign,
  ExternalLink,
  FileText,
  Landmark,
  MapPin,
  Megaphone,
  TrendingUp,
  Users,
} from 'lucide-react';
import type { Metadata } from 'next';
import { getConstituency } from '@/lib/people-api';
import { RealityBadge } from '@/components/reality-labels';
import { PrintButton } from '@/components/print-button';
import { KenyaMapPlaceholder } from './map';

export const metadata: Metadata = {
  title: 'Constituency Dashboard',
  description:
    'Per-constituency civic dashboard: MP info, bills affecting the area, budget allocated, local projects, public participation opportunities, recent civic events and government context.',
};

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>;
}): Promise<Metadata> {
  const { id } = await params;
  let title = 'Constituency Dashboard';
  try {
    const c = await getConstituency(id);
    title = `${c.name} — Constituency Dashboard`;
  } catch {
    /* ignore */
  }
  return { title };
}

function formatKESm(m: number): string {
  if (m >= 1000) return `KES ${(m / 1000).toFixed(2)}B`;
  return `KES ${m.toFixed(1)}M`;
}

function formatKESb(b: number): string {
  return `KES ${b.toFixed(1)}B`;
}

function formatDate(d?: string): string {
  if (!d) return '—';
  try {
    return new Date(d).toLocaleDateString('en-KE', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  } catch {
    return d;
  }
}

const eventKindLabel: Record<string, string> = {
  bill_published: 'Bill published',
  gazette_notice: 'Gazette notice',
  committee_meeting: 'Committee meeting',
  public_participation: 'Public participation',
  budget_allocation: 'Budget allocation',
};

const projectStatusColor: Record<string, string> = {
  ongoing: 'bg-amber-100 text-amber-800',
  completed: 'bg-emerald-100 text-emerald-800',
  stalled: 'bg-rose-100 text-rose-800',
};

const participationStatusColor: Record<string, string> = {
  upcoming: 'bg-emerald-100 text-emerald-800',
  held: 'bg-stone-200 text-stone-700',
  cancelled: 'bg-rose-100 text-rose-800',
};

export default async function ConstituencyDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  let c: Awaited<ReturnType<typeof getConstituency>> | null = null;
  try {
    c = await getConstituency(id);
  } catch {
    notFound();
  }

  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6 print:max-w-none print:px-6 print:py-4">
      <div className="flex items-center justify-between gap-2 print:hidden">
        <Link
          href="/constituencies"
          className="inline-flex items-center gap-1 text-sm text-civic-stone hover:text-civic-forest"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden="true" /> All constituencies
        </Link>
        <PrintButton />
      </div>

      {/* Header */}
      <header className="mt-4 mb-6 border-b border-civic-border pb-5 print:border-civic-ink">
        <div className="flex flex-col gap-4 sm:flex-row sm:items-start">
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
              <MapPin className="h-4 w-4" aria-hidden="true" />
              <span>Constituency · {c.country}</span>
              <RealityBadge kind="FACT" size="md" />
            </div>
            <h1 className="mt-1 font-serif text-3xl font-semibold text-civic-forest">
              {c.name}
            </h1>
            <p className="mt-1 text-sm text-civic-stone">
              {c.county} County · {c.region}
            </p>
            <dl className="mt-3 grid grid-cols-2 gap-3 text-xs sm:grid-cols-3">
              <div>
                <dt className="font-semibold text-civic-stone">Population</dt>
                <dd className="inline-flex items-center gap-1 text-civic-ink">
                  <Users className="h-3 w-3" aria-hidden="true" />
                  {c.population.toLocaleString('en-KE')}
                </dd>
              </div>
              <div>
                <dt className="font-semibold text-civic-stone">Area</dt>
                <dd className="text-civic-ink">{c.area_km2.toLocaleString('en-KE')} km²</dd>
              </div>
              <div>
                <dt className="font-semibold text-civic-stone">Density</dt>
                <dd className="text-civic-ink">
                  {c.area_km2 > 0
                    ? `${(c.population / c.area_km2).toFixed(1)} /km²`
                    : '—'}
                </dd>
              </div>
            </dl>
          </div>
          {/* SVG outline of Kenya with the constituency highlighted */}
          <KenyaMapPlaceholder constituencyName={c.name} />
        </div>
      </header>

      {/* MP info card */}
      <section aria-labelledby="mp-heading" className="mb-8">
        <h2
          id="mp-heading"
          className="font-serif text-xl font-semibold text-civic-forest"
        >
          MP for this area
        </h2>
        <div className="mt-3 rounded-lg border border-civic-border bg-white p-4">
          <div className="flex items-start justify-between gap-3">
            <div className="min-w-0">
              <p className="font-serif text-lg font-semibold text-civic-forest">
                {c.mp.name}
              </p>
              <p className="mt-0.5 text-xs text-civic-stone">
                {c.mp.role} · {c.mp.party}
              </p>
              <Link
                href={`/people/${c.mp.person_id}/scorecard`}
                className="mt-3 inline-flex items-center gap-1 rounded-md bg-civic-forest px-3 py-1.5 text-xs font-semibold text-civic-paper hover:bg-civic-leaf"
              >
                Open MP scorecard <ArrowRight className="h-3 w-3" aria-hidden="true" />
              </Link>
            </div>
            <a
              href={c.mp.source_url}
              target="_blank"
              rel="noopener noreferrer"
              className="inline-flex flex-shrink-0 items-center gap-1 text-xs text-sky-700 hover:underline"
            >
              Source <ExternalLink className="h-3 w-3" aria-hidden="true" />
            </a>
          </div>
        </div>
      </section>

      {/* Bills Affecting This Area */}
      <section aria-labelledby="bills-heading" className="mb-8">
        <h2
          id="bills-heading"
          className="font-serif text-xl font-semibold text-civic-forest"
        >
          Bills affecting this area
        </h2>
        <p className="mt-1 text-xs text-civic-stone">
          Bills tagged by topic / sector that affect this constituency. Each
          Bill links to the platform&apos;s Bill detail page and to the official
          parliamentary record.
        </p>
        {c.bills_affecting_area.length === 0 ? (
          <p className="mt-3 rounded-md border border-civic-border bg-civic-mist p-4 text-sm text-civic-stone">
            No Bills tagged for this area in the current parliamentary period.
          </p>
        ) : (
          <ul className="mt-3 space-y-2">
            {c.bills_affecting_area.map((b) => (
              <li
                key={b.bill_id}
                className="rounded-md border border-civic-border bg-white p-3"
              >
                <div className="flex items-start justify-between gap-3">
                  <div className="min-w-0">
                    <Link
                      href={b.url}
                      className="font-medium text-civic-forest hover:underline"
                    >
                      {b.title}
                    </Link>
                    <p className="mt-0.5 text-xs text-civic-stone">
                      {b.house} · <span className="text-civic-ink">{b.tag}</span>
                    </p>
                    {b.topics.length > 0 && (
                      <ul className="mt-1 flex flex-wrap gap-1">
                        {b.topics.map((t) => (
                          <li
                            key={t}
                            className="rounded-full bg-civic-mist px-2 py-0.5 text-[10px] uppercase tracking-wide text-civic-stone"
                          >
                            {t}
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>
                  <a
                    href={b.source_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex flex-shrink-0 items-center gap-1 text-xs text-sky-700 hover:underline"
                  >
                    Source <ExternalLink className="h-3 w-3" aria-hidden="true" />
                  </a>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* Budget & Spending */}
      <section aria-labelledby="budget-heading" className="mb-8">
        <h2
          id="budget-heading"
          className="font-serif text-xl font-semibold text-civic-forest"
        >
          Budget &amp; spending
        </h2>
        <p className="mt-1 text-xs text-civic-stone">
          Allocations affecting this constituency for the current fiscal year.
          Each line carries the gazette notice or budget statement URL so a
          citizen can verify the figure against the official record.
        </p>
        {c.budget_allocated.length === 0 ? (
          <p className="mt-3 rounded-md border border-civic-border bg-civic-mist p-4 text-sm text-civic-stone">
            No budget allocations published yet.
          </p>
        ) : (
          <ul className="mt-3 space-y-2">
            {c.budget_allocated.map((b) => (
              <li
                key={b.fiscal_year + b.vote}
                className="flex items-start justify-between gap-3 rounded-md border border-civic-border bg-white p-3"
              >
                <div className="min-w-0">
                  <div className="flex items-center gap-1.5 text-xs text-civic-stone">
                    <CircleDollarSign className="h-3.5 w-3.5" aria-hidden="true" />
                    <span>FY {b.fiscal_year}</span>
                    <span>·</span>
                    <span>{b.source}</span>
                  </div>
                  <p className="mt-1 font-medium text-civic-forest">{b.vote}</p>
                  <a
                    href={b.source_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="mt-0.5 inline-flex items-center gap-1 text-[11px] text-sky-700 hover:underline"
                  >
                    Source <ExternalLink className="h-2.5 w-2.5" aria-hidden="true" />
                  </a>
                </div>
                <p className="font-serif text-lg font-semibold text-civic-forest">
                  {formatKESm(b.amount_kes_millions)}
                </p>
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* Local Projects */}
      <section aria-labelledby="projects-heading" className="mb-8">
        <h2
          id="projects-heading"
          className="font-serif text-xl font-semibold text-civic-forest"
        >
          Local projects
        </h2>
        <p className="mt-1 text-xs text-civic-stone">
          Projects gazetted for the area. Each project carries the gazette
          notice URL so a citizen can verify the project exists in the official
          record (not a campaign promise).
        </p>
        {c.local_projects.length === 0 ? (
          <p className="mt-3 rounded-md border border-civic-border bg-civic-mist p-4 text-sm text-civic-stone">
            No gazetted local projects yet.
          </p>
        ) : (
          <ul className="mt-3 grid gap-2 sm:grid-cols-2">
            {c.local_projects.map((p) => (
              <li
                key={p.name}
                className="rounded-md border border-civic-border bg-white p-3"
              >
                <div className="flex items-start justify-between gap-2">
                  <div className="min-w-0">
                    <p className="font-medium text-civic-forest">{p.name}</p>
                    <p className="mt-0.5 text-xs text-civic-stone">
                      Sector: {p.sector} · {formatKESm(p.allocated_kes_millions)}
                    </p>
                  </div>
                  <span
                    className={`flex-shrink-0 rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${projectStatusColor[p.status] ?? 'bg-stone-100 text-stone-700'}`}
                  >
                    {p.status}
                  </span>
                </div>
                <p className="mt-1 text-[11px] text-civic-stone">
                  Gazette: {formatDate(p.gazette_date)}
                </p>
                <a
                  href={p.source_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="mt-0.5 inline-flex items-center gap-1 text-[11px] text-sky-700 hover:underline"
                >
                  Source <ExternalLink className="h-2.5 w-2.5" aria-hidden="true" />
                </a>
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* Public Participation */}
      <section aria-labelledby="participation-heading" className="mb-8">
        <h2
          id="participation-heading"
          className="font-serif text-xl font-semibold text-civic-forest"
        >
          Public participation
        </h2>
        <p className="mt-1 text-xs text-civic-stone">
          Upcoming (and recent) public participation forums relevant to the
          area. Each entry links to the official organiser page so a citizen can
          confirm the time, location and submission channels.
        </p>
        {c.public_participation.length === 0 ? (
          <p className="mt-3 rounded-md border border-civic-border bg-civic-mist p-4 text-sm text-civic-stone">
            No upcoming public participation forums.
          </p>
        ) : (
          <ul className="mt-3 space-y-2">
            {c.public_participation.map((p) => (
              <li
                key={p.title}
                className="flex items-start justify-between gap-3 rounded-md border border-civic-border bg-white p-3"
              >
                <div className="min-w-0">
                  <div className="flex items-center gap-1.5 text-xs text-civic-stone">
                    <Megaphone className="h-3.5 w-3.5" aria-hidden="true" />
                    <span>{p.organiser}</span>
                  </div>
                  <p className="mt-1 font-medium text-civic-forest">{p.title}</p>
                  <p className="mt-0.5 text-xs text-civic-stone">
                    {formatDate(p.date)} · {p.location}
                  </p>
                  <a
                    href={p.source_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="mt-0.5 inline-flex items-center gap-1 text-[11px] text-sky-700 hover:underline"
                  >
                    Source <ExternalLink className="h-2.5 w-2.5" aria-hidden="true" />
                  </a>
                </div>
                <span
                  className={`flex-shrink-0 rounded-full px-2 py-0.5 text-[10px] font-semibold uppercase tracking-wide ${participationStatusColor[p.status] ?? 'bg-stone-100 text-stone-700'}`}
                >
                  {p.status}
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* Recent Events */}
      <section aria-labelledby="events-heading" className="mb-8">
        <h2
          id="events-heading"
          className="font-serif text-xl font-semibold text-civic-forest"
        >
          Recent events
        </h2>
        <p className="mt-1 text-xs text-civic-stone">
          Timeline of civic events in this constituency: Bills published,
          gazette notices, committee meetings and budget allocations.
        </p>
        {c.recent_events.length === 0 ? (
          <p className="mt-3 rounded-md border border-civic-border bg-civic-mist p-4 text-sm text-civic-stone">
            No recent events.
          </p>
        ) : (
          <ol className="mt-4 space-y-3 border-l-2 border-civic-border pl-5">
            {c.recent_events.map((e, i) => (
              <li key={i} className="relative">
                <span
                  aria-hidden="true"
                  className="absolute -left-[1.4rem] flex h-3 w-3 items-center justify-center rounded-full border-2 border-white bg-civic-forest"
                />
                <div className="rounded-md border border-civic-border bg-white p-3">
                  <div className="flex items-start justify-between gap-3">
                    <div className="min-w-0">
                      <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
                        <FileText className="h-3.5 w-3.5" aria-hidden="true" />
                        <span>{eventKindLabel[e.kind] ?? e.kind}</span>
                        <span>·</span>
                        <span>{formatDate(e.date)}</span>
                      </div>
                      <p className="mt-1 font-medium text-civic-forest">{e.title}</p>
                      {e.detail && (
                        <p className="mt-0.5 text-xs text-civic-stone">
                          Detail: <span className="font-semibold text-civic-ink">{e.detail}</span>
                        </p>
                      )}
                      <p className="mt-0.5 text-[11px] text-civic-stone">
                        Source: {e.source}
                      </p>
                    </div>
                    <a
                      href={e.source_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex flex-shrink-0 items-center gap-1 text-xs text-sky-700 hover:underline"
                    >
                      Open <ExternalLink className="h-3 w-3" aria-hidden="true" />
                    </a>
                  </div>
                </div>
              </li>
            ))}
          </ol>
        )}
      </section>

      {/* Government context */}
      <section aria-labelledby="govt-heading" className="mb-8">
        <h2
          id="govt-heading"
          className="font-serif text-xl font-semibold text-civic-forest"
        >
          Government context
        </h2>
        <p className="mt-1 text-xs text-civic-stone">
          Where this constituency sits in the wider government structure:
          administration, parliamentary term, county government.
        </p>
        <div className="mt-3 rounded-lg border border-civic-border bg-white p-4">
          <dl className="grid gap-3 text-sm sm:grid-cols-2">
            <div>
              <dt className="text-xs font-semibold text-civic-stone">National administration</dt>
              <dd className="inline-flex items-center gap-1.5 text-civic-ink">
                <Landmark className="h-3.5 w-3.5" aria-hidden="true" />
                <Link
                  href={`/governments/${c.government_context.administration_id}`}
                  className="hover:underline"
                >
                  {c.government_context.administration_name}
                </Link>
              </dd>
            </div>
            <div>
              <dt className="text-xs font-semibold text-civic-stone">Parliamentary term</dt>
              <dd className="inline-flex items-center gap-1.5 text-civic-ink">
                <CalendarClock className="h-3.5 w-3.5" aria-hidden="true" />
                {c.government_context.parliamentary_term}
              </dd>
            </div>
            <div>
              <dt className="text-xs font-semibold text-civic-stone">County government</dt>
              <dd className="text-civic-ink">{c.government_context.county_government}</dd>
            </div>
            <div>
              <dt className="text-xs font-semibold text-civic-stone">Governor</dt>
              <dd className="text-civic-ink">{c.government_context.governor}</dd>
            </div>
          </dl>
          <a
            href={c.government_context.source_url}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-3 inline-flex items-center gap-1 text-xs text-sky-700 hover:underline"
          >
            County government source <ExternalLink className="h-3 w-3" aria-hidden="true" />
          </a>
        </div>
      </section>

      {/* Public debt context */}
      <section aria-labelledby="debt-heading" className="mb-8">
        <h2
          id="debt-heading"
          className="font-serif text-xl font-semibold text-civic-forest"
        >
          Public debt context
        </h2>
        <div className="mt-3 rounded-lg border border-amber-300 bg-amber-50 p-4">
          <div className="flex items-start gap-2">
            <TrendingUp className="mt-0.5 h-4 w-4 flex-shrink-0 text-amber-700" aria-hidden="true" />
            <div>
              <p className="text-sm text-amber-900">{c.debt_context.region_note}</p>
              <dl className="mt-3 grid gap-2 text-xs sm:grid-cols-3">
                <div>
                  <dt className="font-semibold text-amber-800">National debt</dt>
                  <dd className="text-amber-900">
                    {formatKESb(c.debt_context.national_debt_kes_billions)}
                  </dd>
                </div>
                <div>
                  <dt className="font-semibold text-amber-800">As of</dt>
                  <dd className="text-amber-900">{formatDate(c.debt_context.as_of_date)}</dd>
                </div>
                <div>
                  <dt className="font-semibold text-amber-800">Source</dt>
                  <dd className="text-amber-900">{c.debt_context.source}</dd>
                </div>
              </dl>
              <a
                href={c.debt_context.source_url}
                target="_blank"
                rel="noopener noreferrer"
                className="mt-2 inline-flex items-center gap-1 text-xs text-amber-800 hover:underline"
              >
                Verify on CBK <ExternalLink className="h-3 w-3" aria-hidden="true" />
              </a>
              <Link
                href="/debt"
                className="ml-3 inline-flex items-center gap-1 text-xs font-semibold text-amber-900 hover:underline"
              >
                Open national debt dashboard <ArrowRight className="h-3 w-3" aria-hidden="true" />
              </Link>
            </div>
          </div>
        </div>
      </section>

      {/* What this page does NOT do */}
      <section className="rounded-lg border border-stone-200 bg-stone-50 p-5 print:hidden">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">
          What this dashboard does NOT do
        </h2>
        <ul className="mt-2 list-disc space-y-1 pl-5 text-sm text-stone-700">
          <li>It does not rank constituencies or score development performance.</li>
          <li>It does not attribute sovereign borrowing personally to any official.</li>
          <li>It does not infer projects — every project comes from a gazette notice.</li>
          <li>It does not aggregate budget lines into a single &quot;funding score&quot;.</li>
        </ul>
      </section>
    </div>
  );
}
