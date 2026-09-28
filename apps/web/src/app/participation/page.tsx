import Link from 'next/link';
import { Plus, ArrowRight, Users, Calendar, Megaphone } from 'lucide-react';
import type { Metadata } from 'next';
import { listPetitions, type Petition, type PetitionStatus } from '@/lib/petitions-api';

export const metadata: Metadata = {
  title: 'Public Participation — e-Petitions',
  description:
    'Browse, sign, and create public-participation e-petitions. Every record is sourced; the platform never derives a political-performance score.',
};

// STATUS_BADGE maps a PetitionStatus to the Tailwind classes for the
// status badge. The colours are deliberately restrained — open is
// emerald (action available), closed is stone (no action), answered
// is indigo (authority has responded). The platform NEVER uses red
// or warning colours for closed petitions — that would imply a
// negative verdict (rule: NO_POLITICAL_PERFORMANCE_SCORE).
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
    month: 'short',
    day: 'numeric',
  });
}

// formatNumber renders integers with thousands separators so
// "1000000" reads as "1,000,000" — citizens should be able to glance
// at the progress bar + count and immediately understand the scale.
function formatNumber(n: number): string {
  return new Intl.NumberFormat('en-KE').format(n);
}

// progressPercent returns the integer percentage (0–100) of
// signatures_count / target_signatures. Capped at 100 so a petition
// that exceeds its target renders a full bar rather than overflowing.
function progressPercent(p: Petition): number {
  if (p.target_signatures <= 0) return 0;
  const pct = Math.round((p.signatures_count / p.target_signatures) * 100);
  return Math.min(100, Math.max(0, pct));
}

