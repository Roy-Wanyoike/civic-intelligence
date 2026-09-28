import Link from 'next/link';
import { notFound } from 'next/navigation';
import {
  AlertTriangle,
  ArrowLeft,
  CheckSquare,
  Clock,
  ExternalLink,
  Facebook,
  FileText,
  Mail,
  MapPin,
  MessageSquareText,
  MinusCircle,
  Phone,
  ThumbsDown,
  ThumbsUp,
  Twitter,
  Users,
  XCircle,
} from 'lucide-react';
import type { Metadata } from 'next';
import {
  getMPScorecard,
  getMPVotes,
  getMPWrittenQuestions,
  getRegisteredInterests,
  type VoteKind,
  type VoteRecord,
  type WrittenQuestion,
  type WrittenQuestionStatus,
  type RegisteredInterest,
  type RegisteredInterestCategory,
} from '@/lib/people-api';
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

const categoryLabels: Record<string, string> = {
  directorship: 'Directorship',
  land_property: 'Land & Property',
  shares: 'Shares & Securities',
  gifts: 'Gifts',
  other_income: 'Other Income',
  loans: 'Loans',
};

const categoryColours: Record<string, string> = {
  directorship: 'bg-blue-100 text-blue-800 border-blue-200',
  land_property: 'bg-amber-100 text-amber-800 border-amber-200',
  shares: 'bg-purple-100 text-purple-800 border-purple-200',
  gifts: 'bg-pink-100 text-pink-800 border-pink-200',
  other_income: 'bg-teal-100 text-teal-800 border-teal-200',
  loans: 'bg-orange-100 text-orange-800 border-orange-200',
};

const writtenQuestionStyling: Record<
  WrittenQuestionStatus,
  { label: string; icon: typeof Clock; badge: string }
> = {
  pending: {
    label: 'Pending',
    icon: Clock,
    badge: 'bg-amber-100 text-amber-800 border-amber-200',
  },
  answered: {
    label: 'Answered',
    icon: CheckSquare,
    badge: 'bg-emerald-100 text-emerald-800 border-emerald-200',
  },
  overdue: {
    label: 'Overdue',
    icon: AlertTriangle,
    badge: 'bg-rose-100 text-rose-800 border-rose-200',
  },
};

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
    badge: 'bg-stone-100 text-stone-500 border-stone-200',
  },
};

const eventTypeIcons: Record<string, typeof Users> = {
  bill_passed: CheckSquare,
  committee_meeting: Users,
};

