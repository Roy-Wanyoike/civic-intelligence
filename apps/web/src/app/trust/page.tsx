'use client';

import { useCallback, useEffect, useMemo, useState } from 'react';
import Link from 'next/link';
import {
  AlertTriangle,
  CheckCircle2,
  ChevronDown,
  ChevronRight,
  ExternalLink,
  FileText,
  Loader2,
  ShieldCheck,
  ShieldAlert,
  ShieldQuestion,
  XCircle,
} from 'lucide-react';
import {
  ApiError,
  type ClaimWithEvidence,
  type Contradiction,
  type TrustSource,
  type AuthorityLevel,
  getProvenance,
  listContradictions,
  listTrustSources,
} from '@/lib/api';
import { cn } from '@/lib/utils';

/**
 * AuthorityLevel visual + textual metadata. The rank controls sort order
 * (PRIMARY_OFFICIAL first) and the badge colour encodes trustworthiness.
 */
const AUTHORITY_META: Record<
  AuthorityLevel,
  { rank: number; label: string; description: string; pill: string }
> = {
  PRIMARY_OFFICIAL: {
    rank: 0,
    label: 'Primary Official',
    description: 'Official government source (parliament.go.ke, president.go.ke)',
    pill: 'bg-civic-leaf text-white',
  },
  OFFICIAL_REPOSITORY: {
    rank: 1,
    label: 'Official Repository',
    description: 'Official legal repository (kenyalaw.org)',
    pill: 'bg-civic-acacia text-civic-ink',
  },
  SECONDARY_VERIFIED: {
    rank: 2,
    label: 'Secondary Verified',
    description: 'Verified secondary source (news with official references)',
    pill: 'bg-civic-stone text-white',
  },
  UNVERIFIED: {
    rank: 3,
    label: 'Unverified',
    description: 'Not used for canonical facts',
    pill: 'bg-civic-border text-civic-ink',
  },
};

/** Health status badge for a source's last liveness probe. */
function HealthBadge({ status }: { status: TrustSource['health_status'] }) {
  const map = {
    healthy: { icon: CheckCircle2, label: 'Healthy', cls: 'text-civic-leaf' },
    degraded: { icon: AlertTriangle, label: 'Degraded', cls: 'text-civic-acacia' },
    down: { icon: XCircle, label: 'Down', cls: 'text-civic-clay' },
    unknown: { icon: ShieldQuestion, label: 'Unknown', cls: 'text-civic-stone' },
  } as const;
  const { icon: Icon, label, cls } = map[status] ?? map.unknown;
  return (
    <span className={cn('inline-flex items-center gap-1 text-xs font-medium', cls)}>
      <Icon className="h-3.5 w-3.5" aria-hidden="true" />
      {label}
    </span>
  );
}

/** Authority badge coloured by rank. */
function AuthorityBadge({ level }: { level: AuthorityLevel }) {
  const meta = AUTHORITY_META[level] ?? AUTHORITY_META.UNVERIFIED;
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-semibold',
        meta.pill,
      )}
    >
      {meta.label}
    </span>
  );
}

/**
 * SourceCard renders a single trust source with authority level, health,
 * last liveness probe, and a link to the official URL.
 */
function SourceCard({ source }: { source: TrustSource }) {
  const lastCheck = source.last_check;
  return (
    <div
      data-testid="trust-source-card"
      className="rounded-lg border border-civic-border bg-civic-paper p-4 transition-shadow hover:shadow-md"
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <h3 className="font-serif text-sm font-semibold text-civic-ink">
            {source.domain}
          </h3>
          <p className="mt-0.5 text-xs text-civic-stone">{source.source_type}</p>
        </div>
        <AuthorityBadge level={source.authority_level} />
      </div>

      <p className="mt-3 text-xs leading-relaxed text-civic-stone">
        {AUTHORITY_META[source.authority_level]?.description ?? '—'}
      </p>

      <div className="mt-3 flex items-center justify-between text-xs">
        <HealthBadge status={source.health_status} />
        {lastCheck && (
          <span className="text-civic-stone">
            {lastCheck.http_status || '—'} · {lastCheck.latency_ms || '—'}ms · TLS{' '}
            {lastCheck.tls_valid ? '✓' : '✗'}
          </span>
        )}
      </div>

      <a
        href={source.official_url}
        target="_blank"
        rel="noopener noreferrer"
        className="mt-3 inline-flex items-center gap-1 text-xs font-medium text-civic-leaf hover:underline"
      >
        Visit source
        <ExternalLink className="h-3 w-3" aria-hidden="true" />
      </a>
    </div>
  );
}

