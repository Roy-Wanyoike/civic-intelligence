import Link from 'next/link';
import { notFound } from 'next/navigation';
import {
  AlertTriangle,
  ArrowLeft,
  ArrowRight,
  CheckSquare,
  Clock,
  ExternalLink,
  Facebook,
  FileText,
  Landmark,
  Mail,
  MapPin,
  MessageCircle,
<<<<<<< HEAD
  MessageSquareText,
=======
  MinusCircle,
>>>>>>> origin/main
  Phone,
  ThumbsDown,
  ThumbsUp,
  Twitter,
  Users,
  XCircle,
} from 'lucide-react';
import type { Metadata } from 'next';
<<<<<<< HEAD
import {
  getMPScorecard,
<<<<<<< HEAD
  getMPWrittenQuestions,
  type WrittenQuestionStatus,
=======
  getRegisteredInterests,
  type RegisteredInterest,
  type RegisteredInterestCategory,
>>>>>>> origin/main
} from '@/lib/people-api';
=======
import { getMPScorecard, getMPVotes, type VoteKind } from '@/lib/people-api';
>>>>>>> origin/main
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

// formatKES renders a KES integer with the Kenyan grouping (e.g.
// "KES 2,500,000"). Returns "Value not disclosed" when value is 0 — the
// API uses 0 to mean "value not disclosed / not applicable" (e.g. a
// non-monetary gift or a state commendation medallion) and the
// description text carries the qualifier in that case (issue #288).
function formatKES(kes: number): string {
  if (!kes) return 'Value not disclosed';
  return `KES ${kes.toLocaleString('en-KE')}`;
}

// interestCategoryConfig maps each declaration category to a label +
// Tailwind colour triplet so the badge on each interest card is
// visually distinct (Gate K — UI Clarity). The colour families are
// chosen so adjacent categories on the same card never collide.
const interestCategoryConfig: Record<
  RegisteredInterestCategory,
  { label: string; badge: string }
> = {
  directorship: {
    label: 'Directorship',
    badge: 'bg-indigo-50 text-indigo-800 border-indigo-200',
  },
  land_property: {
    label: 'Land & property',
    badge: 'bg-emerald-50 text-emerald-800 border-emerald-200',
  },
  shares: {
    label: 'Shares',
    badge: 'bg-sky-50 text-sky-800 border-sky-200',
  },
  gifts: {
    label: 'Gifts',
    badge: 'bg-amber-50 text-amber-800 border-amber-200',
  },
  other_income: {
    label: 'Other income',
    badge: 'bg-violet-50 text-violet-800 border-violet-200',
  },
  loans: {
    label: 'Loans',
    badge: 'bg-rose-50 text-rose-800 border-rose-200',
  },
};

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

<<<<<<< HEAD
// writtenQuestionStyling maps each WrittenQuestionStatus to its label
// + icon + tailwind classes for the per-row badge in the Written
// Questions section (issue #286). The colour code is:
//   - answered → green  (CheckSquare)        — Minister has responded
//   - pending  → amber  (Clock)              — tabled, awaiting response
//   - overdue  → red    (AlertTriangle)      — deadline passed, no response
// The platform NEVER ranks these — the colour coding is purely a
// readability affordance, not a verdict on the Minister or the MP.
// (rule: NO_POLITICAL_PERFORMANCE_SCORE).
const writtenQuestionStyling: Record<
  WrittenQuestionStatus,
  { label: string; icon: typeof CheckSquare; badge: string }
