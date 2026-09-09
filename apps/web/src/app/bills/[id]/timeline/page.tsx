import { notFound } from 'next/navigation';
import Link from 'next/link';
import { mockBills, mockTimeline } from '@/lib/mock-data';
import { TimelineView } from '@/components/timeline';
import { formatDate } from '@/lib/utils';
import type { Metadata } from 'next';

export const dynamic = 'force-static';

export async function generateStaticParams() {
  return mockBills.map((b) => ({ id: b.id }));
}

export async function generateMetadata({
  params,
}: {
  params: Promise<{ id: string }>;
}): Promise<Metadata> {
  const { id } = await params;
  const bill = mockBills.find((b) => b.id === id);
  return { title: `Timeline · ${bill?.title ?? 'Bill'}` };
}

export default async function BillTimelinePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const bill = mockBills.find((b) => b.id === id);
  if (!bill) notFound();
  const events = mockTimeline.filter((e) => e.bill_id === bill.id);

  return (
    <article className="mx-auto max-w-4xl px-4 py-10 sm:px-6">
      <nav aria-label="Breadcrumb" className="mb-4 text-sm text-civic-stone">
        <ol className="flex items-center gap-2">
          <li><Link href="/bills" className="hover:text-civic-leaf">Bills</Link></li>
          <li aria-hidden="true">/</li>
          <li><Link href={`/bills/${bill.id}`} className="hover:text-civic-leaf">{bill.identifier}</Link></li>
          <li aria-hidden="true">/</li>
          <li className="text-civic-ink">Timeline</li>
        </ol>
      </nav>

      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Timeline</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Verified events for <strong className="text-civic-ink">{bill.title}</strong>.
          Every event links to its source. Dates that could not be confirmed are
          preserved as approximate or unknown — never fabricated.
        </p>
      </header>

      <TimelineView events={events} />

      <p className="mt-8 text-xs text-civic-stone">
        Last updated {formatDate(bill.updated_at)} · {events.length} verified event(s).
      </p>
    </article>
  );
}