export default async function ScorecardPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;

  let scorecard;
  try {
    scorecard = await getMPScorecard(id);
  } catch {
    notFound();
  }

  // Fetch votes, written questions, and registered interests in parallel
  const [votesData, questionsData, interestsData] = await Promise.all([
    getMPVotes(id).catch(() => null),
    getMPWrittenQuestions(id).catch(() => null),
    getRegisteredInterests(id).catch(() => null),
  ]);

  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6 lg:px-8">
      <div className="mb-6 flex items-center justify-between">
        <Link
          href="/people"
          className="inline-flex items-center gap-1 text-sm text-civic-stone hover:text-civic-leaf"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden="true" /> All MPs
        </Link>
        <div className="flex gap-2">
          <PrintButton />
          <ScorecardShareButton name={scorecard.name} personId={scorecard.person_id} />
        </div>
      </div>

      {/* Header */}
      <header className="mb-8">
        <h1 className="font-serif text-3xl font-bold text-civic-forest">
          {scorecard.name}
        </h1>
        <p className="mt-1 text-civic-stone">
          {scorecard.role} · {scorecard.constituency} · {scorecard.party}
        </p>
        <div className="mt-2">
          <RealityBadge kind="FACT" />
        </div>
      </header>

      {/* Disclaimer */}
      <div
        className="mb-8 rounded-lg border border-civic-border bg-civic-mist p-4"
        role="note"
      >
        <p className="text-sm text-civic-stone">
          <strong className="text-civic-ink">This scorecard presents factual records only.</strong>{' '}
          The platform does not rank MPs or imply political approval. Every metric links to its source.
        </p>
      </div>

      {/* Metrics */}
      <section className="mb-8">
        <h2 className="mb-4 font-serif text-xl font-semibold text-civic-forest">
          Parliamentary Activity
        </h2>
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {scorecard.metrics?.bills_sponsored !== undefined && (
            <div className="rounded-lg border border-civic-border bg-civic-paper p-4">
              <div className="flex items-center gap-2 text-civic-stone">
                <FileText className="h-4 w-4" aria-hidden="true" />
                <span className="text-xs font-medium uppercase tracking-wide">Bills Sponsored</span>
              </div>
              <p className="mt-2 font-serif text-2xl font-bold text-civic-forest">
                {scorecard.metrics.bills_sponsored.value}
              </p>
            </div>
          )}
          {scorecard.metrics?.questions_asked !== undefined && (
            <div className="rounded-lg border border-civic-border bg-civic-paper p-4">
              <div className="flex items-center gap-2 text-civic-stone">
                <MessageSquareText className="h-4 w-4" aria-hidden="true" />
                <span className="text-xs font-medium uppercase tracking-wide">Questions Asked</span>
              </div>
              <p className="mt-2 font-serif text-2xl font-bold text-civic-forest">
                {scorecard.metrics.questions_asked.value}
              </p>
            </div>
          )}
          {scorecard.metrics?.attendance_rate !== undefined && (
            <div className="rounded-lg border border-civic-border bg-civic-paper p-4">
              <div className="flex items-center gap-2 text-civic-stone">
                <CheckSquare className="h-4 w-4" aria-hidden="true" />
                <span className="text-xs font-medium uppercase tracking-wide">Attendance Rate</span>
              </div>
              <p className="mt-2 font-serif text-2xl font-bold text-civic-forest">
                {formatPercent(scorecard.metrics.attendance_rate.value)}
              </p>
            </div>
          )}
        </div>
      </section>

      {/* Contact Info */}
      {scorecard.email || scorecard.phone || scorecard.office_address || scorecard.twitter || scorecard.facebook ? (
        <section className="mb-8">
          <h2 className="mb-4 font-serif text-xl font-semibold text-civic-forest">
            Contact
          </h2>
          <div className="rounded-lg border border-civic-border bg-civic-paper p-5">
            <div className="grid gap-3 sm:grid-cols-2">
              {scorecard.email && (
                <a href={`mailto:${scorecard.email}`} className="flex items-center gap-2 text-sm text-civic-ink hover:text-civic-leaf">
                  <Mail className="h-4 w-4 text-civic-stone" aria-hidden="true" /> {scorecard.email}
                </a>
              )}
              {scorecard.phone && (
                <a href={`tel:${scorecard.phone}`} className="flex items-center gap-2 text-sm text-civic-ink hover:text-civic-leaf">
                  <Phone className="h-4 w-4 text-civic-stone" aria-hidden="true" /> {scorecard.phone}
                </a>
              )}
              {scorecard.office_address && (
                <div className="flex items-center gap-2 text-sm text-civic-ink">
                  <MapPin className="h-4 w-4 text-civic-stone" aria-hidden="true" /> {scorecard.office_address}
                </div>
              )}
              {scorecard.twitter && (
                <a href={`https://twitter.com/${scorecard.twitter.replace('@', '')}`} target="_blank" rel="noopener noreferrer" className="flex items-center gap-2 text-sm text-civic-ink hover:text-civic-leaf">
                  <Twitter className="h-4 w-4 text-civic-stone" aria-hidden="true" /> {scorecard.twitter}
                </a>
              )}
              {scorecard.facebook && (
                <a href={`https://facebook.com/${scorecard.facebook}`} target="_blank" rel="noopener noreferrer" className="flex items-center gap-2 text-sm text-civic-ink hover:text-civic-leaf">
                  <Facebook className="h-4 w-4 text-civic-stone" aria-hidden="true" /> {scorecard.facebook}
                </a>
              )}
            </div>
            {scorecard._note && (
              <p className="mt-3 text-xs text-civic-stone">{scorecard._note}</p>
            )}
          </div>
        </section>
      ) : null}

      {/* Voting Record */}
      {votesData && votesData.items && votesData.items.length > 0 && (
        <section className="mb-8">
          <h2 className="mb-4 font-serif text-xl font-semibold text-civic-forest">
            Voting Record
          </h2>
          <div className="overflow-hidden rounded-lg border border-civic-border">
            <table className="w-full text-sm">
              <thead className="bg-civic-mist">
                <tr>
                  <th className="px-4 py-2 text-left font-medium text-civic-stone">Bill</th>
                  <th className="px-4 py-2 text-left font-medium text-civic-stone">Vote</th>
                  <th className="px-4 py-2 text-left font-medium text-civic-stone">Date</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-civic-border bg-civic-paper">
                {votesData.items.map((vote: VoteRecord, i: number) => {
                  const styling = voteStyling[vote.vote] || voteStyling.aye;
                  const Icon = styling.icon;
                  return (
                    <tr key={i}>
                      <td className="px-4 py-3 text-civic-ink">
                        {vote.bill_title || vote.bill_id}
                      </td>
                      <td className="px-4 py-3">
                        <span className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium ${styling.badge}`}>
                          <Icon className="h-3 w-3" aria-hidden="true" />
                          {styling.label}
                        </span>
                      </td>
                      <td className="px-4 py-3 text-civic-stone">
                        {formatDate(vote.date)}
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </section>
      )}

      {/* Written Questions */}
      {questionsData && questionsData.items && questionsData.items.length > 0 && (
        <section className="mb-8">
          <h2 className="mb-4 font-serif text-xl font-semibold text-civic-forest">
            Written Questions
          </h2>
          <div className="space-y-3">
            {questionsData.items.map((q: WrittenQuestion, i: number) => {
              const styling = writtenQuestionStyling[q.status] || writtenQuestionStyling.pending;
              const Icon = styling.icon;
              return (
                <div key={i} className="rounded-lg border border-civic-border bg-civic-paper p-4">
                  <div className="flex items-start justify-between gap-3">
                    <div className="flex-1">
                      <p className="text-sm font-medium text-civic-ink">{q.question_text}</p>
                      <p className="mt-1 text-xs text-civic-stone">
                        To {q.minister}, {q.ministry} · Asked {formatDate(q.asked_at)}
                      </p>
                      {q.response_text && (
                        <div className="mt-2 rounded-md bg-civic-mist p-2 text-xs text-civic-ink">
                          <strong>Response:</strong> {q.response_text}
                        </div>
                      )}
                    </div>
                    <span className={`inline-flex items-center gap-1 rounded-full border px-2 py-0.5 text-xs font-medium ${styling.badge}`}>
                      <Icon className="h-3 w-3" aria-hidden="true" />
                      {styling.label}
                    </span>
                  </div>
                </div>
              );
            })}
          </div>
        </section>
      )}

      {/* Registered Interests */}
      {interestsData && interestsData.items && interestsData.items.length > 0 && (
        <section className="mb-8">
          <h2 className="mb-4 font-serif text-xl font-semibold text-civic-forest">
            Registered Interests
          </h2>
          <div className="grid gap-3 sm:grid-cols-2">
            {interestsData.items.map((interest: RegisteredInterest, i: number) => {
              const cat = interest.category as RegisteredInterestCategory;
              const label = categoryLabels[cat] || cat;
              const colour = categoryColours[cat] || 'bg-stone-100 text-stone-800 border-stone-200';
              return (
                <div key={i} className="rounded-lg border border-civic-border bg-civic-paper p-4">
                  <div className="flex items-start justify-between gap-2">
                    <span className={`inline-flex items-center rounded-full border px-2 py-0.5 text-xs font-medium ${colour}`}>
                      {label}
                    </span>
                    <span className="text-xs text-civic-stone">{formatDate(interest.declared_at)}</span>
                  </div>
                  <p className="mt-2 text-sm text-civic-ink">{interest.description}</p>
                  {interest.value_kes > 0 && (
                    <p className="mt-1 text-xs text-civic-stone">
                      Value: KES {interest.value_kes.toLocaleString()}
                    </p>
                  )}
                  {interest.source_url && (
                    <a href={interest.source_url} target="_blank" rel="noopener noreferrer" className="mt-2 inline-flex items-center gap-1 text-xs text-civic-leaf hover:underline">
                      <ExternalLink className="h-3 w-3" aria-hidden="true" /> Source
                    </a>
                  )}
                </div>
              );
            })}
          </div>
        </section>
      )}

      {/* Recent Activity */}
      {scorecard.recent_activity && scorecard.recent_activity.length > 0 && (
        <section className="mb-8">
          <h2 className="mb-4 font-serif text-xl font-semibold text-civic-forest">
            Recent Activity
          </h2>
          <div className="space-y-2">
            {scorecard.recent_activity.map((item, i) => {
              const Icon = eventTypeIcons[item.kind] || FileText;
              return (
                <div key={i} className="flex items-center gap-3 rounded-lg border border-civic-border bg-civic-paper p-3">
                  <Icon className="h-4 w-4 flex-shrink-0 text-civic-stone" aria-hidden="true" />
                  <div className="flex-1">
                    <p className="text-sm font-medium text-civic-ink">{item.title}</p>
                    <p className="text-xs text-civic-stone">{formatDate(item.date)}</p>
                  </div>
                </div>
              );
            })}
          </div>
        </section>
      )}
    </div>
  );
}
