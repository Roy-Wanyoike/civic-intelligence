import type { Metadata } from 'next';

export const metadata: Metadata = {
  title: 'Developer API',
  description: 'Build on the Civic Intelligence Platform API.',
};

export default function DevelopersPage() {
  return (
    <div className="mx-auto max-w-5xl px-4 py-10 sm:px-6">
      <h1 className="font-serif text-3xl font-semibold text-civic-forest">Developer API</h1>
      <p className="mt-2 text-sm text-civic-stone">
        Access verified Kenyan civic data through our REST API.
      </p>
      <div className="mt-8 rounded-lg border border-civic-border bg-civic-paper p-6">
        <h2 className="font-serif text-lg font-semibold text-civic-ink">Available Endpoints</h2>
        <ul className="mt-4 space-y-2 text-sm font-mono text-civic-ink">
          <li>GET /api/v1/bills — List Bills</li>
          <li>GET /api/v1/bills/&#123;id&#125; — Bill detail</li>
          <li>GET /api/v1/bills/&#123;id&#125;/timeline — Bill timeline</li>
          <li>GET /api/v1/bills/&#123;id&#125;/summary — AI summary</li>
          <li>GET /api/v1/trending — Trending bills</li>
          <li>GET /api/v1/terminology/&#123;term&#125; — Parliamentary terms</li>
          <li>GET /api/v1/acts — Acts of Parliament</li>
          <li>GET /api/v1/loans — Government loans</li>
          <li>GET /api/v1/grants — Government grants</li>
          <li>GET /api/v1/feed — Civic activity feed</li>
          <li>GET /api/v1/search?q=... — Search</li>
        </ul>
      </div>
      <div className="mt-4 rounded-lg border border-dashed border-civic-border p-6 text-center text-sm text-civic-stone">
        API keys, rate limiting, and SDKs coming soon.
      </div>
    </div>
  );
}
