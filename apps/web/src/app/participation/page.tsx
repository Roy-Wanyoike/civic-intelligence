import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Public Participation',
  description: 'Official public participation opportunities for Kenyan Bills.',
};

export default function ParticipationPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Public Participation</h1>
        <p className="mt-2 text-sm text-civic-stone">
          When official public participation is open, it will be listed here with deadlines and official submission channels.
        </p>
      </header>
      <div className="mt-8 rounded-lg border border-dashed border-civic-border p-8 text-center text-sm text-civic-stone">
        No open participation opportunities. Check back when Parliament publishes new calls for submissions.
      </div>
    </div>
  );
}
