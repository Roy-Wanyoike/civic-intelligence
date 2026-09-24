import Link from 'next/link';
import { ArrowLeft, Plus } from 'lucide-react';
import type { Metadata } from 'next';
import { NewPetitionForm } from './new-petition-form';

export const metadata: Metadata = {
  title: 'New Petition — Public Participation',
  description: 'Create a public-participation e-petition.',
};

export default function NewPetitionPage() {
  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <Link
        href="/participation"
        className="inline-flex items-center gap-1 text-sm text-civic-stone hover:text-civic-forest"
      >
        <ArrowLeft className="h-4 w-4" aria-hidden="true" /> Back to petitions
      </Link>
      <header className="mt-4 mb-6">
        <div className="flex items-center gap-2 text-xs uppercase tracking-wide text-civic-stone">
          <Plus className="h-4 w-4" aria-hidden="true" />
          <span>Create a Petition</span>
        </div>
        <h1 className="mt-2 font-serif text-3xl font-semibold text-civic-forest">
          New e-Petition
        </h1>
        <p className="mt-2 text-sm text-civic-stone">
          Draft a title, description, and target signature threshold. The
          platform never derives a political-performance score from the
          signature count — the citizen interprets the result, the
          platform does not.
        </p>
      </header>
      <NewPetitionForm />
    </div>
  );
}
