import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Kenya Gazette',
  description: 'Official Kenya Gazette notices.',
};

export default function GazettePage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <header>
        <h1 className="font-serif text-3xl font-semibold text-civic-forest">Kenya Gazette</h1>
        <p className="mt-2 text-sm text-civic-stone">
          Official government notices from the Kenya Gazette.
        </p>
      </header>
      <div className="mt-8 rounded-lg border border-dashed border-civic-border p-8 text-center text-sm text-civic-stone">
        Gazette notice ingestion is pending (issue #30).
      </div>
    </div>
  );
}
