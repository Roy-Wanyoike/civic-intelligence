import { notFound } from 'next/navigation';
import Link from 'next/link';
import { mockBills } from '@/lib/mock-data';
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
  return { title: `Versions · ${bill?.title ?? 'Bill'}` };
}

export default async function BillVersionsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const bill = mockBills.find((b) => b.id === id);
  if (!bill) notFound();

  const versions = Array.from({ length: bill.version_count }, (_, i) => ({
    id: `${bill.id}-v${i + 1}`,
    bill_id: bill.id,
    version_no: i + 1,
    content_hash: `sha256:${bill.id.slice(-8)}-v${i + 1}-${'a'.repeat(8)}`,
    retrieved_at: new Date(Date.parse(bill.created_at) + i * 30 * 86400_000).toISOString(),
    source_url: `https://parliament.go.ke/bills/${bill.year}/${bill.identifier.split(' ').join('-').toLowerCase()}/v${i + 1}`,
    document_id: `${bill.id}-doc-${i + 1}`,
  }));

  return (
    <article className="mx-auto max-w-4xl px-4 py-10 sm:px-6">
      <nav aria-label="Breadcrumb" className="mb-4 text-sm text-civic-stone">
        <ol className="flex items-center gap-2">
          <li><Link href="/bills" className="hover:text-civic-leaf">Bills</Link></li>
          <li aria-hidden="true">/</li>
          <li><Link href={`/bills/${bill.id}`} className="hover:text-civic-leaf">{bill.identifier}</Link></li>
          <li aria-hidden="true">/</li>
          <li className="text-civic-ink">Versions</li>
        </ol>
      </nav>

      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Versions</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Every version of this Bill is preserved immutably. Compare any two versions
          to see what changed — additions, removals, modifications — with a
          plain-language explanation.
        </p>
      </header>

      <ul className="mt-6 divide-y divide-civic-border rounded-lg border border-civic-border bg-civic-paper">
        {[...versions].reverse().map((v, idx) => (
          <li key={v.id} className="flex items-center justify-between p-4">
            <div>
              <div className="flex items-center gap-2">
                <span className="font-semibold text-civic-ink">Version {v.version_no}</span>
                {idx === 0 && (
                  <span className="rounded-full bg-civic-leaf px-2 py-0.5 text-xs font-semibold text-white">
                    Current
                  </span>
                )}
              </div>
              <p className="mt-1 text-xs text-civic-stone">
                Retrieved {formatDate(v.retrieved_at)} · {v.content_hash.slice(0, 24)}…
              </p>
              <a href={v.source_url} target="_blank" rel="noopener noreferrer" className="mt-1 inline-block text-xs text-civic-leaf hover:underline">
                View source document ↗
              </a>
            </div>
          </li>
        ))}
      </ul>
    </article>
  );
}
