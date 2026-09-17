// New scenario builder. Phase 18 §16, §18.
//
// Guides the user through: Question → Baseline → Assumptions → Evidence →
// Scenario → Simulation → Results → Limitations.
import Link from 'next/link';
import { ArrowLeft, Sparkles } from 'lucide-react';
import type { Metadata } from 'next';
import { RealityDisclaimer } from '@/components/reality-labels';
import { NewScenarioForm } from './new-scenario-form';

export const metadata: Metadata = {
  title: 'New Scenario — What If?',
  description: 'Build a hypothetical civic scenario step by step.',
};

export default function NewScenarioPage() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <Link
        href="/scenarios"
        className="inline-flex items-center gap-1 text-sm text-civic-stone hover:text-civic-forest"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" /> Back to scenarios
      </Link>
      <header className="mt-4 mb-6">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Sparkles className="h-4 w-4" aria-hidden="true" />
          <span>Scenario Builder</span>
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
          Build a &ldquo;What If?&rdquo;
        </h1>
        <p className="mt-2 text-sm text-civic-stone">
          Walk through every component of a scenario. The platform never
          fabricates evidence or assumptions; if you do not know a value,
          mark it <strong>UNKNOWN</strong> rather than guessing.
        </p>
      </header>
      <div className="mb-6">
        <RealityDisclaimer kind="HYPOTHETICAL" />
      </div>
      <NewScenarioForm />
    </div>
  );
}
