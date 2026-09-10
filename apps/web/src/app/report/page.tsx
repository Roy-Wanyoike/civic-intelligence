import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Report an Error',
  description: 'Report a factual error on the Civic Intelligence Platform.',
};

export default function ReportPage() {
  return (
    <div className="mx-auto max-w-2xl px-4 py-10 sm:px-6">
      <h1 className="font-serif text-3xl font-semibold text-civic-forest">Report an Error</h1>
      <p className="mt-2 text-sm text-civic-stone">
        Help us maintain accuracy. Every report goes through a review process.
        We never silently rewrite history — corrections create new auditable versions.
      </p>
      <form className="mt-8 space-y-4">
        <div>
          <label htmlFor="url" className="block text-xs font-medium text-civic-stone">Page URL with the error</label>
          <input type="url" id="url" name="url" placeholder="https://civic-intelligence.vercel.app/bills/..." className="mt-1 w-full rounded-md border border-civic-border bg-civic-paper px-3 py-2 text-sm" />
        </div>
        <div>
          <label htmlFor="description" className="block text-xs font-medium text-civic-stone">What's wrong?</label>
          <textarea id="description" name="description" rows={4} placeholder="Describe the error..." className="mt-1 w-full rounded-md border border-civic-border bg-civic-paper px-3 py-2 text-sm" />
        </div>
        <div>
          <label htmlFor="evidence" className="block text-xs font-medium text-civic-stone">Source URL with correct information (optional)</label>
          <input type="url" id="evidence" name="evidence" placeholder="https://..." className="mt-1 w-full rounded-md border border-civic-border bg-civic-paper px-3 py-2 text-sm" />
        </div>
        <div>
          <label htmlFor="email" className="block text-xs font-medium text-civic-stone">Your email (for updates)</label>
          <input type="email" id="email" name="email" placeholder="you@example.com" className="mt-1 w-full rounded-md border border-civic-border bg-civic-paper px-3 py-2 text-sm" />
        </div>
        <button type="submit" className="w-full rounded-md bg-civic-leaf px-4 py-2 text-sm font-semibold text-white hover:bg-civic-leaf/90">
          Submit Report
        </button>
      </form>
    </div>
  );
}
