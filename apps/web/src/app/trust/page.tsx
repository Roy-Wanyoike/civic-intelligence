import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Trust & Evidence',
  description: 'How Civic Intelligence ensures every claim is traceable to an official source.',
};

export default function TrustPage() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <h1 className="font-serif text-3xl font-semibold text-civic-forest">Trust & Evidence</h1>
      <p className="mt-2 text-sm text-civic-stone">
        Every factual claim on this platform is traceable to an authoritative source.
      </p>
      <div className="mt-8 space-y-6">
        <div className="rounded-lg border border-civic-border bg-civic-paper p-6">
          <h2 className="font-serif text-lg font-semibold text-civic-ink">Evidence Chain</h2>
          <p className="mt-2 text-sm text-civic-stone">
            Every claim follows this chain:
          </p>
          <pre className="mt-3 rounded-md bg-civic-mist p-3 text-xs text-civic-ink overflow-x-auto">
{`Claim
  ↓
Evidence
  ↓
Document
  ↓
Snapshot
  ↓
Official Source
  ↓
Date Retrieved
  ↓
Content Hash`}
          </pre>
        </div>
        <div className="rounded-lg border border-civic-border bg-civic-paper p-6">
          <h2 className="font-serif text-lg font-semibold text-civic-ink">Source Authority Levels</h2>
          <ul className="mt-3 space-y-2 text-sm text-civic-stone">
            <li><strong className="text-civic-leaf">PRIMARY_OFFICIAL</strong> — Official government source (parliament.go.ke, president.go.ke)</li>
            <li><strong className="text-civic-acacia">OFFICIAL_REPOSITORY</strong> — Official legal repository (kenyalaw.org)</li>
            <li><strong className="text-civic-stone">SECONDARY_VERIFIED</strong> — Verified secondary sources (news with official references)</li>
            <li><strong className="text-civic-stone">UNVERIFIED</strong> — Unverified sources (not used for canonical facts)</li>
          </ul>
        </div>
        <div className="rounded-lg border border-civic-border bg-civic-paper p-6">
          <h2 className="font-serif text-lg font-semibold text-civic-ink">Report an Error</h2>
          <p className="mt-2 text-sm text-civic-stone">
            Found something wrong? Every page has a "Report an error" link.
            Corrections go through a review process and create auditable versions.
            We never silently rewrite history.
          </p>
        </div>
      </div>
    </div>
  );
}
