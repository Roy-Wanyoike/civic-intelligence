'use client';

import { useCallback, useEffect, useMemo, useRef, useState } from 'react';
import {
  ChevronLeft,
  ChevronRight,
  Landmark,
  Users,
  FileText,
  Megaphone,
  BookOpen,
  Scale,
  DollarSign,
  Loader2,
  ExternalLink,
  Bell,
  AlertCircle,
  Table as TableIcon,
  Grid as GridIcon,
} from 'lucide-react';
import {
  listCalendarEvents,
  ApiError,
  CALENDAR_EVENT_TYPES,
  type CalendarEvent,
  type CalendarEventType,
} from '@/lib/api';

/**
 * CalendarView — interactive month calendar.
 *
 * Layout:
 *   ┌──────────────────────────────────────────────────────────┐
 *   │ [‹] September 2026 [›]    [Filter ☑☑☑☑☑☑☑]   [Grid|Table] │
 *   ├──────────────────────────────────────┬───────────────────┤
 *   │  Mon Tue Wed Thu Fri Sat Sun          │ Selected day      │
 *   │  ·························            │ ───────────────── │
 *   │  (7-col grid; each cell shows up to   │ • Event 1         │
 *   │   3 event blocks + "+N more")         │ • Event 2         │
 *   │                                       │ [Subscribe]       │
 *   └──────────────────────────────────────┴───────────────────┘
 *
 * Accessibility:
 *   - The month grid is a real <table role="grid"> with <th> headers
 *     and a <caption>. Screen readers announce the month + day-of-week
 *     + event count per cell.
 *   - A separate "Table view" toggle (aria-pressed) renders the same
 *     events as a flat chronological list (more keyboard-friendly).
 *   - Mobile: swipe left/right on the grid shifts the month (touch
 *     events) — handled by touchstart/touchend x-delta detection.
 *
 * Print:
 *   - The side panel + filter chips + view toggle are hidden in print
 *     CSS (@media print) via the `print:hidden` Tailwind variant; the
 *     grid expands to full width.
 */

// --- Event type metadata ---

interface EventTypeMeta {
  label: string;
  icon: typeof FileText;
  /** Tailwind classes for the day-cell chip + the filter chip swatch. */
  chipClass: string;
  dotClass: string;
}

const EVENT_TYPE_META: Record<CalendarEventType, EventTypeMeta> = {
  parliament_session: {
    label: 'Parliament Session',
    icon: Landmark,
    chipClass: 'bg-civic-forest/10 text-civic-forest border-civic-forest/30',
    dotClass: 'bg-civic-forest',
  },
  committee_meeting: {
    label: 'Committee Meeting',
    icon: Users,
    chipClass: 'bg-civic-leaf/10 text-civic-leaf border-civic-leaf/30',
    dotClass: 'bg-civic-leaf',
  },
  bill_reading: {
    label: 'Bill Reading',
    icon: FileText,
    chipClass: 'bg-civic-acacia/15 text-civic-ink border-civic-acacia/40',
    dotClass: 'bg-civic-acacia',
  },
  public_participation: {
    label: 'Public Participation',
    icon: Megaphone,
    chipClass: 'bg-civic-clay/10 text-civic-clay border-civic-clay/30',
    dotClass: 'bg-civic-clay',
  },
  gazette_publication: {
    label: 'Gazette Publication',
    icon: BookOpen,
    chipClass: 'bg-civic-stone/10 text-civic-stone border-civic-stone/30',
    dotClass: 'bg-civic-stone',
  },
  court_hearing: {
    label: 'Court Hearing',
    icon: Scale,
    chipClass: 'bg-civic-ink/10 text-civic-ink border-civic-ink/30',
    dotClass: 'bg-civic-ink',
  },
  budget_presentation: {
    label: 'Budget Presentation',
    icon: DollarSign,
    chipClass: 'bg-civic-acacia/20 text-civic-ink border-civic-acacia/50',
    dotClass: 'bg-civic-acacia',
  },
};

// --- Date helpers (UTC-only; calendar events are date-only) ---

function parseYMD(ymd: string): Date {
  // Date-only parses as UTC midnight; do NOT use new Date(ymd) which
  // shifts to local timezone and can move the day backward.
  const [y, m, d] = ymd.split('-').map(Number);
  return new Date(Date.UTC(y, m - 1, d));
}

