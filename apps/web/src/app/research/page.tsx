import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Research Workspace',
  description: 'Save Bills, documents, and searches for research.',
};

export default function ResearchPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <h1 className="font-serif text-3xl font-semibold text-civic-forest">Research Workspace</h1>
      <p className="mt-2 text-sm text-civic-stone">
        Build research collections around civic topics. Save Bills, documents, searches, and notes.
      </p>
      <div className="mt-8 rounded-lg border border-dashed border-civic-border p-8 text-center text-sm text-civic-stone">
        Research workspaces coming soon. Sign in to create your first workspace.
      </div>
    </div>
  );
}
