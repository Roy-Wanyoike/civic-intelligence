'use client';

import { useState } from 'react';
import { RealityBadge } from '@/components/reality-labels';
import type { Scenario } from '@/lib/scenarios-api';

interface CompareFormProps {
  currentId: string;
  scenarios: Scenario[];
}

export function CompareForm({ currentId, scenarios }: CompareFormProps) {
  const [selected, setSelected] = useState<string[]>([
    currentId,
    ...scenarios.filter((s) => s.id !== currentId).slice(0, 1).map((s) => s.id),
  ]);
  const [result, setResult] = useState<{
    scenarios: Scenario[];
    dimensions: { name: string; description: string }[];
    disclaimer: string;
  } | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [loading, setLoading] = useState(false);

  const toggle = (id: string) => {
    setSelected((cur) =>
      cur.includes(id) ? cur.filter((x) => x !== id) : [...cur, id],
    );
  };

  const run = async () => {
    setLoading(true);
    setError(null);
    try {
      const resp = await fetch('/api/v1/scenarios/compare', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ scenario_ids: selected }),
      });
      if (!resp.ok) {
        throw new Error(`HTTP ${resp.status}`);
      }
      const data = await resp.json();
      setResult(data);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Compare failed');
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="space-y-6">
      <section className="rounded-lg border border-stone-200 bg-white p-5">
        <h2 className="font-serif text-lg font-semibold text-civic-forest">
          Select scenarios to compare
        </h2>
        <ul className="mt-3 space-y-2">
          {scenarios.map((s) => (
            <li key={s.id} className="flex items-start gap-3">
              <input
                id={`cmp-${s.id}`}
                type="checkbox"
                checked={selected.includes(s.id)}
                onChange={() => toggle(s.id)}
                className="mt-1"
              />
              <label htmlFor={`cmp-${s.id}`} className="flex-1">
                <div className="flex items-center gap-2">
                  <span className="text-sm font-semibold text-civic-forest">
                    {s.name}
                  </span>
                  <RealityBadge kind="HYPOTHETICAL" />
                </div>
                <p className="text-xs text-stone-700">{s.description}</p>
              </label>
            </li>
          ))}
        </ul>
        <button
          type="button"
          onClick={run}
          disabled={selected.length < 2 || loading}
          className="mt-4 rounded-lg bg-civic-forest px-4 py-2 text-sm font-semibold text-white transition hover:bg-civic-forest/90 disabled:opacity-50"
        >
          {loading ? 'Comparing...' : 'Compare'}
        </button>
        {error && (
          <div className="mt-3 rounded-lg border border-rose-300 bg-rose-50 p-3 text-sm text-rose-800">
            {error}
          </div>
        )}
      </section>
      {result && (
        <>
          <section className="rounded-lg border border-amber-200 bg-amber-50 p-4">
            <p className="text-sm font-semibold text-amber-900">
              <RealityBadge kind="LIMITATION" /> {result.disclaimer}
            </p>
          </section>
          <section className="rounded-lg border border-stone-200 bg-white p-5">
            <h2 className="font-serif text-lg font-semibold text-civic-forest">
              Comparison dimensions
            </h2>
            <ul className="mt-3 space-y-2">
              {result.dimensions.map((d) => (
                <li key={d.name} className="text-sm">
                  <span className="font-semibold text-civic-forest">{d.name}</span>{' '}
                  — <span className="text-stone-700">{d.description}</span>
                </li>
              ))}
            </ul>
          </section>
          <section className="overflow-x-auto rounded-lg border border-stone-200">
            <table className="min-w-full divide-y divide-stone-200 text-sm">
              <thead className="bg-stone-50">
                <tr>
                  <th className="px-3 py-2 text-left font-semibold text-civic-stone">
                    Dimension
                  </th>
                  {result.scenarios.map((s) => (
                    <th key={s.id} className="px-3 py-2 text-left font-semibold text-civic-forest">
                      {s.name}
                    </th>
                  ))}
                </tr>
              </thead>
              <tbody className="divide-y divide-stone-100 bg-white">
                <tr>
                  <td className="px-3 py-2 font-semibold text-stone-700">Type</td>
                  {result.scenarios.map((s) => (
                    <td key={s.id} className="px-3 py-2 text-stone-800">{s.type}</td>
                  ))}
                </tr>
                <tr>
                  <td className="px-3 py-2 font-semibold text-stone-700">Assumptions</td>
                  {result.scenarios.map((s) => (
                    <td key={s.id} className="px-3 py-2 text-stone-800">
                      {s.assumptions?.length ?? 0}
                    </td>
                  ))}
                </tr>
                <tr>
                  <td className="px-3 py-2 font-semibold text-stone-700">Variables</td>
                  {result.scenarios.map((s) => (
                    <td key={s.id} className="px-3 py-2 text-stone-800">
                      {s.variables?.length ?? 0}
                    </td>
                  ))}
                </tr>
                <tr>
                  <td className="px-3 py-2 font-semibold text-stone-700">Time horizon</td>
                  {result.scenarios.map((s) => (
                    <td key={s.id} className="px-3 py-2 text-stone-800">
                      {s.time_horizon?.start?.slice(0, 10)} → {s.time_horizon?.end?.slice(0, 10)}
                    </td>
                  ))}
                </tr>
                <tr>
                  <td className="px-3 py-2 font-semibold text-stone-700">Model</td>
                  {result.scenarios.map((s) => (
                    <td key={s.id} className="px-3 py-2 text-stone-800">
                      {s.model_id} (v{s.model_version})
                    </td>
                  ))}
                </tr>
                <tr>
                  <td className="px-3 py-2 font-semibold text-stone-700">Status</td>
                  {result.scenarios.map((s) => (
                    <td key={s.id} className="px-3 py-2 text-stone-800">{s.status}</td>
                  ))}
                </tr>
              </tbody>
            </table>
          </section>
        </>
      )}
    </div>
  );
}
