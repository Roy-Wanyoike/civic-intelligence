import type { Metadata } from 'next';
import { ResearchWorkspace } from './research-workspace';

export const metadata: Metadata = {
  title: 'Research Workspace',
  description:
    'Create research missions around civic topics. Save Bills, documents, and searches; run the 7-agent Civic Agent Network against your evidence base.',
};

export const dynamic = 'force-dynamic';

const AGENTS = [
  {
    id: 'researcher',
    name: 'Researcher',
    description:
      'Discovers primary and secondary sources for the research question. Returns a ranked list of documents with authority levels.',
    icon: 'FileSearch',
    color: 'text-civic-leaf',
  },
  {
    id: 'historical-comparator',
    name: 'Historical Comparator',
    description:
      'Surfaces analogous past Bills, Acts, or fiscal events so the question can be framed against historical precedent.',
    icon: 'History',
    color: 'text-civic-acacia',
  },
  {
    id: 'assumption-auditor',
    name: 'Assumption Auditor',
    description:
      'Extracts the explicit and implicit assumptions baked into the question and tags each with a RealityBadge kind="ASSUMPTION".',
    icon: 'AlertTriangle',
    color: 'text-amber-600',
  },
  {
    id: 'model-auditor',
    name: 'Model Auditor',
    description:
      'Audits any underlying model (fiscal, demographic, economic) for hidden coefficients and outputs a model card.',
    icon: 'Cpu',
    color: 'text-violet-600',
  },
  {
    id: 'sensitivity-analyst',
    name: 'Sensitivity Analyst',
    description:
      'Runs one-at-a-time sensitivity sweeps over the assumptions and reports which inputs dominate the output range.',
    icon: 'Activity',
    color: 'text-sky-600',
  },
  {
    id: 'evidence-auditor',
    name: 'Evidence Auditor',
    description:
      'Validates the evidence chain for each claim (Source → Document → Evidence → Fact → Claim → Explanation) and flags gaps.',
    icon: 'ShieldCheck',
    color: 'text-emerald-600',
  },
  {
    id: 'scenario-synthesizer',
    name: 'Scenario Synthesizer',
    description:
      'Composes the agent outputs into alternative scenarios (baseline / optimistic / pessimistic), each tagged HYPOTHETICAL.',
    icon: 'Sparkles',
    color: 'text-fuchsia-600',
  },
] as const;

export default function ResearchPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header className="mb-8">
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">
          Research Workspace
        </h1>
        <p className="mt-2 text-sm text-civic-stone">
          Create research missions around civic topics. Save Bills, documents,
          searches, and notes; orchestrate the 7-agent Civic Agent Network
          against your evidence base. Each mission is a self-contained research
          artefact you can return to and share.
        </p>
      </header>

      <ResearchWorkspace agents={AGENTS} />
    </div>
  );
}
