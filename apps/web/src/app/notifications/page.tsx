import type { Metadata } from 'next';
import Link from 'next/link';
import { Bell } from 'lucide-react';
import { FollowingList } from '../following/following-list';

export const metadata: Metadata = {
  title: 'Notifications',
  description: 'Your civic notifications and email alert subscriptions.',
};

export default function NotificationsPage() {
  return (
    <div className="mx-auto max-w-4xl px-4 py-10 sm:px-6">
      <h1 className="font-serif text-3xl font-semibold text-civic-forest">Notifications</h1>
      <p className="mt-2 text-sm text-civic-stone">
        Verified civic alerts for Bills, committees, and topics you follow. Toggle the &ldquo;Also
        email me&rdquo; checkbox on any subscription below to receive a daily digest email (issue #279).
      </p>

      <section className="mt-8">
        <div className="rounded-lg border border-dashed border-civic-border p-8 text-center text-sm text-civic-stone">
          <Bell className="mx-auto mb-2 h-5 w-5" aria-hidden="true" />
          <p>In-app notifications will appear here once the live event matcher (issue #112) is wired.</p>
          <p className="mt-1">
            Until then, the daily digest preview is available at{' '}
            <Link href="/brief" className="text-civic-leaf hover:underline">/brief</Link>.
          </p>
        </div>
      </section>

      <section className="mt-10">
        <h2 className="mb-4 font-serif text-xl font-semibold text-civic-ink">
          Email alert subscriptions
        </h2>
        <p className="mb-4 text-sm text-civic-stone">
          Each subscription can deliver alerts in-app (always on) and via the daily digest email
          (opt-in). Toggle the &ldquo;Also email me&rdquo; checkbox to enable email delivery.
        </p>
        <FollowingList />
      </section>
    </div>
  );
}
