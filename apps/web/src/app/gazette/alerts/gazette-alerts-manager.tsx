'use client';

import { useCallback, useEffect, useState } from 'react';
import {
  Loader2,
  Plus,
  Trash2,
  AlertCircle,
  BellRing,
  ChevronRight,
  ExternalLink,
  ArrowLeft,
} from 'lucide-react';
import {
  ApiError,
  createGazetteAlert,
  deleteGazetteAlert,
  listGazetteAlertMatches,
  listGazetteAlerts,
  type GazetteAlert,
  type GazetteNotice,
} from '@/lib/api';

/**
 * GazetteAlertsManager — client component owning the alert list + the
 * create form + the match view for a selected alert.
 *
 * State machine:
 *   - userId: read from localStorage (set by the future sign-in flow).
 *     When absent, the component falls back to a per-browser anonymous
 *     ID so the feature is usable end-to-end before identity (#20) ships.
 *   - alerts: list of GazetteAlert records (with match_count eagerly
 *     populated by the API).
 *   - selectedAlertId: when set, the right-hand panel renders the
 *     matching gazette notices for that alert.
 *   - keywords: the create-form's textarea value (comma-separated).
 *
 * Layout (lg breakpoint):
 *   ┌────────────────────────┬───────────────────────────┐
 *   │ [Create alert form]    │ [Selected alert matches]  │
 *   │ [Alert list]           │                           │
 *   └────────────────────────┴───────────────────────────┘
 */
const ANON_USER_KEY = 'civic.gazette.anon_user_id';

function getOrCreateAnonUserID(): string {
  if (typeof window === 'undefined') return 'anon-dev';
  try {
    const existing = window.localStorage.getItem(ANON_USER_KEY);
    if (existing) return existing;
    const id = `anon-${Math.random().toString(36).slice(2)}-${Date.now().toString(36)}`;
    window.localStorage.setItem(ANON_USER_KEY, id);
    return id;
  } catch {
    return 'anon-dev';
  }
}

function formatGazetteDate(ymd: string): string {
  const [y, m, d] = ymd.split('-').map(Number);
  if (!y || !m || !d) return ymd;
  return new Date(Date.UTC(y, m - 1, d)).toLocaleDateString('en-KE', {
    weekday: 'short',
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    timeZone: 'UTC',
  });
}

/**
 * HighlightKeywords wraps each occurrence of a matched keyword in the
 * text with a <mark>. Matched keywords are case-insensitive substrings.
 *
 * The implementation splits on keyword boundaries and emits plain text
 * for non-matching segments — avoids dangerouslySetInnerHTML.
 */
function HighlightKeywords({ text, keywords }: { text: string; keywords: string[] }) {
  if (!text) return null;
  if (keywords.length === 0) return <>{text}</>;

  // Build a single regex that matches any keyword, case-insensitive.
  const escaped = keywords
    .map((k) => k.replace(/[.*+?^${}()|[\]\\]/g, '\\$&'))
    .filter((k) => k.length > 0);
  if (escaped.length === 0) return <>{text}</>;
  const re = new RegExp(`(${escaped.join('|')})`, 'gi');

  const parts = text.split(re);
  return (
    <>
      {parts.map((part, i) =>
        re.test(part) ? (
          <mark key={i} className="rounded bg-civic-acacia/30 px-0.5 text-civic-ink">
            {part}
          </mark>
        ) : (
          <span key={i}>{part}</span>
        ),
      )}
    </>
  );
}

