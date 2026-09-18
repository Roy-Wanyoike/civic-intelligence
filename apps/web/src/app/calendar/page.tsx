import type { Metadata } from 'next';
import { CalendarDays } from 'lucide-react';
import { CalendarView } from './calendar-view';
import { listCalendarEvents } from '@/lib/api';
import { PrintButton } from '@/components/print-button';

export const metadata: Metadata = {
  title: 'Civic Calendar — Civic Intelligence',
  description:
    'Visual calendar of upcoming parliamentary sessions, committee meetings, public participation deadlines, bill readings, gazette publications, court hearings, and budget presentations.',
};

export const dynamic = 'force-dynamic';

/**
 * Civic Calendar page.
 *
 * The server component fetches the current month's events from
 * /api/v1/calendar and hands them to the interactive CalendarView client
 * component. The client component owns the selected month, filters, and
 * selected-day state; it re-fetches when the month changes.
 *
 * Source-of-truth for the calendar is the Go BFF (calendar.go), which
 * currently serves an in-memory sample schedule of 26 Kenya Parliament
 * events. In production this is replaced by a civic_calendar.events
 * SQL repository populated by the ingestion service from
 * parliament.go.ke + kenyalaw.go.ke.
 */
export default async function CalendarPage() {
  const now = new Date();
  const month = `${now.getUTCFullYear()}-${String(now.getUTCMonth() + 1).padStart(2, '0')}`;

  let initialEvents: Awaited<ReturnType<typeof listCalendarEvents>>['items'] = [];
  let loadError: string | null = null;
  try {
    const resp = await listCalendarEvents({ month, country: 'KE' });
    initialEvents = resp.items ?? [];
  } catch (e) {
    loadError = e instanceof Error ? e.message : 'Failed to load calendar events';
  }

  return (
    <div className="mx-auto max-w-7xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <div className="flex items-start justify-between gap-4">
          <div>
            <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
              <CalendarDays className="h-4 w-4" aria-hidden="true" />
              <span>Kenya · Civic Calendar</span>
            </div>
            <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
              Civic Calendar
            </h1>
            <p className="mt-2 max-w-2xl text-sm text-civic-stone">
              Upcoming parliamentary sessions, committee meetings, bill readings,
              public participation deadlines, gazette publications, court hearings,
              and budget presentations. Click any day to see its events.
            </p>
          </div>
          <div className="hidden sm:block">
            <PrintButton />
          </div>
        </div>
      </header>

      {loadError && (
        <div
          role="alert"
          className="mb-6 rounded-lg border border-civic-clay/30 bg-civic-clay/5 p-4 text-sm text-civic-clay"
        >
          Could not load calendar events: {loadError}. The Go BFF may not be running.
        </div>
      )}

      <CalendarView initialMonth={month} initialEvents={initialEvents} />

      <section className="mt-12 rounded-lg border border-dashed border-civic-border p-6 text-sm text-civic-stone">
        <h2 className="font-serif text-lg font-semibold text-civic-ink">About this calendar</h2>
        <p className="mt-2">
          Events are sourced from <code className="rounded bg-civic-mist px-1 py-0.5 text-xs">parliament.go.ke</code>,
          <code className="ml-1 rounded bg-civic-mist px-1 py-0.5 text-xs">kenyalaw.org</code>, and
          <code className="ml-1 rounded bg-civic-mist px-1 py-0.5 text-xs">judiciary.go.ke</code> by the ingestion service.
          The platform does not reinterpret the calendar — every entry links back to the source.
        </p>
        <p className="mt-2">
          <strong>Subscribe to event</strong> adds the event to your notifications (requires sign-in once
          the identity service ships). Until then, the button is informational.
        </p>
      </section>
    </div>
  );
}
