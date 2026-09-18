import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Government Policies',
  description: 'Official Kenyan government policies.',
};

export default function PoliciesPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <h1 className="font-serif text-3xl font-semibold text-civic-forest">Government Policies</h1>
      <p className="mt-2 text-sm text-civic-stone">Official policies from Kenyan government ministries and institutions.</p>
      <div className="mt-8 rounded-lg border border-dashed border-civic-border p-8 text-center text-sm text-civic-stone">
        Policy data loading — connect the Go API for real data.
      </div>
    </div>
  );
}
