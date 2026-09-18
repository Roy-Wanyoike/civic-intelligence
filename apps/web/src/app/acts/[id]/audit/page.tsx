import Link from 'next/link';
import { notFound } from 'next/navigation';
import { ArrowLeft, CheckCircle2, XCircle, AlertTriangle } from 'lucide-react';
import type { Metadata } from 'next';
import { getJSON_ as getJSON } from '@/lib/api';
import { RealityBadge, RealityDisclaimer } from '@/components/reality-labels';

export const metadata: Metadata = {
  title: 'Audit an Act — Post-Assent Lifecycle',
  description: 'Full lifecycle audit of an Act: assent, publication, commencement, regulations, court history, amendments, current status, data gaps.',
};

interface ActAuditResponse {
  act_id: string;
  assent_recorded: boolean;
  act_published: boolean;
  act_number: string;
  commencement_notice: boolean;
  regulations_issued: boolean;
  amended: boolean;
  court_challenged: boolean;
  judicial_decision: boolean;
  repealed: boolean;
  current_status: string;
  audit_statuses: string[];
  data_gaps: string[];
  disclaimer: string;
}

function StatusRow({ label, ok, note }: { label: string; ok: boolean | null; note?: string }) {
  return (
    <li className="flex items-start gap-3 py-2">
      {ok === true ? (
        <CheckCircle2 className="mt-0.5 h-4 w-4 flex-shrink-0 text-emerald-600" aria-hidden="true" />
      ) : ok === false ? (
        <XCircle className="mt-0.5 h-4 w-4 flex-shrink-0 text-rose-600" aria-hidden="true" />
      ) : (
        <AlertTriangle className="mt-0.5 h-4 w-4 flex-shrink-0 text-amber-600" aria-hidden="true" />
      )}
      <div>
        <p className="text-sm font-semibold text-stone-800">{label}</p>
        {note && <p className="text-xs text-stone-600">{note}</p>}
      </div>
    </li>
  );
}

export default async function ActAuditPage({
  params,
}: {
  params: Promise<{ id: string }>;
}) {
  const { id } = await params;
  let audit: ActAuditResponse | null = null;
  try {
    audit = await getJSON<ActAuditResponse>(`/api/v1/acts/${id}/audit`);
  } catch {
    notFound();
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
          <h1 className="font-serif text-2xl font-semibold text-civic-forest">
            Audit an Act
          </h1>
          <RealityBadge kind="FACT" size="md" />
        </div>
        <p className="mt-1 text-sm text-civic-stone">
          Full lifecycle audit for Act: {audit.act_number}. Missing records
          are reported as <strong>NOT_VERIFIED</strong> — never as
          &ldquo;never happened&rdquo;.
        </p>
      </header>
      <div className="mb-6">
        <RealityDisclaimer kind="FACT" />
      </div>
      <section className="rounded-lg border border-stone-200 bg-white p-5">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">
          Post-Assent Audit
        </h2>
        <ul className="mt-3 divide-y divide-stone-100">
          <StatusRow label="Presidential Assent recorded" ok={audit.assent_recorded} />
          <StatusRow label="Act published in Gazette" ok={audit.act_published} />
          <StatusRow
            label="Commencement notice found"
            ok={audit.commencement_notice}
            note={audit.commencement_notice ? undefined : 'NOT_VERIFIED — absence of record is not absence of event'}
          />
          <StatusRow label="Regulations issued" ok={audit.regulations_issued} />
          <StatusRow label="Court challenges tracked" ok={audit.court_challenged} />
          <StatusRow label="Judicial decisions tracked" ok={audit.judicial_decision} />
          <StatusRow label="Amendments tracked" ok={audit.amended} />
          <StatusRow label="Repeal status tracked" ok={audit.repealed} />
        </ul>
      </section>
      <section className="mt-4 rounded-lg border border-stone-200 bg-white p-5">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">
          Audit Statuses
        </h2>
        <ul className="mt-2 flex flex-wrap gap-2">
          {audit.audit_statuses.map((s) => (
            <li
              key={s}
              className="rounded-full bg-stone-100 px-3 py-1 text-xs font-semibold text-stone-700"
            >
              {s}
            </li>
          ))}
        </ul>
      </section>
      {audit.data_gaps.length > 0 && (
        <section className="mt-4 rounded-lg border border-amber-200 bg-amber-50 p-5">
          <div className="flex items-center gap-2">
            <h2 className="font-serif text-lg font-semibold text-amber-900">
              Data Gaps
            </h2>
            <RealityBadge kind="UNKNOWN" />
          </div>
          <ul className="mt-2 list-disc pl-5 text-sm text-amber-900">
            {audit.data_gaps.map((g) => (
              <li key={g}>{g}</li>
            ))}
          </ul>
          <p className="mt-3 text-xs text-amber-800">
            Data gaps are NOT assertions that nothing happened. The platform
            distinguishes NOT_VERIFIED from negative findings.
          </p>
        </section>
      )}
      <p className="mt-4 text-xs text-stone-600">{audit.disclaimer}</p>
    </div>
  );
}
