import type { Metadata } from 'next';
import Link from 'next/link';
import { Download, Database } from 'lucide-react';

export const metadata: Metadata = {
  title: 'Civic Datasets',
  description: 'Download structured Kenyan civic data.',
};

export default function DatasetsPage() {
  const datasets = [
    { name: 'Bills', count: '50+', formats: ['CSV', 'JSON'], url: '/api/v1/bills' },
    { name: 'Acts of Parliament', count: '5+', formats: ['CSV', 'JSON'], url: '/api/v1/acts' },
    { name: 'Government Loans', count: '10', formats: ['CSV', 'JSON'], url: '/api/v1/loans' },
    { name: 'Government Grants', count: '5', formats: ['CSV', 'JSON'], url: '/api/v1/grants' },
    { name: 'Parliamentary Terminology', count: '10', formats: ['JSON'], url: '/api/v1/terminology' },
    { name: 'Civic Feed', count: '20+', formats: ['JSON'], url: '/api/v1/feed' },
  ];

  return (
    <div className="mx-auto max-w-3xl px-4 py-10 sm:px-6">
      <h1 className="font-serif text-3xl font-semibold text-civic-forest">Civic Datasets</h1>
      <p className="mt-2 text-sm text-civic-stone">
        Download structured civic data. All datasets include source URLs for evidence.
      </p>
      <ul className="mt-8 space-y-3">
        {datasets.map(ds => (
          <li key={ds.name} className="rounded-lg border border-civic-border bg-civic-paper p-4">
            <div className="flex items-center justify-between">
              <div className="flex items-center gap-2">
                <Database className="h-4 w-4 text-civic-leaf" />
                <span className="font-serif text-base font-semibold text-civic-ink">{ds.name}</span>
                <span className="text-xs text-civic-stone">{ds.count} records</span>
              </div>
              <div className="flex gap-2">
                {ds.formats.map(fmt => (
                  <Link key={fmt} href={ds.url} className="rounded-md border border-civic-border px-2 py-1 text-xs text-civic-leaf hover:bg-civic-leaf/10">
                    {fmt}
                  </Link>
                ))}
              </div>
            </div>
          </li>
        ))}
      </ul>
    </div>
  );
}
