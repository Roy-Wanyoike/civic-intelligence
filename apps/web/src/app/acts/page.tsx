import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Acts of Parliament',
  description: 'Browse enacted Acts of Parliament in Kenya.',
};

export default function ActsPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Acts of Parliament</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Enacted legislation from Kenya Law. Every Act links to its official source.
        </p>
      </header>
      <div className="mt-8 rounded-lg border border-dashed border-civic-border p-8 text-center text-sm text-civic-stone">
        Act data ingestion is pending. Bills that receive Presidential Assent become Acts and will be listed here.
      </div>
    </div>
  );
}
