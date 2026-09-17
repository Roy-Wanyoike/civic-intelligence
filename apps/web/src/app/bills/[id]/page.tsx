import { notFound } from 'next/navigation';
import Link from 'next/link';
import { FileText, GitCompare, History, MessageCircle, Bell } from 'lucide-react';
import { mockBills, mockTimeline } from '@/lib/mock-data';
import { TimelineView } from '@/components/timeline';
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
  if (!bill) return { title: 'Bill not found' };
  return { title: bill.title, description: bill.purpose ?? bill.description ?? bill.title };
}

export default async function BillDetailPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  const bill = mockBills.find((b) => b.id === id);
  if (!bill) notFound();

  return (
    <article className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      {/* Breadcrumb */}
      <nav aria-label="Breadcrumb" className="mb-6 text-sm text-civic-stone">
        <ol className="flex items-center gap-2">
          <li><Link href="/" className="hover:text-civic-leaf">Home</Link></li>
          <li aria-hidden="true">/</li>
          <li><Link href="/bills" className="hover:text-civic-leaf">Bills</Link></li>
          <li aria-hidden="true">/</li>
          <li className="text-civic-ink">{bill.identifier}</li>
        </ol>
      </nav>

      {/* Header — title + status */}
      <header className="mb-8">
        <div className="flex flex-wrap items-center gap-2 text-xs text-civic-stone">
          <span className="rounded-full bg-civic-mist px-3 py-1 font-medium text-civic-ink">{bill.identifier}</span>
          <span>{bill.year}</span>
          <span aria-hidden="true">·</span>
          <span>{bill.house_name}</span>
          {bill.committee_name && (<><span aria-hidden="true">·</span><span>{bill.committee_name}</span></>)}
        </div>
        <h1 className="mt-4 font-serif text-3xl font-semibold text-civic-forest sm:text-4xl">
          {bill.title}
        </h1>
        {bill.sponsor_name && (
          <p className="mt-3 text-sm text-civic-stone">Sponsored by {bill.sponsor_name}</p>
        )}
        <div className="mt-5 flex flex-wrap items-center gap-2">
          <span className="rounded-full bg-civic-leaf px-4 py-1.5 text-xs font-semibold text-white">
            {bill.status.replace('_', ' ')}
          </span>
          {bill.current_stage && (
            <span className="rounded-full bg-civic-acacia/20 px-4 py-1.5 text-xs font-medium text-civic-ink">
              Stage: {bill.current_stage}
            </span>
          )}
          {bill.topics?.map((t) => (
            <span key={t} className="rounded-full bg-civic-mist px-2.5 py-1 text-xs text-civic-stone">{t}</span>
          ))}
        </div>
      </header>

      {/* Quick Explanation */}
      <section className="mb-8 rounded-xl border border-civic-border bg-civic-mist/50 p-6">
        <h2 className="font-serif text-xl font-semibold text-civic-forest">In plain language</h2>
        <p className="mt-3 text-civic-ink leading-relaxed">{bill.description}</p>
        {bill.current_stage_simple_explanation && (
          <div className="mt-5 border-l-2 border-civic-acacia pl-4">
            <h3 className="text-sm font-semibold text-civic-ink">What this stage means</h3>
            <p className="mt-1 text-sm text-civic-stone">{bill.current_stage_simple_explanation}</p>
          </div>
        )}
      </section>

      {/* Tab nav */}
      <nav className="mb-8 border-b border-civic-border" aria-label="Bill sections">
        <ul className="flex flex-wrap gap-1">
          {[
            { href: '', icon: FileText, label: 'Overview', active: true },
            { href: '/timeline', icon: History, label: 'Timeline' },
            { href: '/versions', icon: FileText, label: 'Versions' },
            { href: '/compare', icon: GitCompare, label: 'Compare' },
            { href: '/documents', icon: FileText, label: 'Documents' },
            { href: '/chat', icon: MessageCircle, label: 'Ask' },
          ].map(({ href, icon: Icon, label, active }) => (
            <li key={label}>
              <Link
                href={`/bills/${bill.id}${href}`}
                className={`inline-flex items-center gap-2 border-b-2 px-5 py-3.5 text-sm transition ${
                  active
                    ? 'border-civic-leaf font-semibold text-civic-leaf'
                    : 'border-transparent text-civic-stone hover:text-civic-leaf'
                }`}
              >
                <Icon className="h-4 w-4" aria-hidden="true" />
                {label}
              </Link>
            </li>
          ))}
        </ul>
      </nav>

      {/* Body — main content + sidebar */}
      <div className="grid gap-10 lg:grid-cols-3">
        <div className="lg:col-span-2 space-y-10">
          {/* Purpose */}
          {bill.purpose && (
            <section>
              <h2 className="font-serif text-xl font-semibold text-civic-forest">Purpose</h2>
              <p className="mt-3 text-civic-ink leading-relaxed">{bill.purpose}</p>
            </section>
          )}

          {/* Timeline preview */}
          <section>
            <div className="flex items-center justify-between">
              <h2 className="font-serif text-xl font-semibold text-civic-forest">Timeline</h2>
              <Link href={`/bills/${bill.id}/timeline`} className="text-sm text-civic-leaf hover:underline">
                Full timeline →
              </Link>
            </div>
            <TimelineView events={mockTimeline.filter((e) => e.bill_id === bill.id)} limit={5} />
          </section>

          {/* Ask about this Bill */}
          <section>
            <h2 className="font-serif text-xl font-semibold text-civic-forest">Ask about this Bill</h2>
            <p className="mt-1 text-sm text-civic-stone">
              Every answer is evidence-grounded and cites its sources.
            </p>
            <BillAskPanel billId={bill.id} />
          </section>
        </div>

        {/* Sidebar */}
        <aside className="space-y-6">
          <div className="rounded-xl border border-civic-border bg-civic-paper p-5">
            <h3 className="text-sm font-semibold text-civic-ink">People</h3>
            <ul className="mt-3 space-y-1.5 text-sm text-civic-stone">
              <li>Sponsor: {bill.sponsor_name ?? '—'}</li>
              <li>Committee: {bill.committee_name ?? '—'}</li>
              <li>House: {bill.house_name ?? '—'}</li>
            </ul>
          </div>

          <div className="rounded-xl border border-civic-border bg-civic-paper p-5">
            <h3 className="text-sm font-semibold text-civic-ink">Documents</h3>
            <p className="mt-2 text-sm text-civic-stone">
              {bill.version_count} version(s) · {bill.citation_count} citation(s)
            </p>
            <Link
              href={`/bills/${bill.id}/documents`}
              className="mt-3 inline-flex items-center gap-1 text-sm text-civic-leaf hover:underline"
            >
              View documents →
            </Link>
          </div>

          <div className="rounded-xl border border-civic-border bg-civic-paper p-5">
            <h3 className="text-sm font-semibold text-civic-ink">Follow this Bill</h3>
            <p className="mt-2 text-sm text-civic-stone">
              Get notified when this Bill changes stage, is amended, or a new document is published.
            </p>
            <button
              type="button"
              className="mt-3 inline-flex w-full items-center justify-center gap-2 rounded-lg bg-civic-leaf px-4 py-2.5 text-sm font-semibold text-white transition hover:bg-civic-leaf/90"
              data-bill-id={bill.id}
            >
              <Bell className="h-4 w-4" aria-hidden="true" />
              Follow
            </button>
          </div>

          <div className="rounded-xl border border-civic-border bg-civic-paper p-5 text-xs text-civic-stone">
            <p>
              <strong className="text-civic-ink">Disclaimer:</strong> Summaries are AI-generated and
              citation-validated. Not legal advice. Always consult source documents.
            </p>
          </div>
        </aside>
      </div>
    </article>
  );
}