function formatYMD(d: Date): string {
  return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, '0')}-${String(
    d.getUTCDate(),
  ).padStart(2, '0')}`;
}

/** Returns "September 2026" formatted in en-KE. */
function formatMonthLabel(ym: string): string {
  const [y, m] = ym.split('-').map(Number);
  return new Date(Date.UTC(y, m - 1, 1)).toLocaleDateString('en-KE', {
    year: 'numeric',
    month: 'long',
    timeZone: 'UTC',
  });
}

/** Returns the YYYY-MM for the month offset by `delta` months from `ym`. */
function shiftMonth(ym: string, delta: number): string {
  const [y, m] = ym.split('-').map(Number);
  const d = new Date(Date.UTC(y, m - 1, 1));
  d.setUTCMonth(d.getUTCMonth() + delta);
  return `${d.getUTCFullYear()}-${String(d.getUTCMonth() + 1).padStart(2, '0')}`;
}

/** Returns the 6-week (42-cell) grid of dates covering `ym`. Weeks start Monday. */
function monthGridDates(ym: string): Date[] {
  const [y, m] = ym.split('-').map(Number);
  const first = new Date(Date.UTC(y, m - 1, 1));
  // JS: Sunday=0, Monday=1, ..., Saturday=6. Shift so Monday is index 0.
  const firstDayIdx = (first.getUTCDay() + 6) % 7;
  const gridStart = new Date(first);
  gridStart.setUTCDate(first.getUTCDate() - firstDayIdx);
  const cells: Date[] = [];
  for (let i = 0; i < 42; i++) {
    const d = new Date(gridStart);
    d.setUTCDate(gridStart.getUTCDate() + i);
    cells.push(d);
  }
  return cells;
}

const WEEKDAY_LABELS = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'];

interface CalendarViewProps {
  initialMonth: string; // YYYY-MM
  initialEvents: CalendarEvent[];
}

export function CalendarView({ initialMonth, initialEvents }: CalendarViewProps) {
  const [month, setMonth] = useState(initialMonth);
  const [events, setEvents] = useState<CalendarEvent[]>(initialEvents);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [selectedDay, setSelectedDay] = useState<string | null>(null);
  const [view, setView] = useState<'grid' | 'table'>('grid');

  // Filter state: a Set of enabled types. Empty set = all types shown
  // (treated as "no filter" rather than "show nothing").
  const [enabledTypes, setEnabledTypes] = useState<Set<CalendarEventType>>(new Set());

  // Touch tracking for mobile swipe (week/month navigation).
  const touchStartX = useRef<number | null>(null);

  // Re-fetch events whenever the month changes.
  useEffect(() => {
    let cancelled = false;
    (async () => {
      setLoading(true);
      setError(null);
      try {
        const resp = await listCalendarEvents({
          month,
          country: 'KE',
          types: enabledTypes.size > 0 ? Array.from(enabledTypes) : undefined,
        });
        if (cancelled) return;
        setEvents(resp.items ?? []);
      } catch (e) {
        if (cancelled) return;
        setError(
          e instanceof ApiError
            ? `HTTP ${e.status}`
            : e instanceof Error
              ? e.message
              : 'Failed to load',
        );
      } finally {
        if (!cancelled) setLoading(false);
      }
    })();
    return () => {
      cancelled = true;
    };
  }, [month, enabledTypes]);

  // Filtered events (also applied client-side so toggling a filter is
  // instant without re-fetching — the server also honours ?types= but
  // the client filter is what the user sees).
  const filteredEvents = useMemo(() => {
    if (enabledTypes.size === 0) return events;
    return events.filter((e) => enabledTypes.has(e.type));
  }, [events, enabledTypes]);

  // Events grouped by YYYY-MM-DD for O(1) cell lookup.
  const eventsByDay = useMemo(() => {
    const map = new Map<string, CalendarEvent[]>();
    for (const e of filteredEvents) {
      const arr = map.get(e.date) ?? [];
      arr.push(e);
      map.set(e.date, arr);
    }
    // Sort each day's events by title so the panel is stable.
    for (const arr of map.values()) {
      arr.sort((a, b) => a.title.localeCompare(b.title));
    }
    return map;
  }, [filteredEvents]);

  const grid = monthGridDates(month);
  const today = formatYMD(new Date());
  const selectedDayEvents = selectedDay ? (eventsByDay.get(selectedDay) ?? []) : [];

  // Default the selected day to today if today is in the visible month.
  useEffect(() => {
    if (selectedDay) return;
    if (today.startsWith(month)) {
      setSelectedDay(today);
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [month]);

  function handlePrevMonth() {
    setMonth((m) => shiftMonth(m, -1));
    setSelectedDay(null);
  }
  function handleNextMonth() {
    setMonth((m) => shiftMonth(m, 1));
    setSelectedDay(null);
  }

  function toggleType(t: CalendarEventType) {
    setEnabledTypes((prev) => {
      const next = new Set(prev);
      if (next.has(t)) next.delete(t);
      else next.add(t);
      return next;
    });
  }

  function clearFilters() {
    setEnabledTypes(new Set());
  }

  // Touch handlers for mobile swipe.
  const onTouchStart = useCallback((e: React.TouchEvent) => {
    touchStartX.current = e.touches[0]?.clientX ?? null;
  }, []);
  const onTouchEnd = useCallback(
    (e: React.TouchEvent) => {
      if (touchStartX.current == null) return;
      const endX = e.changedTouches[0]?.clientX ?? touchStartX.current;
      const delta = endX - touchStartX.current;
      const threshold = 50; // px
      if (delta > threshold) {
        handlePrevMonth();
      } else if (delta < -threshold) {
        handleNextMonth();
      }
      touchStartX.current = null;
    },
    // handlePrev/handleNextMonth are stable (they only call setState).
    [],
  );

  return (
    <div className="grid gap-6 lg:grid-cols-[1fr_22rem]">
      {/* === Main column: month grid OR table view === */}
      <div className="space-y-4">
        {/* Toolbar */}
        <div className="flex flex-wrap items-center justify-between gap-2 print:hidden">
          <div className="flex items-center gap-2">
            <button
              type="button"
              onClick={handlePrevMonth}
              aria-label="Previous month"
              className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-civic-border text-civic-forest hover:bg-civic-mist"
            >
              <ChevronLeft className="h-4 w-4" aria-hidden="true" />
            </button>
            <span className="min-w-[10rem] text-center font-serif text-lg font-semibold text-civic-forest">
              {formatMonthLabel(month)}
            </span>
            <button
              type="button"
              onClick={handleNextMonth}
              aria-label="Next month"
              className="inline-flex h-9 w-9 items-center justify-center rounded-md border border-civic-border text-civic-forest hover:bg-civic-mist"
            >
              <ChevronRight className="h-4 w-4" aria-hidden="true" />
            </button>
            <button
              type="button"
              onClick={() => {
                const now = new Date();
                setMonth(
                  `${now.getUTCFullYear()}-${String(now.getUTCMonth() + 1).padStart(2, '0')}`,
                );
                setSelectedDay(null);
              }}
              className="ml-1 rounded-md border border-civic-border px-3 py-1.5 text-xs text-civic-stone hover:bg-civic-mist"
            >
              Today
            </button>
          </div>

          {/* View toggle: Grid (default) vs Table (accessible list view) */}
          <div className="inline-flex items-center rounded-md border border-civic-border p-0.5" role="group" aria-label="Calendar view">
            <button
              type="button"
              onClick={() => setView('grid')}
              aria-pressed={view === 'grid'}
              className={`inline-flex items-center gap-1 rounded px-2 py-1 text-xs ${
                view === 'grid'
                  ? 'bg-civic-forest text-civic-paper'
                  : 'text-civic-stone hover:bg-civic-mist'
              }`}
            >
              <GridIcon className="h-3.5 w-3.5" aria-hidden="true" />
              Grid
            </button>
            <button
              type="button"
              onClick={() => setView('table')}
              aria-pressed={view === 'table'}
              className={`inline-flex items-center gap-1 rounded px-2 py-1 text-xs ${
                view === 'table'
                  ? 'bg-civic-forest text-civic-paper'
                  : 'text-civic-stone hover:bg-civic-mist'
              }`}
            >
              <TableIcon className="h-3.5 w-3.5" aria-hidden="true" />
              Table
            </button>
          </div>
        </div>

        {/* Filter chips */}
        <fieldset className="print:hidden">
          <legend className="sr-only">Filter by event type</legend>
          <div className="flex flex-wrap gap-1.5">
            {CALENDAR_EVENT_TYPES.map((t) => {
              const meta = EVENT_TYPE_META[t];
              const active = enabledTypes.size === 0 || enabledTypes.has(t);
              const Icon = meta.icon;
              return (
                <label
                  key={t}
                  className={`inline-flex cursor-pointer items-center gap-1.5 rounded-full border px-2.5 py-1 text-xs transition ${
                    active
                      ? meta.chipClass
                      : 'border-civic-border bg-civic-mist text-civic-stone opacity-60'
                  }`}
                >
                  <input
                    type="checkbox"
                    checked={enabledTypes.size === 0 ? true : enabledTypes.has(t)}
                    onChange={() => toggleType(t)}
                    className="sr-only"
                  />
                  <Icon className="h-3 w-3" aria-hidden="true" />
                  {meta.label}
                  <span
                    className={`ml-1 inline-block h-1.5 w-1.5 rounded-full ${meta.dotClass}`}
                    aria-hidden="true"
                  />
                </label>
              );
            })}
            {enabledTypes.size > 0 && (
              <button
                type="button"
                onClick={clearFilters}
                className="rounded-full border border-civic-border px-2.5 py-1 text-xs text-civic-stone hover:bg-civic-mist"
              >
                Clear
              </button>
            )}
          </div>
        </fieldset>

        {error && (
          <div role="alert" className="flex items-center gap-2 rounded-lg border border-civic-clay/30 bg-civic-clay/5 p-3 text-sm text-civic-clay">
            <AlertCircle className="h-4 w-4" aria-hidden="true" />
            {error}
          </div>
        )}

        {loading && (
          <div className="flex items-center gap-2 rounded-lg border border-civic-border p-3 text-sm text-civic-stone">
            <Loader2 className="h-4 w-4 animate-spin" aria-hidden="true" />
            Loading…
          </div>
        )}

        {/* === Grid view === */}
        {view === 'grid' ? (
          <div
            className="overflow-x-auto"
            onTouchStart={onTouchStart}
            onTouchEnd={onTouchEnd}
            aria-label="Calendar month grid. Swipe left or right to change months on mobile."
          >
            <table className="w-full min-w-[44rem] border-collapse text-sm">
              <caption className="sr-only">
                Civic calendar for {formatMonthLabel(month)}. Each cell lists the day's events.
              </caption>
              <thead>
                <tr>
                  {WEEKDAY_LABELS.map((d) => (
                    <th
                      key={d}
                      scope="col"
                      className="border-b border-civic-border px-2 py-1.5 text-left text-xs font-semibold uppercase tracking-wide text-civic-stone"
                    >
                      {d}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody>
                {Array.from({ length: 6 }).map((_, weekIdx) => (
                  <tr key={weekIdx}>
                    {grid.slice(weekIdx * 7, weekIdx * 7 + 7).map((day) => {
                      const ymd = formatYMD(day);
                      const inMonth = day.getUTCMonth() === Number(month.split('-')[1]) - 1;
                      const dayEvents = eventsByDay.get(ymd) ?? [];
                      const isToday = ymd === today;
                      const isSelected = ymd === selectedDay;
                      return (
                        <td
                          key={ymd}
                          role="gridcell"
                          className={`h-24 border border-civic-border p-1 align-top align-middle transition ${
                            inMonth ? 'bg-civic-paper' : 'bg-civic-mist/50'
                          } ${isSelected ? 'ring-2 ring-inset ring-civic-leaf' : ''}`}
                        >
                          <button
                            type="button"
                            onClick={() => setSelectedDay(ymd)}
                            aria-label={`${day.toLocaleDateString('en-KE', {
                              weekday: 'long',
                              year: 'numeric',
                              month: 'long',
                              day: 'numeric',
                              timeZone: 'UTC',
                            })}${dayEvents.length > 0 ? ` — ${dayEvents.length} event${dayEvents.length === 1 ? '' : 's'}` : ''}`}
                            className="flex h-full w-full flex-col items-start gap-0.5 text-left"
                          >
                            <span
                              className={`inline-flex h-6 min-w-6 items-center justify-center rounded-full px-1.5 text-xs font-medium ${
                                isToday
                                  ? 'bg-civic-forest text-civic-paper'
                                  : inMonth
                                    ? 'text-civic-ink'
                                    : 'text-civic-stone'
                              }`}
                            >
                              {day.getUTCDate()}
                            </span>
                            {/* Up to 2 event chips; "+N more" for the rest. */}
                            <div className="hidden w-full flex-col gap-0.5 sm:flex">
                              {dayEvents.slice(0, 2).map((e) => {
                                const meta = EVENT_TYPE_META[e.type];
                                return (
                                  <span
                                    key={e.id}
                                    className={`truncate rounded border px-1 py-0.5 text-[10px] leading-tight ${meta.chipClass}`}
                                    title={e.title}
                                  >
                                    {e.title}
                                  </span>
                                );
                              })}
                              {dayEvents.length > 2 && (
                                <span className="px-1 text-[10px] text-civic-stone">
                                  +{dayEvents.length - 2} more
                                </span>
                              )}
                            </div>
                            {/* Mobile: dots only */}
                            <div className="flex flex-wrap gap-0.5 sm:hidden">
                              {dayEvents.slice(0, 4).map((e) => (
                                <span
                                  key={e.id}
                                  className={`h-1.5 w-1.5 rounded-full ${EVENT_TYPE_META[e.type].dotClass}`}
                                  aria-hidden="true"
                                />
                              ))}
                              {dayEvents.length > 4 && (
                                <span className="text-[9px] text-civic-stone">
                                  +{dayEvents.length - 4}
                                </span>
                              )}
                            </div>
                          </button>
                        </td>
                      );
                    })}
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          /* === Table view (accessible chronological list) === */
          <div className="overflow-x-auto">
            <table className="w-full min-w-[40rem] border-collapse text-sm">
              <caption className="sr-only">
                Civic calendar events for {formatMonthLabel(month)} in chronological order.
              </caption>
              <thead>
                <tr>
                  <th scope="col" className="border-b border-civic-border px-2 py-1.5 text-left text-xs font-semibold uppercase tracking-wide text-civic-stone">
                    Date
                  </th>
                  <th scope="col" className="border-b border-civic-border px-2 py-1.5 text-left text-xs font-semibold uppercase tracking-wide text-civic-stone">
                    Type
                  </th>
                  <th scope="col" className="border-b border-civic-border px-2 py-1.5 text-left text-xs font-semibold uppercase tracking-wide text-civic-stone">
                    Title
                  </th>
                  <th scope="col" className="border-b border-civic-border px-2 py-1.5 text-left text-xs font-semibold uppercase tracking-wide text-civic-stone">
                    Institution
                  </th>
                </tr>
              </thead>
              <tbody>
                {filteredEvents.length === 0 && (
                  <tr>
                    <td colSpan={4} className="border-b border-civic-border px-3 py-6 text-center text-sm text-civic-stone">
                      No events match the current filters.
                    </td>
                  </tr>
                )}
                {filteredEvents
                  .slice()
                  .sort((a, b) => a.date.localeCompare(b.date))
                  .map((e) => {
                    const meta = EVENT_TYPE_META[e.type];
                    const Icon = meta.icon;
                    return (
                      <tr key={e.id} className="hover:bg-civic-mist/50">
                        <td className="border-b border-civic-border px-3 py-2 align-top text-xs text-civic-stone">
                          {parseYMD(e.date).toLocaleDateString('en-KE', {
                            weekday: 'short',
                            day: 'numeric',
                            month: 'short',
                            timeZone: 'UTC',
                          })}
                        </td>
                        <td className="border-b border-civic-border px-3 py-2 align-top">
                          <span
                            className={`inline-flex items-center gap-1 rounded border px-1.5 py-0.5 text-[10px] ${meta.chipClass}`}
                          >
                            <Icon className="h-3 w-3" aria-hidden="true" />
                            {meta.label}
                          </span>
                        </td>
                        <td className="border-b border-civic-border px-3 py-2 align-top text-sm text-civic-ink">
                          {e.title}
                          {e.description && (
                            <p className="mt-0.5 text-xs text-civic-stone">{e.description}</p>
                          )}
                        </td>
                        <td className="border-b border-civic-border px-3 py-2 align-top text-xs text-civic-stone">
                          {e.institution ?? '—'}
                        </td>
                      </tr>
                    );
                  })}
              </tbody>
            </table>
          </div>
        )}

        {/* Mobile-only swipe hint */}
        <p className="text-center text-xs text-civic-stone sm:hidden print:hidden">
          Swipe left / right to change months
        </p>
      </div>

      {/* === Side panel: selected day's events === */}
      <aside
        className="lg:sticky lg:top-20 lg:self-start print:hidden"
        aria-label="Selected day events"
      >
        <div className="rounded-lg border border-civic-border bg-civic-paper p-4">
          <h2 className="font-serif text-base font-semibold text-civic-forest">
            {selectedDay
              ? parseYMD(selectedDay).toLocaleDateString('en-KE', {
                  weekday: 'long',
                  year: 'numeric',
                  month: 'long',
                  day: 'numeric',
                  timeZone: 'UTC',
                })
              : 'Select a day'}
          </h2>
          <p className="mt-1 text-xs text-civic-stone">
            {selectedDayEvents.length === 0
              ? 'No events on this day.'
              : `${selectedDayEvents.length} event${selectedDayEvents.length === 1 ? '' : 's'}`}
          </p>

          <ul className="mt-4 space-y-3">
            {selectedDayEvents.map((e) => {
              const meta = EVENT_TYPE_META[e.type];
              const Icon = meta.icon;
              return (
                <li
                  key={e.id}
                  className={`rounded-md border p-3 ${meta.chipClass}`}
                >
                  <div className="flex items-start justify-between gap-2">
                    <div className="min-w-0">
                      <p className="text-sm font-semibold">{e.title}</p>
                      <p className="mt-0.5 flex items-center gap-1 text-[11px] opacity-80">
                        <Icon className="h-3 w-3" aria-hidden="true" />
                        {meta.label}
                      </p>
                    </div>
                  </div>
                  {e.institution && (
                    <p className="mt-1 text-xs opacity-80">{e.institution}</p>
                  )}
                  {e.description && (
                    <p className="mt-1.5 text-xs opacity-90">{e.description}</p>
                  )}
                  <div className="mt-2 flex flex-wrap items-center gap-2">
                    {e.source_url && (
                      <a
                        href={e.source_url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className="inline-flex items-center gap-1 text-xs underline hover:no-underline"
                      >
                        Source <ExternalLink className="h-3 w-3" aria-hidden="true" />
                      </a>
                    )}
                    <SubscribeButton event={e} />
                  </div>
                </li>
              );
            })}
          </ul>
        </div>
      </aside>
    </div>
  );
}

/**
 * SubscribeButton adds the event to the user's notifications.
 *
 * Until identity (#20) ships, this is informational — it renders a
 * disabled-styled button that opens a tooltip explaining that
 * notifications require sign-in. When a token IS available in
 * localStorage, it posts a follow via createSubscription.
 */
function SubscribeButton({ event }: { event: CalendarEvent }) {
  const [state, setState] = useState<'idle' | 'subscribed' | 'pending' | 'error'>('idle');
  const [token, setToken] = useState<string | undefined>(undefined);

  useEffect(() => {
    try {
      const stored = window.localStorage.getItem('civic.token');
      if (stored) setToken(stored);
    } catch {
      /* localStorage not available */
    }
  }, []);

  async function handleSubscribe() {
    if (!token) {
      setState('error');
      return;
    }
    setState('pending');
    try {
      // createSubscription is in lib/api.ts (issue #110). We import it
      // dynamically so this client component doesn't ship the full
      // subscription API surface unless the user actually subscribes.
      const { createSubscription } = await import('@/lib/api');
      await createSubscription({
        entityType: 'institution',
        entityId: event.id,
        token,
      });
      setState('subscribed');
    } catch {
      setState('error');
    }
  }

  if (state === 'subscribed') {
    return (
      <span className="inline-flex items-center gap-1 rounded-md bg-civic-leaf/15 px-2 py-1 text-xs text-civic-leaf">
        <Bell className="h-3 w-3" aria-hidden="true" />
        Subscribed
      </span>
    );
  }

  return (
    <button
      type="button"
      onClick={handleSubscribe}
      disabled={state === 'pending' || !token}
      title={!token ? 'Sign in to subscribe to event notifications' : undefined}
      className="inline-flex items-center gap-1 rounded-md border border-civic-border px-2 py-1 text-xs text-civic-forest hover:bg-civic-mist disabled:cursor-not-allowed disabled:opacity-50"
    >
      <Bell className="h-3 w-3" aria-hidden="true" />
      {state === 'pending' ? 'Subscribing…' : 'Subscribe'}
    </button>
  );
}
