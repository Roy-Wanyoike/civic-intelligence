import Link from 'next/link';
import { ArrowLeft, GitBranch, AlertCircle } from 'lucide-react';
import type { Metadata } from 'next';
import { getJSON_ as getJSON } from '@/lib/api';
import { RealityBadge, RealityDisclaimer } from '@/components/reality-labels';

export const metadata: Metadata = {
  title: 'Legal Lineage — Bill to Act to Amendments',
  description: 'Full legal lineage of an Act: origin Bill, parliamentary journey, assent, publication, commencement, regulations, amendments, court decisions, current status.',
};

interface LineageResponse {
  act_id: string;
  lineage: { step: string; title: string; description: string }[];
  disclaimer: string;
}

export default async function LineagePage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  let lineage: LineageResponse | null = null;
  let bffUnreachable = false;
  try {
    lineage = await getJSON<LineageResponse>(`/api/v1/acts/${id}/lineage`);
  } catch (err) {
    // eslint-disable-next-line no-console
    console.warn(`[acts/[id]/lineage] fetch failed for ${id}:`, err);
    bffUnreachable = true;
  }

  if (bffUnreachable || !lineage) {
    return (
      <div className="mx-auto max-w-3xl px-4 py-16 sm:px-6">
        <div className="rounded-xl border border-amber-300 bg-amber-50 p-6 text-amber-900">
          <div className="flex items-center gap-2">
            <AlertCircle className="h-5 w-5 flex-shrink-0" aria-hidden="true" />
            <h1 className="font-serif text-xl font-semibold">
              Lineage unavailable
            </h1>
          </div>
          <p className="mt-2 text-sm">
            The Acts data API is unreachable right now. Please try again in a
            few minutes — every lineage step is fetched live from{' '}
            <span className="font-medium">kenyalaw.org</span> via the Go BFF.
          </p>
          <div className="mt-4 flex flex-wrap gap-2">
            <Link
              href={`/acts/${id}/lineage`}
              className="inline-flex items-center gap-2 rounded-md bg-civic-forest px-4 py-2 text-sm font-semibold text-civic-paper hover:bg-civic-leaf"
            >
              Try again
            </Link>
            <Link
              href={`/acts/${id}`}
              className="inline-flex items-center gap-2 rounded-md border border-amber-300 bg-white px-4 py-2 text-sm font-semibold text-amber-900 hover:bg-amber-100"
            >
              ← Back to act
            </Link>
          </div>
        </div>
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <Link
        href={`/acts/${id}`}
        className="inline-flex items-center gap-1 text-sm text-civic-stone hover:text-civic-forest"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" /> Back to act
      </Link>
      <header className="mt-4 mb-6">
        <div className="flex items-center gap-2">
          <GitBranch className="h-5 w-5 text-civic-forest" aria-hidden="true" />
          <h1 className="font-serif text-2xl font-semibold text-civic-forest">
            Full Legal Lineage
          </h1>
          <RealityBadge kind="FACT" size="md" />
        </div>
        <p className="mt-1 text-sm text-civic-stone">
          Every step in the lineage of this Act, from origin Bill through to
          current legal status. Each step is independently sourced.
        </p>
      </header>
      <div className="mb-6">
        <RealityDisclaimer kind="FACT" />
      </div>
      <ol className="relative space-y-4 border-l-2 border-stone-200 pl-6">
        {lineage.lineage.map((step, idx) => (
          <li key={step.step} className="relative">
            <span
              className="absolute -left-[1.65rem] flex h-4 w-4 items-center justify-center rounded-full border-2 border-white bg-civic-forest"
              aria-hidden="true"
            />
            <div className="rounded-lg border border-stone-200 bg-white p-4">
              <div className="flex items-center justify-between gap-2">
                <h3 className="font-serif text-base font-semibold text-civic-forest">
                  {idx + 1}. {step.title}
                </h3>
                <span className="rounded-full bg-stone-100 px-2 py-0.5 text-[10px] font-semibold uppercase text-stone-600">
                  {step.step.replace(/_/g, ' ')}
                </span>
              </div>
              <p className="mt-1 text-sm text-stone-700">{step.description}</p>
            </div>
          </li>
        ))}
      </ol>
      <p className="mt-6 text-xs text-stone-600">{lineage.disclaimer}</p>
    </div>
  );
}
