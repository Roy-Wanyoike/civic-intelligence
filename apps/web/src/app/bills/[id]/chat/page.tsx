import { notFound } from 'next/navigation';
import Link from 'next/link';
import { mockBills } from '@/lib/mock-data';
import { BillAskPanel } from '@/components/bill-ask-panel';
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
  return { title: `Ask · ${bill?.title ?? 'Bill'}` };
}

export default async function BillChatPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const bill = mockBills.find((b) => b.id === id);
  if (!bill) notFound();

  return (
    <article className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <nav aria-label="Breadcrumb" className="mb-4 text-sm text-civic-stone">
        <ol className="flex items-center gap-2">
          <li><Link href="/bills" className="hover:text-civic-leaf">Bills</Link></li>
          <li aria-hidden="true">/</li>
          <li><Link href={`/bills/${bill.id}`} className="hover:text-civic-leaf">{bill.identifier}</Link></li>
          <li aria-hidden="true">/</li>
          <li className="text-civic-ink">Ask</li>
        </ol>
      </nav>

      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Ask about this Bill</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Every answer is grounded in evidence and cites its sources. If we can&apos;t
          verify something, we say so explicitly.
        </p>
      </header>

      <BillAskPanel billId={bill.id} expanded />
    </article>
  );
}
