import type { Metadata } from 'next';
import { BellRing, FileSearch } from 'lucide-react';
import { GazetteAlertsManager } from './gazette-alerts-manager';

export const metadata: Metadata = {
  title: 'Gazette Alerts — Civic Intelligence',
  description:
    'Subscribe to keywords in the Kenya Gazette. When a matching legal notice is published, you get notified.',
};

export const dynamic = 'force-dynamic';

/**
 * Gazette Alerts page (task ENG-K1 — Feature 2).
 *
 * The server component renders the page chrome; the interactive
 * GazetteAlertsManager client component owns the alert list + create
 * form + match-view state.
 *
 * IMPORTANT — alert scope (clearly labelled in the UI): alerts are
 * matched ONLY against gazette notices already published
 * (gazette_date <= today). Draft or future-dated notices are never
 * matched. The "alerts are not predictive" caveat is rendered both in
 * the manager + in the API response's `note` field.
 */
export default function GazetteAlertsPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <BellRing className="h-4 w-4" aria-hidden="true" />
          <span>Kenya · Gazette Alerts</span>
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
          Gazette Alerts
        </h1>
        <p className="mt-2 max-w-2xl text-sm text-civic-stone">
          Subscribe to keywords in the Kenya Gazette. When a matching legal
          notice is published, you get notified.
        </p>
      </header>

      {/* Prominent caveat — alerts are not predictive. */}
      <div
        role="note"
        className="mb-8 flex items-start gap-3 rounded-lg border border-civic-acacia/40 bg-civic-acacia/5 p-4 text-sm text-civic-ink"
      >
        <FileSearch className="mt-0.5 h-5 w-5 flex-shrink-0 text-civic-acacia" aria-hidden="true" />
        <div>
          <p className="font-medium">Alerts are checked against published gazette notices only.</p>
          <p className="mt-1 text-xs text-civic-stone">
            Draft notices, future-dated notices, and notices under editorial review are never matched.
            You will be notified the day a matching notice appears in an official Kenya Gazette issue.
          </p>
        </div>
      </div>

      <GazetteAlertsManager />

      <section className="mt-12 rounded-lg border border-dashed border-civic-border p-6 text-sm text-civic-stone">
        <h2 className="font-serif text-lg font-semibold text-civic-ink">About gazette alerts</h2>
        <p className="mt-2">
          Alerts are stored at <code className="rounded bg-civic-mist px-1 py-0.5 text-xs">gazette.alerts</code> in
          production (migration pending). Matches are computed on demand against
          <code className="mx-1 rounded bg-civic-mist px-1 py-0.5 text-xs">gazette.notices</code>, which is
          populated by the ingestion service from <code className="rounded bg-civic-mist px-1 py-0.5 text-xs">kenyalaw.org/kenya_gazette</code>.
        </p>
        <p className="mt-2">
          Until identity (#20) is fully wired, alerts are scoped by an explicit <code className="rounded bg-civic-mist px-1 py-0.5 text-xs">user_id</code> (your
          browser session). When the OIDC token is present, the principal&apos;s user_id takes precedence.
        </p>
      </section>
    </div>
  );
}
