import Link from 'next/link';
import { notFound } from 'next/navigation';
import { ArrowLeft, ExternalLink, FileText, ScrollText, Scale, Bell } from 'lucide-react';
import type { Metadata } from 'next';
import { getJSON_ as getJSON } from '@/lib/api';
import { RealityBadge, RealityDisclaimer } from '@/components/reality-labels';
import { FollowLawButton } from './follow-button';

export const metadata: Metadata = {
  title: 'Act of Parliament',
  description: 'Inspect an Act of Parliament — its origin Bill, assent, commencement, regulations, court history and amendments.',
};

interface ActDetail {
  id: string;
  title: string;
  citation: string;
  assent_date?: string;
  commencement_date?: string;
  source_url?: string;
  status?: string;
  country?: string;
  summary?: string;
}

function formatDate(d?: string): string {
  if (!d) return '—';
  return new Date(d).toLocaleDateString('en-KE', { year: 'numeric', month: 'long', day: 'numeric' });
}

export default async function ActDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  let act: ActDetail | null = null;
  try {
    act = await getJSON<ActDetail>(`/api/v1/acts/${id}`);
  } catch {
    notFound();
  }

  return (
    <div className="mx-auto max-w-4xl px-4 py-10 sm:px-6">
      <Link
        href="/acts"
        className="inline-flex items-center gap-1 text-sm text-civic-stone hover:text-civic-forest"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" /> All acts
      </Link>

      <header className="mt-4 mb-6">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <ScrollText className="h-4 w-4" aria-hidden="true" />
          <span>Act of Parliament · {act.country || 'KE'}</span>
        </div>
        <div className="mt-2 flex items-start justify-between gap-3">
          <h1 className="font-serif text-3xl font-semibold text-civic-forest">
            {act.title}
          </h1>
          <RealityBadge kind="FACT" size="md" />
        </div>
        <p className="mt-1 text-sm text-civic-stone">
          {act.citation}
        </p>
      </header>

      <div className="mb-6">
        <RealityDisclaimer kind="FACT" />
      </div>

      <section className="mb-6 rounded-lg border border-stone-200 bg-white p-5">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">Overview</h2>
        <dl className="mt-3 grid gap-3 text-sm sm:grid-cols-2">
          <div>
            <dt className="text-xs font-semibold text-civic-stone">Citation</dt>
            <dd className="text-stone-800">{act.citation}</dd>
          </div>
          <div>
            <dt className="text-xs font-semibold text-civic-stone">Status</dt>
            <dd className="text-stone-800">{act.status || 'in_force'}</dd>
          </div>
          <div>
            <dt className="text-xs font-semibold text-civic-stone">Assent date</dt>
            <dd className="text-stone-800">{formatDate(act.assent_date)}</dd>
          </div>
          <div>
            <dt className="text-xs font-semibold text-civic-stone">Commencement date</dt>
            <dd className="text-stone-800">{formatDate(act.commencement_date)}</dd>
          </div>
        </dl>
        {act.source_url && (
          <a
            href={act.source_url}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-3 inline-flex items-center gap-1 text-xs text-sky-700 hover:underline"
          >
            Read on Kenya Law <ExternalLink className="h-3 w-3" aria-hidden="true" />
          </a>
        )}
      </section>

      {act.summary && (
        <section className="mb-6 rounded-lg border border-stone-200 bg-white p-5">
          <h2 className="font-serif text-lg font-semibold text-civic-forest">Summary</h2>
          <p className="mt-2 text-sm text-stone-700">{act.summary}</p>
        </section>
      )}

      <section className="mb-6 rounded-lg border border-violet-200 bg-violet-50 p-5">
        <div className="flex items-center gap-2">
          <Bell className="h-5 w-5 text-violet-700" aria-hidden="true" />
          <h2 className="font-serif text-lg font-semibold text-violet-900">
            Follow this law
          </h2>
        </div>
        <p className="mt-1 text-sm text-violet-900">
          Monitor commencement, regulations, implementation, amendments, court
          cases, government notices, institutional actions, and related Bills.
          The platform notifies you when authoritative evidence indicates
          something changed.
        </p>
        <FollowLawButton actId={act.id} />
      </section>

      <section className="mb-6 rounded-lg border border-stone-200 bg-white p-5">
        <div className="flex items-center gap-2">
          <FileText className="h-5 w-5 text-civic-forest" aria-hidden="true" />
          <h2 className="font-serif text-lg font-semibold text-civic-forest">
            What happened after Presidential Assent?
          </h2>
        </div>
        <p className="mt-1 text-sm text-stone-700">
          Every step of the post-assent lifecycle is tracked with evidence.
          Missing records are reported as NOT_VERIFIED — never as
          &ldquo;never happened&rdquo;.
        </p>
        <div className="mt-4 grid gap-3 sm:grid-cols-2">
          <Link
            href={`/acts/${act.id}/audit`}
            className="rounded-lg border border-stone-200 bg-stone-50 p-4 transition hover:border-civic-forest/40"
          >
            <h3 className="font-serif text-base font-semibold text-civic-forest">
              Audit an Act
            </h3>
            <p className="mt-1 text-xs text-stone-700">
              Full lifecycle audit: assent → publication → commencement →
              regulations → implementation → court history → amendments →
              current status → data gaps.
            </p>
          </Link>
          <Link
            href={`/acts/${act.id}/lineage`}
            className="rounded-lg border border-stone-200 bg-stone-50 p-4 transition hover:border-civic-forest/40"
          >
            <h3 className="font-serif text-base font-semibold text-civic-forest">
              Full Legal Lineage
            </h3>
            <p className="mt-1 text-xs text-stone-700">
              Bill → Bill Version → Amendment → Passed Version → Assent → Act
              → Act Version → Regulation → Judicial Interpretation → Repeal.
            </p>
          </Link>
          <Link
            href={`/acts/${act.id}/events`}
            className="rounded-lg border border-stone-200 bg-stone-50 p-4 transition hover:border-civic-forest/40"
          >
            <h3 className="font-serif text-base font-semibold text-civic-forest">
              Post-Assent Events
            </h3>
            <p className="mt-1 text-xs text-stone-700">
              Every event tracked after the Bill became an Act, with source
              evidence.
            </p>
          </Link>
        </div>
      </section>

      <section className="rounded-lg border border-amber-200 bg-amber-50 p-5">
        <div className="flex items-center gap-2">
          <Scale className="h-5 w-5 text-amber-700" aria-hidden="true" />
          <h2 className="font-serif text-lg font-semibold text-amber-900">
            Constitutional Context
          </h2>
          <RealityBadge kind="EVIDENCE" />
        </div>
        <p className="mt-1 text-sm text-amber-900">
          The Constitutional Context Engine distinguishes:
        </p>
        <ul className="mt-2 list-disc pl-5 text-sm text-amber-900">
          <li><strong>EXPLICIT</strong> — the Bill cites the Article directly.</li>
          <li><strong>INFERRED</strong> — the system identifies a possible relationship.</li>
          <li><strong>JUDICIAL</strong> — a court decision interpreted the relationship.</li>
          <li><strong>UNKNOWN</strong> — no sufficiently reliable relationship established.</li>
        </ul>
        <p className="mt-3 text-xs text-amber-800">
          The platform NEVER independently declares an Act unconstitutional
          merely because an AI model detects a possible conflict.
        </p>
      </section>
    </div>
  );
}