> = {
  answered: {
    label: 'Answered',
    icon: CheckSquare,
    badge: 'bg-emerald-100 text-emerald-800 border-emerald-200',
  },
  pending: {
    label: 'Pending',
    icon: Clock,
    badge: 'bg-amber-100 text-amber-800 border-amber-200',
  },
  overdue: {
    label: 'Overdue',
    icon: AlertTriangle,
    badge: 'bg-rose-100 text-rose-800 border-rose-200',
  },
=======
// voteStyling maps each VoteKind to its label + icon + tailwind classes
// for the per-row badge in the Voting Record section (issue #284).
// The colour code is:
//   - aye     → green (ThumbsUp)       — voted in favour
//   - nay     → red   (ThumbsDown)     — voted against
//   - abstain → gray  (MinusCircle)    — present but abstained
//   - absent  → gray  (XCircle)         — not present for the division
// abstain + absent share the gray palette (both "did not vote aye or
// nay") but the icon + label keep them visually distinct so a citizen
// can still tell them apart. The platform NEVER ranks these — the
// colour coding is purely a readability affordance, not a verdict.
const voteStyling: Record<
  VoteKind,
  { label: string; icon: typeof ThumbsUp; badge: string }
> = {
  aye: {
    label: 'Aye',
    icon: ThumbsUp,
    badge: 'bg-emerald-100 text-emerald-800 border-emerald-200',
  },
  nay: {
    label: 'Nay',
    icon: ThumbsDown,
    badge: 'bg-rose-100 text-rose-800 border-rose-200',
  },
  abstain: {
    label: 'Abstain',
    icon: MinusCircle,
    badge: 'bg-stone-100 text-stone-700 border-stone-200',
  },
  absent: {
    label: 'Absent',
    icon: XCircle,
    badge: 'bg-stone-100 text-stone-700 border-stone-200',
  },
>>>>>>> origin/main
};

export default async function ScorecardPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
<<<<<<< HEAD
  // Fetch the scorecard + written questions in parallel. The scorecard
  // is the page's primary data — if it fails, we 404. The written
  // questions fetch is best-effort: if it fails (e.g. transient
  // upstream hiccup), the page still renders the rest of the
  // scorecard and the Written Questions section falls back to its
  // empty state (issue #286 graceful-degradation contract).
  let sc: Awaited<ReturnType<typeof getMPScorecard>> | null = null;
  let writtenQuestions:
    | Awaited<ReturnType<typeof getMPWrittenQuestions>>
    | null = null;
  try {
    [sc, writtenQuestions] = await Promise.all([
      getMPScorecard(id),
      getMPWrittenQuestions(id).catch(() => null),
=======
  // Fetch the scorecard + votes in parallel. The scorecard is the page's
  // primary data — if it fails, we 404. The votes fetch is best-effort:
  // if it fails (e.g. transient upstream hiccup), the page still renders
  // the rest of the scorecard and the Voting Record section falls back
  // to its empty state (issue #284 graceful-degradation contract).
  let sc: Awaited<ReturnType<typeof getMPScorecard>> | null = null;
<<<<<<< HEAD
  // Registered interests are fetched in parallel with the scorecard
  // (issue #288). A failure to load interests does NOT 404 the whole
  // page — the scorecard is the primary resource and interests are a
  // secondary tab. An empty items list (e.g. for a future MP whose
  // declaration has not been seeded yet) is rendered as "No declared
  // interests" rather than omitting the section, so citizens can see
  // the platform tracks declarations even when none are on record.
  let interests: RegisteredInterest[] = [];
  let interestsDisclaimer = '';
  try {
    sc = await getMPScorecard(id);
    const interestsResp = await getRegisteredInterests(id);
    interests = interestsResp.items ?? [];
    interestsDisclaimer = interestsResp.disclaimer;
=======
  let votes: Awaited<ReturnType<typeof getMPVotes>> | null = null;
  try {
    [sc, votes] = await Promise.all([
      getMPScorecard(id),
      getMPVotes(id).catch(() => null),
>>>>>>> origin/main
    ]);
>>>>>>> origin/main
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

<<<<<<< HEAD
      {/* Written questions — issue #286. MP → Minister written
          questions tabled in this parliamentary period, colour-coded
          by lifecycle status: green=answered, amber=pending,
          red=overdue. The colour coding is a readability affordance
          ONLY — the platform NEVER ranks these or derives a
          "responsiveness score" (rule: NO_POLITICAL_PERFORMANCE_SCORE).
          Each row links to the official parliament.go.ke record. */}
      <section aria-labelledby="written-questions-heading" className="mb-8">
        <h2
          id="written-questions-heading"
          className="font-serif text-xl font-semibold text-civic-forest"
        >
          Written questions
        </h2>
        <p className="mt-1 text-xs text-civic-stone">
          Questions tabled to Cabinet Secretaries + Ministers, with
          lifecycle status (answered / pending / overdue) and the
          response text when published. Each entry links to the official
          parliamentary record.
        </p>
        {!writtenQuestions || writtenQuestions.items.length === 0 ? (
          <p className="mt-3 rounded-md border border-civic-border bg-civic-mist p-4 text-sm text-civic-stone">
            No written questions tabled in this parliamentary period.
          </p>
        ) : (
          <ul className="mt-3 space-y-3">
            {writtenQuestions.items.map((q, i) => {
              const styling =
                writtenQuestionStyling[q.status] ??
                writtenQuestionStyling.pending;
              const StatusIcon = styling.icon;
              return (
                <li
                  key={`${q.id}-${i}`}
                  className="rounded-md border border-civic-border bg-white p-4"
                >
                  {/* Status badge + minister + ministry */}
                  <div className="flex flex-wrap items-center gap-2">
                    <span
                      className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-semibold ${styling.badge}`}
                    >
                      <StatusIcon className="h-3 w-3" aria-hidden="true" />
                      {styling.label}
                    </span>
                    <span className="inline-flex items-center gap-1 text-xs text-civic-stone">
                      <MessageSquareText
                        className="h-3.5 w-3.5"
                        aria-hidden="true"
                      />
                      <span className="font-medium text-civic-ink">
                        {q.minister}
                      </span>
                      <span aria-hidden="true">·</span>
                      <span>{q.ministry}</span>
                    </span>
                  </div>

                  {/* Question text */}
                  <p className="mt-2 text-sm leading-relaxed text-civic-forest">
                    {q.question_text}
                  </p>

                  {/* Metadata row — asked date + deadline + responded date */}
                  <dl className="mt-2 flex flex-wrap gap-x-4 gap-y-1 text-[11px] text-civic-stone">
                    <div className="flex items-center gap-1">
                      <dt className="font-semibold uppercase tracking-wide">
                        Asked:
                      </dt>
                      <dd>{formatDate(q.asked_at)}</dd>
                    </div>
                    <div className="flex items-center gap-1">
                      <dt className="font-semibold uppercase tracking-wide">
                        Deadline:
                      </dt>
                      <dd>{formatDate(q.deadline)}</dd>
                    </div>
                    {q.responded_at && (
                      <div className="flex items-center gap-1">
                        <dt className="font-semibold uppercase tracking-wide">
                          Response date:
                        </dt>
                        <dd>{formatDate(q.responded_at)}</dd>
                      </div>
                    )}
                  </dl>

                  {/* Response text — only rendered when the Minister has
                      published a response (status=answered). */}
                  {q.response_text && (
                    <div className="mt-3 rounded-md border border-emerald-200 bg-emerald-50 p-3">
                      <p className="text-[11px] font-semibold uppercase tracking-wide text-emerald-800">
                        Minister&apos;s response
                      </p>
                      <p className="mt-1 text-sm leading-relaxed text-emerald-900">
                        {q.response_text}
                      </p>
                    </div>
                  )}

                  {/* Source link */}
                  <a
                    href={q.source_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="mt-3 inline-flex items-center gap-1 text-xs text-sky-700 hover:underline"
                  >
                    Source <ExternalLink className="h-3 w-3" aria-hidden="true" />
                  </a>
=======
      {/* Registered interests — issue #288. Each declared interest renders
          as a card with a category badge, the description (verbatim from
          the Declaration of Interests Register), the KES value (or
          "Value not disclosed" when 0), the declaration date, and a
          source link to the canonical register entry on parliament.go.ke.
          The platform does NOT yet scrape the live register — the
          disclaimer below makes the seed-only provenance explicit so the
          declarations are never silently shipped as authoritative. */}
      <section aria-labelledby="interests-heading" className="mb-8">
        <h2
          id="interests-heading"
          className="font-serif text-xl font-semibold text-civic-forest"
        >
          Registered interests
        </h2>
        <p className="mt-1 text-xs text-civic-stone">
          Directorships, land + property, shareholdings, gifts, other income
          and loans declared to the Clerk of the Senate / National Assembly
          under the Leadership and Integrity Act, 2012. Every entry links to
          its canonical register entry so a citizen can spot conflicts of
          interest before an MP votes on a Bill or sits in a committee
          inquiry that touches one of their declared holdings.
        </p>
        {interests.length === 0 ? (
          <p className="mt-3 rounded-md border border-civic-border bg-civic-mist p-4 text-sm text-civic-stone">
            No declared interests on record in this parliamentary period.
          </p>
        ) : (
          <ul className="mt-3 space-y-2">
            {interests.map((ri, i) => {
              const cfg = interestCategoryConfig[ri.category] ?? {
                label: ri.category,
                badge:
                  'bg-stone-100 text-stone-700 border-stone-300',
              };
              return (
                <li
                  key={`${ri.category}-${ri.declared_at}-${i}`}
                  className="rounded-md border border-civic-border bg-white p-3"
                >
                  <div className="flex flex-wrap items-start justify-between gap-3">
                    <div className="min-w-0 flex-1">
                      <div className="flex flex-wrap items-center gap-2">
                        <span
                          className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-semibold ${cfg.badge}`}
                        >
                          {cfg.label}
                        </span>
                        <span className="text-[11px] uppercase tracking-wide text-civic-stone">
                          Declared {formatDate(ri.declared_at)}
                        </span>
                      </div>
                      <p className="mt-1.5 text-sm text-civic-ink">
                        {ri.description}
                      </p>
                      <p className="mt-1 text-xs text-civic-stone">
                        Value:{' '}
                        <span className="font-semibold text-civic-ink">
                          {formatKES(ri.value_kes)}
                        </span>
                      </p>
                    </div>
                    <a
                      href={ri.source_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex flex-shrink-0 items-center gap-1 text-xs text-sky-700 hover:underline"
                    >
                      Source <ExternalLink className="h-3 w-3" aria-hidden="true" />
                    </a>
                  </div>
>>>>>>> origin/main
                </li>
              );
            })}
          </ul>
        )}
<<<<<<< HEAD
=======
        {interestsDisclaimer && (
          <p className="mt-2 text-[11px] text-civic-stone">
            {interestsDisclaimer}
          </p>
        )}
>>>>>>> origin/main
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

      {/* Voting record — issue #284. Per-division votes the MP cast in
          this parliamentary period, color-coded: green=aye, red=nay,
          gray=abstain/absent. The colour coding is a readability
          affordance ONLY — the platform NEVER ranks these. Each row
          links to the official Votes-and-Proceedings entry. */}
      <section aria-labelledby="votes-heading" className="mb-8">
        <h2
          id="votes-heading"
          className="font-serif text-xl font-semibold text-civic-forest"
        >
          Voting record
        </h2>
        <p className="mt-1 text-xs text-civic-stone">
          Per-division votes cast in this parliamentary period — color-coded
          for readability only. Each entry links to the official
          Votes-and-Proceedings record.
        </p>
        {!votes || votes.items.length === 0 ? (
          <p className="mt-3 rounded-md border border-civic-border bg-civic-mist p-4 text-sm text-civic-stone">
            No recorded votes in this parliamentary period.
          </p>
        ) : (
          <ul className="mt-3 space-y-2">
            {votes.items.map((v, i) => {
              const styling = voteStyling[v.vote] ?? voteStyling.absent;
              const VoteIcon = styling.icon;
              return (
                <li
                  key={`${v.bill_id}-${v.person_id}-${i}`}
                  className="flex items-start justify-between gap-3 rounded-md border border-civic-border bg-white p-3"
                >
                  <div className="min-w-0">
                    <div className="flex flex-wrap items-center gap-2">
                      <span
                        className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-semibold ${styling.badge}`}
                      >
                        <VoteIcon className="h-3 w-3" aria-hidden="true" />
                        {styling.label}
                      </span>
                      <span className="text-xs text-civic-stone">
                        {formatDate(v.date)} · {v.division}
                      </span>
                    </div>
                    <p className="mt-1 font-medium text-civic-forest">
                      {v.bill_title}
                    </p>
                    <p className="mt-0.5 text-[11px] text-civic-stone">
                      Source: Votes and Proceedings
                    </p>
                  </div>
                  <a
                    href={v.source_url}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="inline-flex flex-shrink-0 items-center gap-1 text-xs text-sky-700 hover:underline"
                  >
                    Source <ExternalLink className="h-3 w-3" aria-hidden="true" />
                  </a>
                </li>
              );
            })}
          </ul>
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
