import Link from 'next/link';
import { Bell, FileText, Users, Building2, Hash, User } from 'lucide-react';
import type { Metadata } from 'next';
import { FollowingList } from './following-list';

export const metadata: Metadata = {
  title: 'Following — Civic Intelligence',
  description: 'Track the Bills, committees, and topics you follow.',
};

export const dynamic = 'force-dynamic';

export default function FollowingPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Following</h1>
        <p className="mt-2 text-sm text-civic-stone">
          The Bills, committees, topics, institutions, and people you follow. You will be notified
          when followed entities change stage, publish a new document, or appear in Hansard.
        </p>
      </header>

      <nav aria-label="Primary" className="mb-6">
        <ul className="flex flex-wrap gap-1 text-sm">
          <li>
            <Link href="/bills" className="rounded-md px-3 py-1.5 text-civic-stone hover:bg-civic-mist hover:text-civic-ink">
              ← Back to Bills
            </Link>
          </li>
          <li>
            <Link href="/notifications" className="rounded-md px-3 py-1.5 text-civic-stone hover:bg-civic-mist hover:text-civic-ink">
              Notifications →
            </Link>
          </li>
        </ul>
      </nav>

      <FollowingList />

      <section className="mt-12 rounded-lg border border-dashed border-civic-border p-6 text-sm text-civic-stone">
        <h2 className="font-serif text-lg font-semibold text-civic-ink">About following</h2>
        <p className="mt-2">
          Each follow creates a row in the <code className="rounded bg-civic-mist px-1 py-0.5 text-xs">notifications.follows</code> table
          (migration 014) — the canonical store. The Go BFF exposes follows at <code className="rounded bg-civic-mist px-1 py-0.5 text-xs">/api/v1/subscriptions</code> (POST to follow, GET to list, DELETE to unfollow).
        </p>
        <p className="mt-2">
          Until identity (#20) is fully wired, follows are stored in memory in the Go API service — they will
          be migrated to Postgres as part of the notifications service rollout (#112).
        </p>
      </section>

      <section className="mt-6">
        <h2 className="font-serif text-lg font-semibold text-civic-ink">What you can follow</h2>
        <ul className="mt-2 grid gap-2 sm:grid-cols-2 lg:grid-cols-3">
          {[
            { icon: FileText, label: 'Bills', desc: 'Stage changes, new versions, assent' },
            { icon: Users, label: 'Committees', desc: 'New reports, public participation calls' },
            { icon: Hash, label: 'Topics', desc: 'Anything from housing to data protection' },
            { icon: Building2, label: 'Institutions', desc: 'NA, Senate, Salaries & Remuneration Authority…' },
            { icon: User, label: 'People', desc: 'Sponsors, committee chairs, CSs' },
            { icon: Bell, label: 'Anything else', desc: '…that the platform learns to track' },
          ].map(({ icon: Icon, label, desc }) => (
            <li key={label} className="rounded-lg border border-civic-border bg-civic-paper p-3">
              <Icon className="h-4 w-4 text-civic-leaf" aria-hidden="true" />
              <h3 className="mt-1 text-sm font-semibold text-civic-ink">{label}</h3>
              <p className="text-xs text-civic-stone">{desc}</p>
            </li>
          ))}
        </ul>
      </section>
    </div>
  );
}
