import Link from 'next/link';
import { notFound } from 'next/navigation';
import {
  ArrowLeft,
  ArrowRight,
  CheckSquare,
  ExternalLink,
  Facebook,
  FileText,
  Landmark,
  Mail,
  MapPin,
  MessageCircle,
  Phone,
  Twitter,
  Users,
} from 'lucide-react';
import type { Metadata } from 'next';
import { getMPScorecard } from '@/lib/people-api';
import { RealityBadge } from '@/components/reality-labels';
import { PrintButton } from '@/components/print-button';
import { ScorecardShareButton } from './share-button';

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>;
}): Promise<Metadata> {
  const { id } = await params;
  let title = 'MP Scorecard';
  try {
    const sc = await getMPScorecard(id);
    title = `${sc.name} — Scorecard`;
  } catch {
    /* ignore */
  }
  return {
    title,
    description:
      'A factual, evidence-backed record of an MP\'s parliamentary activity. Not a ranking — just the facts with sources.',
  };
}

function formatPercent(rate: number): string {
  // rate is 0.0–1.0; format as a percentage with one decimal place.
  return `${(rate * 100).toFixed(1)}%`;
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

const activityKindLabel: Record<string, string> = {
  question: 'Question',
  statement: 'Statement',
  vote: 'Vote',
  bill_sponsored: 'Bill sponsored',
  committee_meeting: 'Committee meeting',
};

const activityKindIcon: Record<string, typeof FileText> = {
  question: MessageCircle,
  statement: FileText,
  vote: CheckSquare,
  bill_sponsored: FileText,
  committee_meeting: Users,
};

export default async function ScorecardPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  let sc: Awaited<ReturnType<typeof getMPScorecard>> | null = null;
  try {
    sc = await getMPScorecard(id);
  } catch {
    notFound();
  }

  return (
    <div className="mx-auto max-w-4xl px-4 py-10 sm:px-6 print:max-w-none print:px-6 print:py-4">
      {/* Back link */}
      <div className="flex items-center justify-between gap-2 print:hidden">
        <Link
          href="/people"
          className="inline-flex items-center gap-1 text-sm text-civic-stone hover:text-civic-forest"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden="true" /> All people
        </Link>
        <div className="flex items-center gap-2">
          <ScorecardShareButton name={sc.name} personId={sc.person_id} />
          <PrintButton />
        </div>
      </div>

      {/* Header */}
      <header className="mt-4 mb-6 border-b border-civic-border pb-5 print:border-civic-ink">
        <div className="flex items-start gap-4">
          {/* Photo placeholder — the platform does NOT fabricate portrait photos. */}
          <div
            aria-hidden="true"
            className="flex h-20 w-20 flex-shrink-0 items-center justify-center rounded-full border border-civic-border bg-civic-mist text-civic-stone"
          >
            <Users className="h-9 w-9" aria-hidden="true" />
          </div>
          <div className="min-w-0 flex-1">
            <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
              <Landmark className="h-4 w-4" aria-hidden="true" />
              <span>MP Scorecard · {sc.parliamentary_period}</span>
              <RealityBadge kind="FACT" size="md" />
            </div>
            <h1 className="mt-1 font-serif text-3xl font-semibold text-civic-forest">
              {sc.name}
            </h1>
            <p className="mt-1 text-sm text-civic-stone">
              {sc.role} · {sc.constituency} · {sc.party}
            </p>
          </div>
        </div>
      </header>

      {/* Disclaimer banner — rendered verbatim from the API */}
      <div
        role="status"
        aria-live="polite"
        className="mb-6 rounded-lg border border-emerald-300 bg-emerald-50 p-4"
      >
        <div className="flex items-start gap-2">
          <RealityBadge kind="FACT" size="md" />
          <p className="text-sm text-emerald-900">
            <strong>Factual records only. Not a ranking.</strong> The platform
            does not rank MPs, score their performance, or imply political
            approval. Every metric below carries a source URL so a citizen can
            verify each datum against the official parliamentary record.
          </p>
        </div>
      </div>

      {/* Metric cards — raw counts + rates only, never a composite score */}
      <section aria-labelledby="metrics-heading" className="mb-8">
        <h2
          id="metrics-heading"
          className="font-serif text-xl font-semibold text-civic-forest"
        >
          Parliamentary record
        </h2>
        <p className="mt-1 text-xs text-civic-stone">
          Raw counts and a single attendance rate — no composite score, no
          weighting. Each metric links to its official source.
        </p>
        <div className="mt-4 grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-5">
          <MetricCard
            label="Attendance rate"
            value={formatPercent(sc.attendance_rate)}
            sourceURL={sc.attendance_source_url}
          />
          <MetricCard
            label="Bills sponsored"
            value={String(sc.bills_sponsored)}
            sourceURL={sc.metrics.bills_sponsored?.source_url}
          />
          <MetricCard
            label="Questions asked"
            value={String(sc.questions_asked)}
            sourceURL={sc.metrics.questions_asked?.source_url}
          />
          <MetricCard
            label="Votes recorded"
            value={String(sc.votes_recorded)}
            sourceURL={sc.metrics.votes_recorded?.source_url}
          />
          <MetricCard
            label="Statements made"
            value={String(sc.statements_made)}
            sourceURL={sc.metrics.statements_made?.source_url}
          />
        </div>
      </section>

      {/* Contact — issue #281. Conditional on at least one non-empty contact
          field so a future MP whose data has not been scraped yet does NOT
          show up with empty "Email: —" / "Phone: —" rows. The _note field
          marks the contact info as seed data pending live scraping from
          parliament.go.ke. */}
      {hasContactInfo(sc) && (
        <section aria-labelledby="contact-heading" className="mb-8">
          <h2
            id="contact-heading"
            className="font-serif text-xl font-semibold text-civic-forest"
          >
            Contact
          </h2>
          <p className="mt-1 text-xs text-civic-stone">
            Official parliamentary contact channels — click any field to
            reach out directly.
          </p>
          <ul className="mt-3 divide-y divide-civic-border overflow-hidden rounded-md border border-civic-border bg-white">
            {sc.email && (
              <ContactRow
                icon={Mail}
                label="Email"
                value={sc.email}
                href={`mailto:${sc.email}`}
              />
            )}
            {sc.phone && (
              <ContactRow
                icon={Phone}
                label="Phone"
                value={sc.phone}
                // tel: links must not contain spaces — strip them.
                href={`tel:${sc.phone.replace(/\s+/g, '')}`}
              />
            )}
            {sc.office_address && (
              <ContactRow
                icon={MapPin}
                label="Office address"
                value={sc.office_address}
              />
            )}
            {sc.twitter && (
              <ContactRow
                icon={Twitter}
                label="Twitter"
                value={sc.twitter}
                href={`https://twitter.com/${sc.twitter.replace(/^@/, '')}`}
                external
              />
            )}
            {sc.facebook && (
              <ContactRow
                icon={Facebook}
                label="Facebook"
                value={sc.facebook}
                href={
                  sc.facebook.startsWith('http')
                    ? sc.facebook
                    : `https://${sc.facebook}`
                }
                external
              />
            )}
          </ul>
          {sc._note && (
            <p className="mt-2 text-[11px] text-civic-stone">
              {sc._note}
            </p>
          )}
        </section>
      )}

      {/* Bills sponsored list */}
      <section aria-labelledby="bills-heading" className="mb-8">
        <h2
          id="bills-heading"
          className="font-serif text-xl font-semibold text-civic-forest"
        >
          Bills sponsored
        </h2>
        <p className="mt-1 text-xs text-civic-stone">
          Each Bill links to its official parliamentary record and to the
          platform&apos;s Bill detail page.
        </p>
        {sc.bills_sponsored_list.length === 0 ? (
          <p className="mt-3 rounded-md border border-civic-border bg-civic-mist p-4 text-sm text-civic-stone">
            No Bills sponsored in this parliamentary period.
          </p>
        ) : (
          <ul className="mt-3 space-y-2">
            {sc.bills_sponsored_list.map((b) => (
              <li
                key={b.url + b.title}
                className="flex items-start justify-between gap-3 rounded-md border border-civic-border bg-white p-3"
              >
                <div className="min-w-0">
                  <Link
                    href={b.url}
                    className="font-medium text-civic-forest hover:underline"
                  >
                    {b.title}
                  </Link>
                  {b.house && (
                    <p className="mt-0.5 text-xs text-civic-stone">{b.house}</p>
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
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* Committee memberships */}
      <section aria-labelledby="committees-heading" className="mb-8">
        <h2
          id="committees-heading"
          className="font-serif text-xl font-semibold text-civic-forest"
        >
          Committee memberships
        </h2>
        <p className="mt-1 text-xs text-civic-stone">
          Committees the MP sits on, with role (Chair, Vice-Chair, Member) —
          sourced from the official parliament portal.
        </p>
        {sc.committee_memberships.length === 0 ? (
          <p className="mt-3 rounded-md border border-civic-border bg-civic-mist p-4 text-sm text-civic-stone">
            No committee memberships in this parliamentary period.
          </p>
        ) : (
          <ul className="mt-3 space-y-2">
            {sc.committee_memberships.map((c) => (
              <li
                key={c.name}
                className="flex items-start justify-between gap-3 rounded-md border border-civic-border bg-white p-3"
              >
                <div>
                  <p className="font-medium text-civic-forest">{c.name}</p>
                  <p className="mt-0.5 text-xs text-civic-stone">
                    Role:{' '}
                    <span className="font-semibold text-civic-ink">{c.role}</span>
                  </p>
                </div>
                <a
                  href={c.source_url}
                  target="_blank"
                  rel="noopener noreferrer"
                  className="inline-flex flex-shrink-0 items-center gap-1 text-xs text-sky-700 hover:underline"
                >
                  Source <ExternalLink className="h-3 w-3" aria-hidden="true" />
                </a>
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* Recent activity timeline */}
      <section aria-labelledby="activity-heading" className="mb-8">
        <h2
          id="activity-heading"
          className="font-serif text-xl font-semibold text-civic-forest"
        >
          Recent activity
        </h2>
        <p className="mt-1 text-xs text-civic-stone">
          Most recent questions, statements, votes and committee meetings —
          each linking to the verifiable Hansard / Votes-and-Proceedings entry.
        </p>
        {sc.recent_activity.length === 0 ? (
          <p className="mt-3 rounded-md border border-civic-border bg-civic-mist p-4 text-sm text-civic-stone">
            No recent activity.
          </p>
        ) : (
          <ol className="mt-4 space-y-3 border-l-2 border-civic-border pl-5">
            {sc.recent_activity.map((a, i) => {
              const Icon = activityKindIcon[a.kind] ?? FileText;
              return (
                <li key={i} className="relative">
                  <span
                    aria-hidden="true"
                    className="absolute -left-[1.4rem] flex h-3 w-3 items-center justify-center rounded-full border-2 border-white bg-civic-forest"
                  />
                  <div className="rounded-md border border-civic-border bg-white p-3">
                    <div className="flex items-start justify-between gap-3">
                      <div className="min-w-0">
                        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
                          <Icon className="h-3.5 w-3.5" aria-hidden="true" />
                          <span>{activityKindLabel[a.kind] ?? a.kind}</span>
                          <span>·</span>
                          <span>{formatDate(a.date)}</span>
                        </div>
                        <p className="mt-1 font-medium text-civic-forest">
                          {a.title}
                        </p>
                        {a.detail && (
                          <p className="mt-0.5 text-xs text-civic-stone">
                            Detail: <span className="font-semibold text-civic-ink">{a.detail}</span>
                          </p>
                        )}
                        <p className="mt-0.5 text-[11px] text-civic-stone">
                          Source: {a.source}
                        </p>
                      </div>
                      <a
                        href={a.source_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex flex-shrink-0 items-center gap-1 text-xs text-sky-700 hover:underline"
                      >
                        Open <ExternalLink className="h-3 w-3" aria-hidden="true" />
                      </a>
                    </div>
                  </div>
                </li>
              );
            })}
          </ol>
        )}
      </section>

      {/* What this page does NOT do */}
      <section className="rounded-lg border border-stone-200 bg-stone-50 p-5 print:hidden">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">
          What this scorecard does NOT do
        </h2>
        <ul className="mt-2 list-disc space-y-1 pl-5 text-sm text-stone-700">
          <li>It does not rank MPs or score their performance.</li>
          <li>It does not aggregate attendance, bills, or votes into a single &quot;approval&quot; number.</li>
          <li>It does not imply political approval or disapproval of any MP.</li>
          <li>It does not infer metrics — every number is sourced from the official parliamentary record.</li>
        </ul>
        <Link
          href="/people"
          className="mt-3 inline-flex items-center gap-1 text-sm font-semibold text-civic-forest hover:underline"
        >
          Browse other MPs <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
        </Link>
      </section>

      <p className="mt-6 text-xs text-civic-stone print:mt-4">
        Source agencies: {sc.source_agencies.join(' · ')}
      </p>
    </div>
  );
}

function MetricCard({
  label,
  value,
  sourceURL,
}: {
  label: string;
  value: string;
  sourceURL?: string;
}) {
  return (
    <div className="rounded-lg border border-civic-border bg-white p-3">
      <p className="text-[11px] font-semibold uppercase tracking-wide text-civic-stone">
        {label}
      </p>
      <p className="mt-1 font-serif text-2xl font-semibold text-civic-forest">
        {value}
      </p>
      {sourceURL && (
        <a
          href={sourceURL}
          target="_blank"
          rel="noopener noreferrer"
          className="mt-1 inline-flex items-center gap-0.5 text-[10px] text-sky-700 hover:underline"
        >
          Source <ExternalLink className="h-2.5 w-2.5" aria-hidden="true" />
        </a>
      )}
    </div>
  );
}

// hasContactInfo returns true when at least one contact field is present,
// so the Contact section is omitted entirely (rather than rendered as an
// empty card) for MPs whose contact info has not been scraped yet.
// Mirrors the API's omitempty invariant (issue #281).
function hasContactInfo(
  sc: Pick<
    Awaited<ReturnType<typeof getMPScorecard>>,
    'email' | 'phone' | 'office_address' | 'twitter' | 'facebook'
  >,
): boolean {
  return Boolean(
    sc.email ||
      sc.phone ||
      sc.office_address ||
      sc.twitter ||
      sc.facebook,
  );
}

// ContactRow is a single row in the Contact section. Each row pairs a
// lucide-react icon with a label + value (optionally a link). The `external`
// flag adds an ExternalLink affordance + the safe `rel="noopener noreferrer"`
// attributes for cross-origin links (Twitter / Facebook profiles).
function ContactRow({
  icon: Icon,
  label,
  value,
  href,
  external,
}: {
  icon: typeof Mail;
  label: string;
  value: string;
  href?: string;
  external?: boolean;
}) {
  return (
    <li className="flex items-start gap-3 p-3">
      <Icon
        className="mt-0.5 h-4 w-4 flex-shrink-0 text-civic-stone"
        aria-hidden="true"
      />
      <div className="min-w-0 flex-1">
        <p className="text-[11px] font-semibold uppercase tracking-wide text-civic-stone">
          {label}
        </p>
        {href ? (
          <a
            href={href}
            {...(external
              ? { target: '_blank', rel: 'noopener noreferrer' }
              : {})}
            className="inline-flex items-center gap-1 break-all font-medium text-civic-forest hover:underline"
          >
            {value}
            {external && (
              <ExternalLink
                className="h-3 w-3 flex-shrink-0"
                aria-hidden="true"
              />
            )}
          </a>
        ) : (
          <p className="break-words font-medium text-civic-forest">{value}</p>
        )}
      </div>
    </li>
  );
}
