import Link from 'next/link';
import { notFound } from 'next/navigation';
import { ArrowLeft, Users, Calendar, Megaphone, CheckCircle2 } from 'lucide-react';
import type { Metadata } from 'next';
import { getPetition, type PetitionStatus } from '@/lib/petitions-api';
import { SignForm } from './sign-form';

export const metadata: Metadata = {
  title: 'Petition — Public Participation',
  description:
    'Inspect a public-participation e-petition — full text, signatures, and sign form.',
};

// STATUS_BADGE mirrors the list view's badge classes. Kept in the
// detail page (rather than exported from a shared module) so the
// detail page can be lifted into a separate route group later
// without dragging the list view's badge styling with it.
const STATUS_BADGE: Record<PetitionStatus, { label: string; className: string }> = {
  open: {
    label: 'Open',
    className: 'bg-emerald-100 text-emerald-800 border-emerald-200',
  },
  closed: {
    label: 'Closed',
    className: 'bg-stone-100 text-stone-700 border-stone-200',
  },
  answered: {
    label: 'Answered',
    className: 'bg-indigo-100 text-indigo-800 border-indigo-200',
  },
};

function formatDate(d: string): string {
  if (!d) return '—';
  return new Date(d).toLocaleDateString('en-KE', {
    year: 'numeric',
    month: 'long',
    day: 'numeric',
  });
}

function formatDateTime(d: string): string {
  if (!d) return '—';
  return new Date(d).toLocaleString('en-KE', {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function formatNumber(n: number): string {
  return new Intl.NumberFormat('en-KE').format(n);
}

function progressPercent(count: number, target: number): number {
  if (target <= 0) return 0;
  return Math.min(100, Math.max(0, Math.round((count / target) * 100)));
}

// PetitionDetailPage is the server component for
// /participation/{id}. It fetches the petition detail + the
// signatures slice from the Go BFF, then renders the full
// description, the signatures list, and the (client-side) SignForm.
//
// On an unknown id (404) the page calls notFound() so Next.js
// renders the 404 page (matching the rest of the platform's
// behaviour for unknown resources).
export default async function PetitionDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;

  let petition;
  let signatures: Awaited<ReturnType<typeof getPetition>>['signatures'] = [];
  let loadError: string | null = null;
  try {
    const resp = await getPetition(id);
    petition = resp.petition;
    signatures = resp.signatures ?? [];
  } catch (e) {
    // The Go BFF returns 404 for unknown petition ids — surface as
    // Next's notFound() so the standard 404 page renders.
    const status = (e as { status?: number }).status;
    if (status === 404) {
      notFound();
    }
    loadError = e instanceof Error ? e.message : 'Failed to load petition';
  }

  if (loadError || !petition) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
        <div className="rounded-lg border border-rose-300 bg-rose-50 p-4 text-sm text-rose-800">
          {loadError || 'Petition not found.'}
        </div>
        <Link
          href="/participation"
          className="mt-4 inline-flex items-center gap-1 text-sm text-civic-stone hover:text-civic-forest"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden="true" /> Back to petitions
        </Link>
      </div>
    );
  }

  const badge = STATUS_BADGE[petition.status] ?? STATUS_BADGE.open;
  const pct = progressPercent(petition.signatures_count, petition.target_signatures);

  return (
    <div className="mx-auto max-w-4xl px-4 py-10 sm:px-6">
      <Link
        href="/participation"
        className="inline-flex items-center gap-1 text-sm text-civic-stone hover:text-civic-forest"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" /> All petitions
      </Link>

      <header className="mt-4 mb-6">
        <div className="flex flex-wrap items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Megaphone className="h-4 w-4" aria-hidden="true" />
          <span>Public Participation · {petition.country}</span>
          <span
            className={
              'ml-1 inline-flex items-center rounded-full border px-2 py-0.5 text-[10px] font-semibold ' +
              badge.className
            }
          >
            {badge.label}
          </span>
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
          {petition.title}
        </h1>
        <p className="mt-1 text-sm text-civic-stone">
          Started by {petition.created_by} on {formatDate(petition.created_at)}.
        </p>
      </header>

      {/* Progress + meta */}
      <section className="mb-6 rounded-lg border border-stone-200 bg-white p-5">
        <div className="flex items-center justify-between text-sm">
          <span className="inline-flex items-center gap-1 text-stone-700">
            <Users className="h-4 w-4" aria-hidden="true" />
            <strong className="text-civic-forest">
              {formatNumber(petition.signatures_count)}
            </strong>{' '}
            / {formatNumber(petition.target_signatures)} signatures
          </span>
          <span className="font-semibold text-civic-forest">{pct}%</span>
        </div>
        <div
          className="mt-2 h-2 w-full overflow-hidden rounded-full bg-stone-100"
          role="progressbar"
          aria-valuenow={pct}
          aria-valuemin={0}
          aria-valuemax={100}
          aria-label={`${petition.signatures_count} of ${petition.target_signatures} signatures`}
        >
          <div
            className={
              'h-full rounded-full ' +
              (petition.status === 'open'
                ? 'bg-civic-forest'
                : petition.status === 'answered'
                  ? 'bg-indigo-500'
                  : 'bg-stone-400')
            }
            style={{ width: `${pct}%` }}
          />
        </div>
        <div className="mt-3 flex flex-wrap items-center gap-4 text-xs text-civic-stone">
          <span className="inline-flex items-center gap-1">
            <Calendar className="h-3.5 w-3.5" aria-hidden="true" />
            Closes {formatDate(petition.closes_at)}
          </span>
          <span className="inline-flex items-center gap-1">
            <CheckCircle2 className="h-3.5 w-3.5" aria-hidden="true" />
            Target {formatNumber(petition.target_signatures)}
          </span>
        </div>
      </section>

      {/* Full description */}
      <section className="mb-6 rounded-lg border border-stone-200 bg-white p-5">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">
          Petition text
        </h2>
        <p className="mt-3 whitespace-pre-line text-sm leading-6 text-stone-800">
          {petition.description}
        </p>
      </section>

      {/* Signatures list — only Name + SignedAt + Verified are
          surfaced. The email is deliberately NOT rendered on the
          public list (the platform never publishes a signer's email). */}
      <section className="mb-6 rounded-lg border border-stone-200 bg-white p-5">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">
          Signatures ({formatNumber(signatures.length)})
        </h2>
        {signatures.length === 0 ? (
          <p className="mt-3 text-sm text-civic-stone">
            No signatures yet. Be the first to sign.
          </p>
        ) : (
          <ul className="mt-3 divide-y divide-stone-100">
            {signatures.map((sig) => (
              <li
                key={sig.id}
                className="flex items-center justify-between gap-3 py-2 text-sm"
              >
                <span className="font-medium text-stone-800">{sig.name}</span>
                <span className="flex items-center gap-2 text-xs text-civic-stone">
                  <span>{formatDateTime(sig.signed_at)}</span>
                  {sig.verified ? (
                    <span className="inline-flex items-center rounded-full bg-emerald-50 px-2 py-0.5 text-[10px] font-semibold text-emerald-800">
                      Verified
                    </span>
                  ) : (
                    <span className="inline-flex items-center rounded-full bg-amber-50 px-2 py-0.5 text-[10px] font-semibold text-amber-800">
                      Pending verification
                    </span>
                  )}
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* Sign form (client component) */}
      <SignForm petition={petition} />
    </div>
  );
}
