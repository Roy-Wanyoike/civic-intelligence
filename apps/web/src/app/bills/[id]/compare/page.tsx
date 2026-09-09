import { notFound } from 'next/navigation';
import Link from 'next/link';
import { GitCompare } from 'lucide-react';
import { mockBills } from '@/lib/mock-data';
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
  return { title: `Compare · ${bill?.title ?? 'Bill'}` };
}

export default async function BillComparePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const bill = mockBills.find((b) => b.id === id);
  if (!bill) notFound();

  return (
    <article className="mx-auto max-w-4xl px-4 py-10 sm:px-6">
      <nav aria-label="Breadcrumb" className="mb-4 text-sm text-civic-stone">
        <ol className="flex items-center gap-2">
          <li><Link href="/bills" className="hover:text-civic-leaf">Bills</Link></li>
          <li aria-hidden="true">/</li>
          <li><Link href={`/bills/${bill.id}`} className="hover:text-civic-leaf">{bill.identifier}</Link></li>
          <li aria-hidden="true">/</li>
          <li className="text-civic-ink">Compare</li>
        </ol>
      </nav>

      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Compare versions</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Pick two versions to see exactly what changed. Additions, removals, and
          modifications are shown with a plain-language explanation. We never
          speculate on <em>why</em> a change was made unless the source explicitly
          states the reason.
        </p>
      </header>

      <div className="mt-8 grid gap-4 sm:grid-cols-2">
        <div className="rounded-lg border border-civic-border bg-civic-paper p-6">
          <GitCompare className="h-6 w-6 text-civic-leaf" aria-hidden="true" />
          <h2 className="mt-2 font-serif text-lg font-semibold">Side-by-side</h2>
          <p className="mt-1 text-sm text-civic-stone">
            View both versions in parallel with additions and removals highlighted inline.
          </p>
        </div>
        <div className="rounded-lg border border-civic-border bg-civic-paper p-6">
          <GitCompare className="h-6 w-6 text-civic-acacia" aria-hidden="true" />
          <h2 className="mt-2 font-serif text-lg font-semibold">Clause-level diff</h2>
          <p className="mt-1 text-sm text-civic-stone">
            Grouped by clause, with affected clauses listed at the top.
          </p>
        </div>
      </div>

      <div className="mt-8 rounded-lg border border-dashed border-civic-border p-8 text-center text-sm text-civic-stone">
        Select two versions on the <Link href={`/bills/${bill.id}/versions`} className="text-civic-leaf hover:underline">Versions</Link> page to begin a comparison.
      </div>
    </article>
  );
}
