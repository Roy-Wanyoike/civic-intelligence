'use client';

import { useState } from 'react';
import { useRouter } from 'next/navigation';
import { RealityBadge } from '@/components/reality-labels';

interface Step {
  title: string;
  description: string;
}

const STEPS: Step[] = [
  { title: 'Question', description: 'What civic outcome are you exploring?' },
  { title: 'Baseline', description: 'What is the observed starting point?' },
  { title: 'Assumptions', description: 'What inputs are you hypothesising?' },
  { title: 'Evidence', description: 'What authoritative material supports each assumption?' },
  { title: 'Scenario', description: 'What model + variables apply?' },
  { title: 'Simulation', description: 'Run the scenario.' },
  { title: 'Results', description: 'Inspect the modeled outcome.' },
  { title: 'Limitations', description: 'What does the model NOT capture?' },
];

export function NewScenarioForm() {
  const router = useRouter();
  const [stepIdx, setStepIdx] = useState(0);
  const [name, setName] = useState('');
  const [question, setQuestion] = useState('');
  const [baseline, setBaseline] = useState('');
  const [assumptions, setAssumptions] = useState<string>('');
  const [error, setError] = useState<string | null>(null);

  const next = () => setStepIdx((i) => Math.min(i + 1, STEPS.length - 1));
  const back = () => setStepIdx((i) => Math.max(i - 1, 0));

  const submit = async () => {
    setError(null);
    try {
      const resp = await fetch('/api/v1/scenarios', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: name || 'Untitled Scenario',
          description: question,
          type: 'POLICY',
          jurisdiction: 'KE',
          baseline: { description: baseline || 'Baseline observed.', reality_layer: 'OBSERVED' },
          assumptions: assumptions
            ? assumptions.split('\n').filter(Boolean).map((s, i) => ({
                id: `asm-${i + 1}`,
                statement: s.trim(),
                assumption_type: 'USER_DEFINED',
                confidence: 'MEDIUM',
              }))
            : [
                {
                  id: 'asm-default',
                  statement: 'A user-defined assumption underpins this scenario.',
                  assumption_type: 'USER_DEFINED',
                  confidence: 'MEDIUM',
                },
              ],
          variables: [
            {
              id: 'var-input',
              name: 'input_value',
              type: 'DECIMAL',
              unit: 'unit',
              value: 10,
              assumption_status: 'USER_DEFINED',
            },
          ],
          time_horizon: {
            start: '2025-01-01T00:00:00Z',
            end: '2026-01-01T00:00:00Z',
            duration: 'P1Y',
          },
          model_id: 'model-simple-deterministic',
          model_version: '1.0.0',
          status: 'READY',
        }),
      });
      if (!resp.ok) {
        const txt = await resp.text();
        throw new Error(`HTTP ${resp.status}: ${txt}`);
      }
      const created = await resp.json();
      router.push(`/scenarios/${created.id}`);
    } catch (e) {
      setError(e instanceof Error ? e.message : 'Failed to create scenario');
    }
  };

  return (
    <div className="rounded-lg border border-stone-200 bg-white p-6">
      <ol className="mb-6 flex flex-wrap gap-2">
        {STEPS.map((s, i) => (
          <li key={s.title} className="flex items-center gap-2">
            <button
              type="button"
              onClick={() => setStepIdx(i)}
              className={`rounded-full px-3 py-1 text-xs font-semibold transition ${
                i === stepIdx
                  ? 'bg-civic-forest text-white'
                  : 'bg-stone-100 text-stone-700 hover:bg-stone-200'
              }`}
              aria-current={i === stepIdx ? 'step' : undefined}
            >
              {i + 1}. {s.title}
            </button>
            {i < STEPS.length - 1 && (
              <span aria-hidden="true" className="text-stone-300">→</span>
            )}
          </li>
        ))}
      </ol>

      <p className="mb-4 text-sm text-stone-700">{STEPS[stepIdx].description}</p>

      {stepIdx === 0 && (
        <div className="space-y-4">
          <div>
            <label htmlFor="scenario-name" className="block text-sm font-semibold text-stone-800">
              Scenario name
            </label>
            <input
              id="scenario-name"
              type="text"
              value={name}
              onChange={(e) => setName(e.target.value)}
              placeholder="e.g. What if public transport fares were reduced by 20%?"
              className="mt-1 w-full rounded-lg border border-stone-300 px-3 py-2 text-sm"
            />
          </div>
          <div>
            <label htmlFor="scenario-question" className="block text-sm font-semibold text-stone-800">
              Question
            </label>
            <textarea
              id="scenario-question"
              value={question}
              onChange={(e) => setQuestion(e.target.value)}
              placeholder="Describe the civic question you are exploring."
              rows={3}
              className="mt-1 w-full rounded-lg border border-stone-300 px-3 py-2 text-sm"
            />
          </div>
        </div>
      )}

      {stepIdx === 1 && (
        <div>
          <label htmlFor="baseline" className="block text-sm font-semibold text-stone-800">
            Baseline <RealityBadge kind="FACT" /> — the observed starting point
          </label>
          <textarea
            id="baseline"
            value={baseline}
            onChange={(e) => setBaseline(e.target.value)}
            placeholder="e.g. Public transport fares in Nairobi were last revised on 1 July 2024."
            rows={4}
            className="mt-1 w-full rounded-lg border border-stone-300 px-3 py-2 text-sm"
          />
          <p className="mt-2 text-xs text-stone-600">
            The baseline is OBSERVED reality. Do not invent baseline facts.
          </p>
        </div>
      )}

      {stepIdx === 2 && (
        <div>
          <label htmlFor="assumptions" className="block text-sm font-semibold text-stone-800">
            Assumptions <RealityBadge kind="ASSUMPTION" /> — one per line
          </label>
          <textarea
            id="assumptions"
            value={assumptions}
            onChange={(e) => setAssumptions(e.target.value)}
            placeholder={'The fare reduction is implemented on 1 January 2026.\nThe reduction is 20% across all routes.'}
            rows={5}
            className="mt-1 w-full rounded-lg border border-stone-300 px-3 py-2 text-sm"
          />
          <p className="mt-2 text-xs text-stone-600">
            Every assumption must be explicit. If you do not know a value,
            mark it <RealityBadge kind="UNKNOWN" />.
          </p>
        </div>
      )}

      {stepIdx === 3 && (
        <div className="space-y-3 text-sm text-stone-700">
          <p>
            <RealityBadge kind="EVIDENCE" /> Evidence grounds your assumptions in
            authoritative material. After creating the scenario, attach
            evidence per assumption on the Assumptions tab.
          </p>
          <p>
            The platform rejects externally sourced assumptions
            (OBSERVED_INPUT, HISTORICAL_REFERENCE, ESTIMATE) that have no
            evidence reference.
          </p>
        </div>
      )}

      {stepIdx === 4 && (
        <div className="space-y-3 text-sm text-stone-700">
          <p>
            <RealityBadge kind="SIMULATION" /> A scenario is paired with a model
            and a set of typed variables. The default model used here is the
            &ldquo;Simple Deterministic Doubler&rdquo; — a trivial model that
            doubles its input.
          </p>
          <p>
            You can replace this with a different model after the scenario is
            created.
          </p>
        </div>
      )}

      {stepIdx === 5 && (
        <div className="space-y-3 text-sm text-stone-700">
          <p>
            <RealityBadge kind="SIMULATION" /> Once the scenario is READY, you
            can run it. The platform never holds long requests open; long
            simulations execute asynchronously and results appear on the
            Results tab.
          </p>
        </div>
      )}

      {stepIdx === 6 && (
        <div className="space-y-3 text-sm text-stone-700">
          <p>
            <RealityBadge kind="SIMULATION" /> Results are SIMULATED. Every
            result carries the model version, engine version, input hash, and
            random seed used so it can be reproduced.
          </p>
        </div>
      )}

      {stepIdx === 7 && (
        <div className="space-y-3 text-sm text-stone-700">
          <p>
            <RealityBadge kind="LIMITATION" /> Every model declares its
            limitations. The Simple Deterministic Doubler has known
            limitations: it does not produce uncertainty, and it has no
            real-world interpretation.
          </p>
          <p>
            Always inspect the Methodology tab before drawing conclusions from
            a simulation.
          </p>
        </div>
      )}

      {error && (
        <div className="mt-4 rounded-lg border border-rose-300 bg-rose-50 p-3 text-sm text-rose-800">
          {error}
        </div>
      )}

      <div className="mt-6 flex items-center justify-between">
        <button
          type="button"
          onClick={back}
          disabled={stepIdx === 0}
          className="rounded-lg border border-stone-300 px-4 py-2 text-sm font-semibold text-stone-700 transition hover:bg-stone-50 disabled:opacity-50"
        >
          Back
        </button>
        {stepIdx < STEPS.length - 1 ? (
          <button
            type="button"
            onClick={next}
            className="rounded-lg bg-civic-forest px-4 py-2 text-sm font-semibold text-white transition hover:bg-civic-forest/90"
          >
            Next
          </button>
        ) : (
          <button
            type="button"
            onClick={submit}
            className="rounded-lg bg-civic-forest px-4 py-2 text-sm font-semibold text-white transition hover:bg-civic-forest/90"
          >
            Create scenario
          </button>
        )}
      </div>
    </div>
  );
}
