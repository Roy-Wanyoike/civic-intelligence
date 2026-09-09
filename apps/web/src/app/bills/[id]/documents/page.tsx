import { notFound } from 'next/navigation';
import Link from 'next/link';
import { FileText, ExternalLink } from 'lucide-react';
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
  return { title: `Documents · ${bill?.title ?? 'Bill'}` };
}

export default async function BillDocumentsPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const bill = mockBills.find((b) => b.id === id);
  if (!bill) notFound();

  const docs = Array.from({ length: bill.version_count }, (_, i) => ({
    id: `${bill.id}-doc-${i + 1}`,
    title: `Bill text — version ${i + 1}`,
    retrieved_at: new Date(Date.parse(bill.created_at) + i * 30 * 86400_000).toISOString(),
    source_url: `https://parliament.go.ke/bills/${bill.year}/${bill.identifier.split(' ').join('-').toLowerCase()}/v${i + 1}.pdf`,
    mime_type: 'application/pdf',
  }));

  return (
    <article className="mx-auto max-w-4xl px-4 py-10 sm:px-6">
      <nav aria-label="Breadcrumb" className="mb-4 text-sm text-civic-stone">
        <ol className="flex items-center gap-2">
          <li><Link href="/bills" className="hover:text-civic-leaf">Bills</Link></li>
          <li aria-hidden="true">/</li>
          <li><Link href={`/bills/${bill.id}`} className="hover:text-civic-leaf">{bill.identifier}</Link></li>
          <li aria-hidden="true">/</li>
          <li className="text-civic-ink">Documents</li>
        </ol>
      </nav>

      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Documents</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Original official documents. Each is archived immutably with a content hash,
          retrieval timestamp, and source URL.
        </p>
      </header>

      <ul className="mt-6 divide-y divide-civic-border rounded-lg border border-civic-border bg-civic-paper">
        {docs.map((d) => (
          <li key={d.id} className="flex items-center justify-between p-4">
            <div className="flex items-start gap-3">
              <FileText className="mt-0.5 h-5 w-5 text-civic-stone" aria-hidden="true" />
              <div>
                <p className="font-medium text-civic-ink">{d.title}</p>
                <p className="mt-1 text-xs text-civic-stone">
                  Retrieved {formatDate(d.retrieved_at)} · {d.mime_type}
                </p>
              </div>
            </div>
            <a href={d.source_url} target="_blank" rel="noopener noreferrer" className="inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline">
              View source <ExternalLink className="h-3 w-3" aria-hidden="true" />
            </a>
          </li>
        ))}
      </ul>
    </article>
  );
}
