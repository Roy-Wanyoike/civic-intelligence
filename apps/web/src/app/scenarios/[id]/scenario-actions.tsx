'use client';

import { useState } from 'react';
import { Play, RefreshCw } from 'lucide-react';
import { RealityBadge } from '@/components/reality-labels';

interface ScenarioActionsProps {
  scenarioId: string;
  status: string;
}

export function ScenarioActions({ scenarioId, status }: ScenarioActionsProps) {
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);

  const canRun = status === 'READY' || status === 'COMPLETED';

  const handleRun = async () => {
    setRunning(true);
    setError(null);
    try {
      const resp = await fetch(
        `/api/v1/scenarios/${scenarioId}?action=run&seed=42`,
        { method: 'POST' },
      );
      if (!resp.ok) {
        const txt = await resp.text();
        throw new Error(`HTTP ${resp.status}: ${txt}`);
      }
      const data = await resp.json();
      setResult(JSON.stringify(data.run ?? data, null, 2));
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Run failed');
    } finally {
      setRunning(false);
    }
  };

  return (
    <div className="rounded-lg border border-violet-200 bg-violet-50 p-5">
      <div className="flex items-center gap-2">
        <h3 className="font-serif text-base font-semibold text-violet-900">
          Run this scenario
        </h3>
        <RealityBadge kind="SIMULATION" />
      </div>
      <p className="mt-1 text-sm text-violet-900">
        Execute the scenario with the configured model. The platform persists
        the input hash, random seed, model version, and engine version so
        every run is reproducible.
      </p>
      <div className="mt-3 flex flex-wrap gap-2">
        <button
          type="button"
          onClick={handleRun}
          disabled={!canRun || running}
          className="inline-flex items-center gap-2 rounded-lg bg-violet-700 px-4 py-2 text-sm font-semibold text-white transition hover:bg-violet-800 disabled:opacity-50"
        >
          <Play className="h-4 w-4" aria-hidden="true" />
          {running ? 'Running...' : 'Run'}
        </button>
        <a
          href={`/scenarios/${scenarioId}/results`}
          className="inline-flex items-center gap-2 rounded-lg border border-violet-300 bg-white px-4 py-2 text-sm font-semibold text-violet-900 transition hover:bg-violet-50"
        >
          <RefreshCw className="h-4 w-4" aria-hidden="true" />
          View results
        </a>
      </div>
      {!canRun && (
        <p className="mt-2 text-xs text-violet-800">
          Scenario must be READY before running. Current status: {status}.
        </p>
      )}
      {error && (
        <div className="mt-3 rounded-lg border border-rose-300 bg-rose-50 p-3 text-sm text-rose-800">
          {error}
        </div>
      )}
      {result && (
        <details className="mt-3">
          <summary className="cursor-pointer text-xs font-semibold text-violet-900">
            View run output (SIMULATED)
          </summary>
          <pre className="mt-2 max-h-96 overflow-auto rounded-lg bg-white p-3 text-xs text-stone-800">
            {result}
          </pre>
        </details>
      )}
    </div>
  );
}