/**
 * EvidenceChainItem renders a single claim + its evidence list as a tree
 * node. Each piece of evidence is a leaf node linking to the source.
 */
function EvidenceChainItem({ entry }: { entry: ClaimWithEvidence }) {
  const [expanded, setExpanded] = useState(true);
  const claim = entry.claim;
  const evidence = entry.evidence ?? [];

  const confidencePct = Math.round((claim.confidence ?? 0) * 100);
  const stateCls = claim.verification_state === 'VERIFIED'
    ? 'text-civic-leaf'
    : claim.verification_state === 'CONFLICTED'
      ? 'text-civic-clay'
      : 'text-civic-stone';

  return (
    <li className="rounded-lg border border-civic-border bg-civic-paper p-4">
      <button
        type="button"
        onClick={() => setExpanded((s) => !s)}
        aria-expanded={expanded}
        className="flex w-full items-start justify-between gap-3 text-left"
      >
        <div className="min-w-0">
          <p className="font-serif text-sm font-semibold text-civic-ink">
            {claim.text}
          </p>
          <div className="mt-2 flex flex-wrap items-center gap-2 text-xs">
            <span className={cn('font-semibold', stateCls)}>
              {claim.verification_state}
            </span>
            <span className="text-civic-stone">·</span>
            <span className="text-civic-stone">
              {claim.claim_type} · {confidencePct}% confidence
            </span>
            <span className="text-civic-stone">·</span>
            <span className="text-civic-stone">
              {claim.subject}
            </span>
          </div>
        </div>
        {expanded ? (
          <ChevronDown className="h-4 w-4 flex-shrink-0 text-civic-stone" aria-hidden="true" />
        ) : (
          <ChevronRight className="h-4 w-4 flex-shrink-0 text-civic-stone" aria-hidden="true" />
        )}
      </button>

      {expanded && evidence.length > 0 && (
        <ul className="mt-3 space-y-2 border-l border-civic-border pl-4">
          {evidence.map((ev) => (
            <li key={ev.id} className="text-xs">
              <div className="flex items-start gap-2">
                <FileText className="mt-0.5 h-3.5 w-3.5 flex-shrink-0 text-civic-stone" aria-hidden="true" />
                <div className="min-w-0">
                  <p className="text-civic-ink">
                    {ev.text_span || '(no text span recorded)'}
                  </p>
                  <p className="mt-0.5 text-civic-stone">
                    {ev.section && <>§{ev.section}</>}
                    {ev.page_number ? ` · p.${ev.page_number}` : ''}
                    {ev.paragraph ? ` · ¶${ev.paragraph}` : ''}
                  </p>
                  <div className="mt-1 flex flex-wrap items-center gap-2">
                    {ev.source && <AuthorityBadge level={ev.source.authority_level} />}
                    <a
                      href={ev.source_url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className="inline-flex items-center gap-1 text-civic-leaf hover:underline"
                    >
                      {new URL(ev.source_url).hostname}
                      <ExternalLink className="h-3 w-3" aria-hidden="true" />
                    </a>
                  </div>
                </div>
              </div>
            </li>
          ))}
        </ul>
      )}
      {expanded && evidence.length === 0 && (
        <p className="mt-3 text-xs italic text-civic-stone">
          No evidence rows attached to this claim yet.
        </p>
      )}
    </li>
  );
}

/**
 * ContradictionCard renders an active source conflict with both claims and
 * both sources side-by-side. Per ADR-0013 we never auto-resolve.
 */
function ContradictionCard({ c }: { c: Contradiction }) {
  const claimA = c.claim_a;
  const claimB = c.claim_b;
  return (
    <div className="rounded-lg border border-civic-clay/40 bg-civic-paper p-4">
      <div className="flex items-center gap-2">
        <AlertTriangle className="h-4 w-4 text-civic-clay" aria-hidden="true" />
        <h3 className="font-serif text-sm font-semibold text-civic-ink">
          Active contradiction
        </h3>
        <span className="ml-auto inline-flex items-center gap-1 rounded-full bg-civic-clay/10 px-2 py-0.5 text-[11px] font-semibold text-civic-clay">
          {c.status}
        </span>
      </div>
      <p className="mt-1 text-xs text-civic-stone">
        Detected {new Date(c.detected_at).toLocaleString('en-KE')}
      </p>

      <div className="mt-3 grid gap-3 sm:grid-cols-2">
        {[claimA, claimB].map((claim, i) => (
          <div key={i} className="rounded-md border border-civic-border bg-civic-mist p-3">
            <p className="text-xs text-civic-stone">
              {i === 0 ? (c.source_a?.domain ?? '—') : (c.source_b?.domain ?? '—')}
            </p>
            <p className="mt-1 text-xs text-civic-ink">{claim?.text ?? 'Claim not found'}</p>
            <p className="mt-1 text-[11px] text-civic-stone">
              {claim?.verification_state ?? '—'}
            </p>
          </div>
        ))}
      </div>

      <p className="mt-3 text-[11px] italic text-civic-stone">
        Per ADR-0013, contradictions are surfaced for human review and never
        silently resolved.
      </p>
    </div>
  );
}

/**
 * Tabs: switch between Evidence chain, Sources, and Contradictions.
 */
type Tab = 'chain' | 'sources' | 'contradictions';

const TABS: { id: Tab; label: string; testId: string }[] = [
  { id: 'chain', label: 'Evidence chain', testId: 'tab-chain' },
  { id: 'sources', label: 'Sources', testId: 'tab-sources' },
  { id: 'contradictions', label: 'Contradictions', testId: 'tab-contradictions' },
];

export default function TrustPage() {
  const [tab, setTab] = useState<Tab>('chain');
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const [claims, setClaims] = useState<ClaimWithEvidence[]>([]);
  const [sources, setSources] = useState<TrustSource[]>([]);
  const [contradictions, setContradictions] = useState<Contradiction[]>([]);

  const refresh = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      // Fetch the example evidence chain for the Housing Bill, all
      // sources, and active contradictions in parallel.
      const [prov, srcs, ctrs] = await Promise.all([
        getProvenance('bill', 'ke-bill-housing-2024'),
        listTrustSources(),
        listContradictions('detected'),
      ]);
      setClaims(prov.claims);
      setSources(srcs.items);
      setContradictions(ctrs.items);
    } catch (err) {
      const msg =
        err instanceof ApiError
          ? `API ${err.status}: ${err.message}`
          : err instanceof Error
            ? err.message
            : 'Unknown error';
      setError(msg);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    refresh();
  }, [refresh]);

  const sortedSources = useMemo(
    () =>
      [...sources].sort(
        (a, b) =>
          AUTHORITY_META[a.authority_level].rank -
          AUTHORITY_META[b.authority_level].rank,
      ),
    [sources],
  );

  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="font-serif text-3xl font-semibold text-civic-forest">
            Trust &amp; Evidence
          </h1>
          <p className="mt-2 max-w-2xl text-sm text-civic-stone">
            Every factual claim on this platform is traceable to an
            authoritative source. Explore the evidence chain, source authority
            levels, and active contradictions.
          </p>
        </div>
        <button
          type="button"
          onClick={refresh}
          disabled={loading}
          className="inline-flex items-center gap-1.5 self-start rounded-md border border-civic-border bg-civic-paper px-3 py-1.5 text-xs font-medium text-civic-ink hover:bg-civic-mist disabled:opacity-50"
        >
          {loading ? (
            <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
          ) : (
            <ShieldCheck className="h-3.5 w-3.5" aria-hidden="true" />
          )}
          Refresh
        </button>
      </header>

      {/* Evidence chain explainer */}
      <section className="mt-8 rounded-lg border border-civic-border bg-civic-paper p-5">
        <h2 className="font-serif text-lg font-semibold text-civic-ink">
          How every claim is verified
        </h2>
        <p className="mt-1 text-sm text-civic-stone">
          A claim is the smallest unit of civic knowledge. Each claim must be
          backed by at least one piece of evidence that points to a snapshot of
          an official document — the same byte stream that was retrieved from
          the source.
        </p>
        <pre
          aria-hidden="true"
          className="mt-3 overflow-x-auto rounded-md bg-civic-mist p-3 text-xs text-civic-ink"
        >{`Claim  →  Evidence  →  Document  →  Snapshot  →  Source  →  Date Retrieved  →  Content Hash`}</pre>
        <p className="mt-3 text-xs text-civic-stone">
          Spotted something wrong?{' '}
          <Link href="/report" className="font-semibold text-civic-leaf hover:underline">
            Report an error →
          </Link>
        </p>
      </section>

      {/* Tabs */}
      <nav
        role="tablist"
        aria-label="Trust explorer sections"
        className="mt-8 flex flex-wrap gap-1 border-b border-civic-border"
      >
        {TABS.map((t) => (
          <button
            key={t.id}
            role="tab"
            aria-selected={tab === t.id}
            data-testid={t.testId}
            onClick={() => setTab(t.id)}
            className={cn(
              'rounded-t-md border-b-2 px-3 py-2 text-sm font-medium transition-colors',
              tab === t.id
                ? 'border-civic-leaf text-civic-ink'
                : 'border-transparent text-civic-stone hover:text-civic-ink',
            )}
          >
            {t.label}
            {t.id === 'contradictions' && contradictions.length > 0 && (
              <span className="ml-1.5 inline-flex items-center justify-center rounded-full bg-civic-clay px-1.5 py-0.5 text-[10px] font-semibold text-white">
                {contradictions.length}
              </span>
            )}
          </button>
        ))}
      </nav>

      {error && (
        <div
          role="alert"
          className="mt-6 flex items-start gap-2 rounded-md border border-civic-clay/40 bg-civic-clay/5 p-4 text-sm text-civic-clay"
        >
          <ShieldAlert className="mt-0.5 h-4 w-4 flex-shrink-0" aria-hidden="true" />
          <div>
            <p className="font-semibold">Could not load trust data</p>
            <p className="mt-1 text-xs">{error}</p>
          </div>
        </div>
      )}

      <div className="mt-6">
        {loading ? (
          <div className="flex items-center justify-center py-12 text-civic-stone">
            <Loader2 className="h-5 w-5 animate-spin" aria-hidden="true" />
            <span className="ml-2 text-sm">Loading trust data…</span>
          </div>
        ) : tab === 'chain' ? (
          claims.length === 0 ? (
            <p className="py-12 text-center text-sm text-civic-stone">
              No claims with evidence yet for this entity.
            </p>
          ) : (
            <ul className="space-y-3">
              {claims.map((entry) => (
                <EvidenceChainItem key={entry.claim.id} entry={entry} />
              ))}
            </ul>
          )
        ) : tab === 'sources' ? (
          <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
            {sortedSources.map((s) => (
              <SourceCard key={s.id} source={s} />
            ))}
          </div>
        ) : contradictions.length === 0 ? (
          <p className="py-12 text-center text-sm text-civic-stone">
            No active contradictions. The contradiction engine will surface
            conflicts here when sources disagree.
          </p>
        ) : (
          <div className="space-y-3">
            {contradictions.map((c) => (
              <ContradictionCard key={c.id} c={c} />
            ))}
          </div>
        )}
      </div>

      {/* Authority legend */}
      <section className="mt-10 rounded-lg border border-civic-border bg-civic-paper p-5">
        <h2 className="font-serif text-lg font-semibold text-civic-ink">
          Source authority levels
        </h2>
        <ul className="mt-3 space-y-2 text-sm">
          {Object.entries(AUTHORITY_META).map(([key, meta]) => (
            <li key={key} className="flex items-start gap-3">
              <span
                className={cn(
                  'inline-flex flex-shrink-0 items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-semibold',
                  meta.pill,
                )}
              >
                {meta.label}
              </span>
              <span className="text-civic-stone">{meta.description}</span>
            </li>
          ))}
        </ul>
      </section>

      {/* Footer CTA */}
      <section className="mt-10 rounded-lg border border-civic-leaf/30 bg-civic-leaf/5 p-5 text-center">
        <p className="font-serif text-base font-semibold text-civic-forest">
          Found something wrong?
        </p>
        <p className="mx-auto mt-1 max-w-md text-xs text-civic-stone">
          Every correction goes through a review process and creates an
          auditable version. We never silently rewrite history.
        </p>
        <Link
          href="/report"
          className="mt-3 inline-flex items-center gap-1.5 rounded-md bg-civic-leaf px-4 py-2 text-sm font-semibold text-white hover:bg-civic-leaf/90"
        >
          Report an error →
        </Link>
      </section>
    </div>
  );
}