export default async function ParticipationPage({
  searchParams,
}: {
  searchParams: Promise<{ status?: string; country?: string }>;
}) {
  const filters = await searchParams;
  const status = (filters.status || '') as PetitionStatus | '';
  const country = filters.country || '';

  let petitions: Petition[] = [];
  let loadError: string | null = null;
  try {
    const resp = await listPetitions({
      status: status || undefined,
      country: country || undefined,
    });
    petitions = resp.items ?? [];
  } catch (e) {
    loadError = e instanceof Error ? e.message : 'Failed to load petitions';
  }

  const statusFilters: Array<{ value: string; label: string }> = [
    { value: '', label: 'All' },
    { value: 'open', label: 'Open' },
    { value: 'closed', label: 'Closed' },
    { value: 'answered', label: 'Answered' },
  ];

  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Megaphone className="h-4 w-4" aria-hidden="true" />
          <span>Public Participation Portal</span>
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
          e-Petitions
        </h1>
        <p className="mt-2 max-w-2xl text-sm text-civic-stone">
          Browse, sign, and create public-participation e-petitions. Every
          record is sourced; the platform never derives a political-
          performance score from signature counts — the citizen
          interprets the result, the platform does not.
        </p>
      </header>

      {/* Filter bar + create CTA */}
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-xs font-semibold uppercase tracking-wide text-civic-stone">
            Status:
          </span>
          {statusFilters.map((f) => {
            const active = status === f.value;
            const qs = new URLSearchParams();
            if (f.value) qs.set('status', f.value);
            if (country) qs.set('country', country);
            const href = qs.toString() ? `/participation?${qs.toString()}` : '/participation';
            return (
              <Link
                key={f.value || 'all'}
                href={href}
                className={
                  'rounded-full border px-3 py-1 text-xs font-semibold transition ' +
                  (active
                    ? 'border-civic-forest bg-civic-forest text-white'
                    : 'border-stone-200 bg-white text-stone-700 hover:border-civic-forest/40')
                }
              >
                {f.label}
              </Link>
            );
          })}
          {country && (
            <span className="ml-2 inline-flex items-center gap-1 rounded-full border border-sky-200 bg-sky-50 px-3 py-1 text-xs font-semibold text-sky-800">
              Country: {country.toUpperCase()}
              <Link
                href={`/participation${status ? `?status=${status}` : ''}`}
                className="ml-1 text-sky-600 hover:underline"
                aria-label={`Clear country filter ${country}`}
              >
                ×
              </Link>
            </span>
          )}
        </div>
        <Link
          href="/participation/new"
          className="inline-flex items-center gap-2 rounded-lg bg-civic-forest px-4 py-2 text-sm font-semibold text-white transition hover:bg-civic-forest/90"
        >
          <Plus className="h-4 w-4" aria-hidden="true" />
          New petition
        </Link>
      </div>

      {loadError && (
        <div className="rounded-lg border border-rose-300 bg-rose-50 p-4 text-sm text-rose-800">
          Failed to load petitions: {loadError}
        </div>
      )}

      {!loadError && petitions.length === 0 && (
        <div className="rounded-lg border border-dashed border-civic-border p-8 text-center text-sm text-civic-stone">
          No petitions match the current filter. Try clearing the status
          or country filter, or{' '}
          <Link href="/participation/new" className="text-civic-forest underline">
            create a new petition
          </Link>
          .
        </div>
      )}

      <ul className="grid gap-4">
        {petitions.map((p) => {
          const badge = STATUS_BADGE[p.status] ?? STATUS_BADGE.open;
          const pct = progressPercent(p);
          return (
            <li
              key={p.id}
              className="rounded-lg border border-stone-200 bg-white p-5 transition hover:border-civic-forest/40 hover:shadow-sm"
            >
              <div className="flex items-start justify-between gap-3">
                <div className="min-w-0">
                  <div className="flex flex-wrap items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
                    <span
                      className={
                        'inline-flex items-center rounded-full border px-2 py-0.5 text-[10px] font-semibold ' +
                        badge.className
                      }
                    >
                      {badge.label}
                    </span>
                    <span aria-hidden="true">·</span>
                    <span>{p.country}</span>
                  </div>
                  <h2 className="mt-1 font-serif text-lg font-semibold text-civic-forest">
                    <Link href={`/participation/${p.id}`} className="hover:underline">
                      {p.title}
                    </Link>
                  </h2>
                  <p className="mt-1 line-clamp-2 text-sm text-stone-700">
                    {p.description}
                  </p>
                </div>
              </div>

              {/* Signature progress bar */}
              <div className="mt-4">
                <div className="flex items-center justify-between text-xs text-civic-stone">
                  <span className="inline-flex items-center gap-1">
                    <Users className="h-3.5 w-3.5" aria-hidden="true" />
                    {formatNumber(p.signatures_count)}{' '}
                    <span className="text-civic-stone/70">
                      / {formatNumber(p.target_signatures)} signatures
                    </span>
                  </span>
                  <span className="font-semibold text-civic-forest">{pct}%</span>
                </div>
                <div
                  className="mt-1 h-2 w-full overflow-hidden rounded-full bg-stone-100"
                  role="progressbar"
                  aria-valuenow={pct}
                  aria-valuemin={0}
                  aria-valuemax={100}
                  aria-label={`${p.signatures_count} of ${p.target_signatures} signatures`}
                >
                  <div
                    className={
                      'h-full rounded-full ' +
                      (p.status === 'open'
                        ? 'bg-civic-forest'
                        : p.status === 'answered'
                          ? 'bg-indigo-500'
                          : 'bg-stone-400')
                    }
                    style={{ width: `${pct}%` }}
                  />
                </div>
              </div>

              <div className="mt-4 flex items-center justify-between text-xs text-civic-stone">
                <span className="inline-flex items-center gap-1">
                  <Calendar className="h-3.5 w-3.5" aria-hidden="true" />
                  Closes {formatDate(p.closes_at)}
                </span>
                <Link
                  href={`/participation/${p.id}`}
                  className="inline-flex items-center gap-1 text-sm font-semibold text-civic-forest hover:underline"
                >
                  View petition <ArrowRight className="h-3.5 w-3.5" aria-hidden="true" />
                </Link>
              </div>
            </li>
          );
        })}
      </ul>
    </div>
  );
}
