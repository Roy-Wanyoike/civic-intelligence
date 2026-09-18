import type { Metadata } from 'next';
import Link from 'next/link';
import { Bell, FileText, TrendingUp, Bookmark } from 'lucide-react';

export const metadata: Metadata = {
  title: 'Dashboard',
  description: 'Your personalized civic intelligence dashboard.',
};

export default function DashboardPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <h1 className="font-serif text-3xl font-semibold text-civic-forest">Your Civic Dashboard</h1>
      <p className="mt-2 text-sm text-civic-stone">
        Personalized civic intelligence — your feed, notifications, and followed matters.
      </p>
      <div className="mt-8 grid gap-6 md:grid-cols-2">
        <div className="rounded-lg border border-civic-border bg-civic-paper p-6">
          <div className="flex items-center gap-2">
            <TrendingUp className="h-5 w-5 text-civic-leaf" />
            <h2 className="font-serif text-lg font-semibold">Civic Feed</h2>
          </div>
          <p className="mt-2 text-sm text-civic-stone">Latest verified civic activity.</p>
          <Link href="/feed" className="mt-3 inline-block text-sm text-civic-leaf hover:underline">View feed →</Link>
        </div>
        <div className="rounded-lg border border-civic-border bg-civic-paper p-6">
          <div className="flex items-center gap-2">
            <Bell className="h-5 w-5 text-civic-acacia" />
            <h2 className="font-serif text-lg font-semibold">Notifications</h2>
          </div>
          <p className="mt-2 text-sm text-civic-stone">Alerts for Bills, committees, and topics you follow.</p>
          <Link href="/notifications" className="mt-3 inline-block text-sm text-civic-leaf hover:underline">View notifications →</Link>
        </div>
        <div className="rounded-lg border border-civic-border bg-civic-paper p-6">
          <div className="flex items-center gap-2">
            <Bookmark className="h-5 w-5 text-civic-clay" />
            <h2 className="font-serif text-lg font-semibold">Following</h2>
          </div>
          <p className="mt-2 text-sm text-civic-stone">Bills, committees, and topics you follow.</p>
          <Link href="/following" className="mt-3 inline-block text-sm text-civic-leaf hover:underline">View follows →</Link>
        </div>
        <div className="rounded-lg border border-civic-border bg-civic-paper p-6">
          <div className="flex items-center gap-2">
            <FileText className="h-5 w-5 text-civic-stone" />
            <h2 className="font-serif text-lg font-semibold">Daily Brief</h2>
          </div>
          <p className="mt-2 text-sm text-civic-stone">Your personalized civic briefing.</p>
          <Link href="/briefing" className="mt-3 inline-block text-sm text-civic-leaf hover:underline">Read brief →</Link>
        </div>
      </div>
    </div>
  );
}
