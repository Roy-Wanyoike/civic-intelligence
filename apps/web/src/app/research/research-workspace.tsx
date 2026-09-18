'use client';

import { useEffect, useState } from 'react';
import {
  FileSearch,
  History,
  AlertTriangle,
  Cpu,
  Activity,
  ShieldCheck,
  Sparkles,
  Loader2,
  Plus,
  Trash2,
  ArrowRight,
} from 'lucide-react';
import { RealityBadge } from '@/components/reality-labels';
import Link from 'next/link';

/**
 * ResearchWorkspace — client-side research interface.
 *
 * Missions are stored in localStorage under `civic.research.missions` so the
 * page works end-to-end without a backend; the backend agent (issue #14) will
 * wire this to /api/v1/research/missions when it lands. The 7-agent Civic
 * Agent Network cards are rendered as a catalogue; each "Run this agent"
 * button is intentionally non-functional and marked with RealityBadge
 * kind="LIMITATION" "Coming soon" — the agent runtime is pending (issue #14).
 *
 * Shape of a mission (stored in localStorage):
 *   { id, question, createdAt }
 * The list is re-rendered on mount via a useEffect that hydrates from
 * localStorage, so SSR + hydration stay deterministic.
 */

type AgentIcon =
  | 'FileSearch'
  | 'History'
  | 'AlertTriangle'
  | 'Cpu'
  | 'Activity'
  | 'ShieldCheck'
  | 'Sparkles';

const ICONS: Record<AgentIcon, typeof FileSearch> = {
  FileSearch,
  History,
  AlertTriangle,
  Cpu,
  Activity,
  ShieldCheck,
  Sparkles,
};

export interface ResearchAgent {
  id: string;
  name: string;
  description: string;
  icon: AgentIcon;
  color: string;
}

export interface ResearchMission {
  id: string;
  question: string;
  createdAt: string;
}

const STORAGE_KEY = 'civic.research.missions';

function loadMissions(): ResearchMission[] {
  if (typeof window === 'undefined') return [];
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    if (!raw) return [];
    const arr = JSON.parse(raw) as ResearchMission[];
    if (!Array.isArray(arr)) return [];
    return arr;
  } catch {
    return [];
  }
}

function saveMissions(missions: ResearchMission[]): void {
  if (typeof window === 'undefined') return;
  try {
    window.localStorage.setItem(STORAGE_KEY, JSON.stringify(missions));
  } catch {
    /* localStorage may be unavailable in private mode */
  }
}

function newId(): string {
  // Deterministic-ish ID without pulling in a UUID dep.
  return `mission-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 8)}`;
}