export function GazetteAlertsManager() {
  const [userId, setUserId] = useState<string>('');
  const [alerts, setAlerts] = useState<GazetteAlert[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  // Create-form state.
  const [keywordsInput, setKeywordsInput] = useState('');
  const [creating, setCreating] = useState(false);
  const [createError, setCreateError] = useState<string | null>(null);

  // Selected alert + match view state.
  const [selectedAlert, setSelectedAlert] = useState<GazetteAlert | null>(null);
  const [matches, setMatches] = useState<GazetteNotice[]>([]);
  const [matchesNote, setMatchesNote] = useState<string>('');
  const [matchesLoading, setMatchesLoading] = useState(false);
  const [matchesError, setMatchesError] = useState<string | null>(null);

  // Resolve the user ID (anon fallback before identity ships).
  useEffect(() => {
    try {
      const stored = window.localStorage.getItem('civic.token');
      if (stored) {
        // Token present — the BFF will derive the user_id from the principal.
        // We still need a user_id for the query string on the GET endpoint.
        // Use the anon ID; the BFF ignores it for authenticated callers.
        setUserId(getOrCreateAnonUserID());
        return;
      }
    } catch {
      /* localStorage not available */
    }
    setUserId(getOrCreateAnonUserID());
  }, []);

  // Fetch the alert list whenever the user ID resolves.
  useEffect(() => {
    if (!userId) return;
    let cancelled = false;
    (async () => {
      setLoading(true);
      setError(null);
      try {
        const resp = await listGazetteAlerts({ userId });
        if (cancelled) return;
        setAlerts(resp.items ?? []);
      } catch (e) {
        if (cancelled) return;
        setError(
          e instanceof ApiError
            ? `HTTP ${e.status}`
            : e instanceof Error
              ? e.message
              : 'Failed to load alerts',
        );
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [userId]);

  // Fetch matches whenever a new alert is selected.
  useEffect(() => {
    if (!selectedAlert) {
      setMatches([]);
      setMatchesNote('');
      return;
    }
    let cancelled = false;
    (async () => {
      setMatchesLoading(true);
      setMatchesError(null);
      try {
        const resp = await listGazetteAlertMatches({ id: selectedAlert.id });
        if (cancelled) return;
        setMatches(resp.items ?? []);
        setMatchesNote(resp.note ?? '');
      } catch (e) {
        if (cancelled) return;
        setMatchesError(
          e instanceof ApiError
            ? `HTTP ${e.status}`
            : e instanceof Error
              ? e.message
              : 'Failed to load matches',
        );
      } finally {
        if (!cancelled) setMatchesLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [selectedAlert]);

  const handleCreate = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!userId) return;
      const kws = keywordsInput
        .split(',')
        .map((k) => k.trim())
        .filter((k) => k.length > 0);
      if (kws.length === 0) {
        setCreateError('Enter at least one keyword.');
        return;
      }
      setCreating(true);
      setCreateError(null);
      try {
        const rec = await createGazetteAlert({ keywords: kws, userId, country: 'KE' });
        setAlerts((prev) => [rec, ...prev.filter((a) => a.id !== rec.id)]);
        setKeywordsInput('');
        // Eagerly show the matches for the new alert.
        setSelectedAlert(rec);
      } catch (e) {
        setCreateError(
          e instanceof ApiError
            ? `HTTP ${e.status}: ${e.detail ? JSON.stringify(e.detail) : ''}`
            : e instanceof Error
              ? e.message
              : 'Failed to create alert',
        );
      } finally {
        setCreating(false);
      }
    },
    [keywordsInput, userId],
  );

  const handleDelete = useCallback(
    async (id: string) => {
      if (!userId) return;
      // Optimistic update.
      const prev = alerts;
      setAlerts((cur) => cur.filter((a) => a.id !== id));
      if (selectedAlert?.id === id) {
        setSelectedAlert(null);
      }
      try {
        await deleteGazetteAlert({ id, userId });
      } catch {
        // Roll back on failure.
        setAlerts(prev);
      }
    },
    [alerts, selectedAlert, userId],
  );

  return (
    <div className="grid gap-6 lg:grid-cols-[1fr_1.4fr]">
      {/* === Left column: create form + alert list === */}
      <div className="space-y-6">
        {/* Create form */}
        <form onSubmit={handleCreate} className="rounded-lg border border-civic-border bg-civic-paper p-4">
          <h2 className="font-serif text-base font-semibold text-civic-forest">
            Create an alert
          </h2>
          <p className="mt-1 text-xs text-civic-stone">
            Comma-separated keywords. Case-insensitive. Matched against gazette notice titles, summaries, and bodies.
          </p>
          <label className="mt-3 block text-xs font-medium text-civic-stone" htmlFor="gazette-keywords">
            Keywords
          </label>
          <input
            id="gazette-keywords"
            type="text"
            value={keywordsInput}
            onChange={(e) => setKeywordsInput(e.target.value)}
            placeholder="tender, health, tax"
            className="mt-1 w-full rounded-md border border-civic-border bg-civic-paper px-3 py-2 text-sm focus:border-civic-leaf focus:outline-none focus:ring-1 focus:ring-civic-leaf"
            aria-describedby="gazette-keywords-help"
          />
          <p id="gazette-keywords-help" className="mt-1 text-[11px] text-civic-stone">
            e.g. <code className="rounded bg-civic-mist px-1">tender</code> matches all
            gazette tender notices; <code className="rounded bg-civic-mist px-1">health</code> matches SHA, KEMSA, and
            pharmacy notices.
          </p>

          {createError && (
            <p role="alert" className="mt-2 flex items-center gap-1 text-xs text-civic-clay">
              <AlertCircle className="h-3 w-3" aria-hidden="true" />
              {createError}
            </p>
          )}

          <button
            type="submit"
            disabled={creating || !userId}
            className="mt-3 inline-flex w-full items-center justify-center gap-1.5 rounded-md bg-civic-forest px-4 py-2 text-sm font-medium text-civic-paper hover:bg-civic-leaf disabled:cursor-not-allowed disabled:opacity-60"
          >
            {creating ? (
              <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            ) : (
              <Plus className="h-4 w-4" aria-hidden="true" />
            )}
            Create alert
          </button>
        </form>

        {/* Alert list */}
        <div className="rounded-lg border border-civic-border bg-civic-paper p-4">
          <h2 className="font-serif text-base font-semibold text-civic-forest">
            Your alerts
            {alerts.length > 0 && (
              <span className="ml-2 text-xs font-normal text-civic-stone">({alerts.length})</span>
            )}
          </h2>

          {loading && (
            <div className="mt-3 flex items-center gap-2 text-xs text-civic-stone">
              <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
              Loading alerts…
            </div>
          )}

          {error && (
            <div role="alert" className="mt-3 flex items-center gap-2 text-xs text-civic-clay">
              <AlertCircle className="h-3.5 w-3.5" aria-hidden="true" />
              {error}
            </div>
          )}

          {!loading && !error && alerts.length === 0 && (
            <p className="mt-3 text-xs text-civic-stone">
              No alerts yet. Create one above to start tracking Kenya Gazette notices.
            </p>
          )}

          <ul className="mt-3 space-y-2">
            {alerts.map((a) => {
              const isSelected = selectedAlert?.id === a.id;
              return (
                <li
                  key={a.id}
                  className={`rounded-md border p-3 transition ${
                    isSelected
                      ? 'border-civic-leaf bg-civic-leaf/5'
                      : 'border-civic-border hover:bg-civic-mist/50'
                  }`}
                >
                  <button
                    type="button"
                    onClick={() => setSelectedAlert(a)}
                    className="block w-full text-left"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0 flex-1">
                        <div className="flex flex-wrap gap-1">
                          {a.keywords.map((kw) => (
                            <span
                              key={kw}
                              className="rounded-full border border-civic-leaf/30 bg-civic-leaf/5 px-2 py-0.5 text-[11px] text-civic-leaf"
                            >
                              {kw}
                            </span>
                          ))}
                        </div>
                        <p className="mt-1.5 text-[11px] text-civic-stone">
                          Created{' '}
                          {new Date(a.created_at).toLocaleDateString('en-KE', {
                            year: 'numeric',
                            month: 'short',
                            day: 'numeric',
                          })}
                        </p>
                      </div>
                      <div className="flex flex-shrink-0 items-center gap-2">
                        <span
                          className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium ${
                            a.match_count > 0
                              ? 'bg-civic-acacia/15 text-civic-ink'
                              : 'bg-civic-mist text-civic-stone'
                          }`}
                          title={`${a.match_count} matching gazette notices`}
                        >
                          <BellRing className="h-3 w-3" aria-hidden="true" />
                          {a.match_count} match{a.match_count === 1 ? '' : 'es'}
                        </span>
                        <ChevronRight className="h-3.5 w-3.5 text-civic-stone" aria-hidden="true" />
                      </div>
                    </div>
                  </button>
                  <div className="mt-2 flex justify-end border-t border-civic-border pt-2">
                    <button
                      type="button"
                      onClick={() => handleDelete(a.id)}
                      aria-label={`Delete alert for ${a.keywords.join(', ')}`}
                      className="inline-flex items-center gap-1 rounded-md px-2 py-1 text-[11px] text-civic-stone hover:bg-civic-clay/10 hover:text-civic-clay"
                    >
                      <Trash2 className="h-3 w-3" aria-hidden="true" />
                      Delete
                    </button>
                  </div>
                </li>
              );
            })}
          </ul>
        </div>
      </div>

      {/* === Right column: match view for the selected alert === */}
      <div className="lg:sticky lg:top-20 lg:self-start">
        <div className="rounded-lg border border-civic-border bg-civic-paper p-4">
          {!selectedAlert ? (
            <div className="text-center text-sm text-civic-stone">
              <BellRing className="mx-auto h-8 w-8 text-civic-stone/40" aria-hidden="true" />
              <p className="mt-2">Select an alert to see matching gazette notices.</p>
              <p className="mt-1 text-xs">Only already-published notices are shown.</p>
            </div>
          ) : (
            <>
              <div className="flex items-start justify-between gap-2">
                <div className="min-w-0">
                  <button
                    type="button"
                    onClick={() => setSelectedAlert(null)}
                    className="mb-1 inline-flex items-center gap-1 text-[11px] text-civic-stone hover:text-civic-ink"
                  >
                    <ArrowLeft className="h-3 w-3" aria-hidden="true" />
                    Back
                  </button>
                  <h2 className="font-serif text-base font-semibold text-civic-forest">
                    Matching gazette notices
                  </h2>
                  <div className="mt-1 flex flex-wrap gap-1">
                    {selectedAlert.keywords.map((kw) => (
                      <span
                        key={kw}
                        className="rounded-full border border-civic-leaf/30 bg-civic-leaf/5 px-2 py-0.5 text-[11px] text-civic-leaf"
                      >
                        {kw}
                      </span>
                    ))}
                  </div>
                </div>
                <button
                  type="button"
                  onClick={() => handleDelete(selectedAlert.id)}
                  aria-label="Delete this alert"
                  className="inline-flex items-center gap-1 rounded-md border border-civic-border px-2 py-1 text-[11px] text-civic-stone hover:bg-civic-clay/10 hover:text-civic-clay"
                >
                  <Trash2 className="h-3 w-3" aria-hidden="true" />
                  Delete alert
                </button>
              </div>

              {matchesNote && (
                <p className="mt-3 rounded-md bg-civic-acacia/5 px-3 py-2 text-[11px] text-civic-stone">
                  {matchesNote}
                </p>
              )}

              {matchesLoading && (
                <div className="mt-3 flex items-center gap-2 text-xs text-civic-stone">
                  <Loader2 className="h-3.5 w-3.5 animate-spin" aria-hidden="true" />
                  Loading matches…
                </div>
              )}

              {matchesError && (
                <div role="alert" className="mt-3 flex items-center gap-2 text-xs text-civic-clay">
                  <AlertCircle className="h-3.5 w-3.5" aria-hidden="true" />
                  {matchesError}
                </div>
              )}

              {!matchesLoading && !matchesError && matches.length === 0 && (
                <div className="mt-4 rounded-md border border-dashed border-civic-border p-6 text-center text-xs text-civic-stone">
                  No matching notices yet. New gazette notices are checked when they are
                  published in the weekly Friday Kenya Gazette issue.
                </div>
              )}

              <ul className="mt-4 space-y-3">
                {matches.map((n) => (
                  <li
                    key={n.id}
                    className="rounded-md border border-civic-border p-3 hover:bg-civic-mist/30"
                  >
                    <div className="flex items-start justify-between gap-2">
                      <h3 className="text-sm font-semibold text-civic-ink">
                        <HighlightKeywords text={n.title} keywords={n.matched_keywords ?? []} />
                      </h3>
                      {n.source_url && (
                        <a
                          href={n.source_url}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="inline-flex flex-shrink-0 items-center gap-1 text-[11px] text-civic-leaf hover:underline"
                          aria-label="Open source (opens in new tab)"
                        >
                          Source <ExternalLink className="h-3 w-3" aria-hidden="true" />
                        </a>
                      )}
                    </div>
                    <div className="mt-1 flex flex-wrap items-center gap-2 text-[11px] text-civic-stone">
                      <span>{formatGazetteDate(n.gazette_date)}</span>
                      <span aria-hidden="true">·</span>
                      <span>{n.notice_number}</span>
                      {n.volume && (
                        <>
                          <span aria-hidden="true">·</span>
                          <span>{n.volume}</span>
                        </>
                      )}
                      {n.authority && (
                        <>
                          <span aria-hidden="true">·</span>
                          <span>{n.authority}</span>
                        </>
                      )}
                    </div>
                    {n.summary && (
                      <p className="mt-1.5 text-xs text-civic-stone">
                        <HighlightKeywords text={n.summary} keywords={n.matched_keywords ?? []} />
                      </p>
                    )}
                    {n.body_text && (
                      <p className="mt-1 text-[11px] leading-relaxed text-civic-stone">
                        <HighlightKeywords text={n.body_text} keywords={n.matched_keywords ?? []} />
                      </p>
                    )}
                    {n.matched_keywords && n.matched_keywords.length > 0 && (
                      <div className="mt-2 flex flex-wrap items-center gap-1">
                        <span className="text-[10px] uppercase tracking-wide text-civic-stone">
                          Matched:
                        </span>
                        {n.matched_keywords.map((kw) => (
                          <span
                            key={kw}
                            className="rounded-full border border-civic-acacia/40 bg-civic-acacia/10 px-2 py-0.5 text-[10px] text-civic-ink"
                          >
                            {kw}
                          </span>
                        ))}
                      </div>
                    )}
                  </li>
                ))}
              </ul>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