export function ResearchWorkspace({ agents }: { agents: readonly ResearchAgent[] }) {
  const [missions, setMissions] = useState<ResearchMission[]>([]);
  const [question, setQuestion] = useState('');
  const [runningAgent, setRunningAgent] = useState<string | null>(null);
  const [hydrated, setHydrated] = useState(false);

  useEffect(() => {
    setMissions(loadMissions());
    setHydrated(true);
  }, []);

  function createMission(q?: string) {
    const text = (q ?? question).trim();
    if (!text) return;
    const mission: ResearchMission = {
      id: newId(),
      question: text,
      createdAt: new Date().toISOString(),
    };
    const next = [mission, ...missions];
    setMissions(next);
    saveMissions(next);
    if (q) setQuestion(q);
  }

  function deleteMission(id: string) {
    const next = missions.filter((m) => m.id !== id);
    setMissions(next);
    saveMissions(next);
  }

  function runAgent(agentId: string) {
    // Non-functional for now — the agent runtime is pending (issue #14).
    // Show a brief "running" indicator so the click registers, then no-op.
    setRunningAgent(agentId);
    setTimeout(() => setRunningAgent(null), 900);
  }

  return (
    <div className="space-y-10">
      {/* Mission creation */}
      <section aria-labelledby="create-heading">
        <h2 id="create-heading" className="font-serif text-xl font-semibold text-civic-forest">
          Start a research mission
        </h2>
        <p className="mt-1 text-sm text-civic-stone">
          Pose a research question. The Civic Agent Network will collect
          sources, audit assumptions, and synthesise scenarios. Missions are
          stored locally in your browser until the backend agent is wired
          (issue #14).
        </p>
        <div className="mt-4 rounded-xl border border-civic-border bg-civic-paper p-4 shadow-sm">
          <label htmlFor="question" className="sr-only">
            Research question
          </label>
          <textarea
            id="question"
            value={question}
            onChange={(e) => setQuestion(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === 'Enter' && (e.metaKey || e.ctrlKey)) {
                e.preventDefault();
                createMission();
              }
            }}
            rows={3}
            placeholder="e.g. What would the fiscal impact be of capping public debt at 50% of GDP?"
            className="w-full resize-none rounded-md border border-civic-border bg-civic-mist px-4 py-3 text-sm focus:outline-none focus:border-civic-leaf"
          />
          <div className="mt-3 flex items-center justify-between gap-2">
            <p className="text-xs text-civic-stone">
              <kbd className="rounded bg-civic-mist px-1">⌘</kbd>+
              <kbd className="rounded bg-civic-mist px-1">Enter</kbd> to create.
            </p>
            <button
              type="button"
              onClick={() => createMission()}
              disabled={!question.trim()}
              className="inline-flex items-center gap-1.5 rounded-md bg-civic-leaf px-4 py-2 text-sm font-semibold text-white transition hover:bg-civic-leaf/90 disabled:cursor-not-allowed disabled:opacity-50"
            >
              <Plus className="h-4 w-4" aria-hidden="true" />
              Create mission
            </button>
          </div>
        </div>
      </section>

      {/* Recent missions */}
      <section aria-labelledby="missions-heading">
        <h2 id="missions-heading" className="font-serif text-xl font-semibold text-civic-forest">
          Recent missions
        </h2>
        {!hydrated ? (
          <p className="mt-2 text-sm text-civic-stone">Loading…</p>
        ) : missions.length === 0 ? (
          <p className="mt-2 rounded-lg border border-dashed border-civic-border p-6 text-center text-sm text-civic-stone">
            No missions yet. Create your first mission above — it will be saved
            in your browser.
          </p>
        ) : (
          <ul className="mt-3 space-y-2">
            {missions.map((m) => (
              <li
                key={m.id}
                className="flex items-start justify-between gap-3 rounded-lg border border-civic-border bg-civic-paper p-4"
              >
                <div className="min-w-0 flex-1">
                  <p className="text-sm text-civic-ink">{m.question}</p>
                  <p className="mt-1 text-xs text-civic-stone">
                    Created {new Date(m.createdAt).toLocaleString()} · local only
                  </p>
                </div>
                <div className="flex flex-shrink-0 items-center gap-2">
                  <Link
                    href={`/search?q=${encodeURIComponent(m.question)}`}
                    className="inline-flex items-center gap-1 rounded-md border border-civic-border px-2 py-1 text-xs text-civic-ink hover:border-civic-leaf"
                  >
                    Search <ArrowRight className="h-3 w-3" aria-hidden="true" />
                  </Link>
                  <button
                    type="button"
                    onClick={() => deleteMission(m.id)}
                    aria-label="Delete mission"
                    className="inline-flex h-7 w-7 items-center justify-center rounded-md text-civic-stone hover:bg-civic-mist hover:text-civic-clay"
                  >
                    <Trash2 className="h-3.5 w-3.5" aria-hidden="true" />
                  </button>
                </div>
              </li>
            ))}
          </ul>
        )}
      </section>

      {/* Agent catalogue */}
      <section aria-labelledby="agents-heading">
        <div className="flex items-center gap-2">
          <h2 id="agents-heading" className="font-serif text-xl font-semibold text-civic-forest">
            Civic Agent Network
          </h2>
          <RealityBadge kind="LIMITATION">Coming soon</RealityBadge>
        </div>
        <p className="mt-1 text-sm text-civic-stone">
          Seven specialist agents collaborate on each mission. The runtime is
          pending the agent orchestration service (issue #14). Each agent card
          below describes what it will do once wired.
        </p>
        <ul className="mt-4 grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {agents.map((a) => {
            const Icon = ICONS[a.icon] ?? FileSearch;
            const isRunning = runningAgent === a.id;
            return (
              <li
                key={a.id}
                className="flex flex-col rounded-xl border border-civic-border bg-civic-paper p-4"
              >
                <div className="flex items-center gap-2">
                  <Icon className={`h-5 w-5 ${a.color}`} aria-hidden="true" />
                  <h3 className="font-serif text-base font-semibold text-civic-ink">
                    {a.name}
                  </h3>
                </div>
                <p className="mt-2 flex-1 text-xs text-civic-stone">
                  {a.description}
                </p>
                <div className="mt-3 flex items-center justify-between">
                  <RealityBadge kind="LIMITATION" />
                  <button
                    type="button"
                    onClick={() => runAgent(a.id)}
                    disabled={isRunning}
                    className="inline-flex items-center gap-1 rounded-md border border-civic-border px-2.5 py-1 text-xs font-medium text-civic-ink transition hover:border-civic-leaf hover:text-civic-leaf disabled:opacity-50"
                    aria-label={`Run ${a.name} (coming soon)`}
                  >
                    {isRunning ? (
                      <Loader2 className="h-3 w-3 animate-spin" aria-hidden="true" />
                    ) : (
                      <Sparkles className="h-3 w-3" aria-hidden="true" />
                    )}
                    Run this agent
                  </button>
                </div>
              </li>
            );
          })}
        </ul>
      </section>
    </div>
  );
}
