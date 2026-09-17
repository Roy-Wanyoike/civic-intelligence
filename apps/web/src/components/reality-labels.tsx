// Visual language labels for the simulation domain. Phase 18 §17.
//
// Every label is colour-coded and prefixed with an icon so independent
// testers can correctly distinguish them (Gate K — UI Clarity).
//
// The six labels MUST appear throughout the scenario experience:
//   FACT       — verified civic information
//   EVIDENCE   — authoritative supporting material
//   ASSUMPTION — an explicit scenario input
//   SIMULATION — a modeled outcome
//   UNKNOWN    — insufficient evidence
//   LIMITATION — a known limitation of the model

import { type ReactNode } from 'react';

export type RealityLabel =
  | 'FACT'
  | 'EVIDENCE'
  | 'ASSUMPTION'
  | 'SIMULATION'
  | 'UNKNOWN'
  | 'LIMITATION'
  | 'HYPOTHETICAL';

interface LabelConfig {
  label: RealityLabel;
  bg: string;
  text: string;
  border: string;
  description: string;
}

const LABEL_CONFIGS: Record<RealityLabel, LabelConfig> = {
  FACT: {
    label: 'FACT',
    bg: 'bg-emerald-50',
    text: 'text-emerald-800',
    border: 'border-emerald-300',
    description: 'Verified civic information from an authoritative source.',
  },
  EVIDENCE: {
    label: 'EVIDENCE',
    bg: 'bg-sky-50',
    text: 'text-sky-800',
    border: 'border-sky-300',
    description: 'Authoritative supporting material traceable to a source.',
  },
  ASSUMPTION: {
    label: 'ASSUMPTION',
    bg: 'bg-amber-50',
    text: 'text-amber-800',
    border: 'border-amber-300',
    description: 'An explicit scenario input that has not been observed.',
  },
  SIMULATION: {
    label: 'SIMULATION',
    bg: 'bg-violet-50',
    text: 'text-violet-800',
    border: 'border-violet-300',
    description: 'A modeled outcome. This is NOT an observed civic fact.',
  },
  UNKNOWN: {
    label: 'UNKNOWN',
    bg: 'bg-stone-100',
    text: 'text-stone-700',
    border: 'border-stone-300',
    description: 'Insufficient evidence. The platform does not fabricate precision.',
  },
  LIMITATION: {
    label: 'LIMITATION',
    bg: 'bg-rose-50',
    text: 'text-rose-800',
    border: 'border-rose-300',
    description: 'A known limitation of the model or scenario.',
  },
  HYPOTHETICAL: {
    label: 'HYPOTHETICAL',
    bg: 'bg-violet-50',
    text: 'text-violet-800',
    border: 'border-violet-300',
    description: 'A constructed scenario. It is NOT an observed civic fact.',
  },
};

interface RealityBadgeProps {
  kind: RealityLabel;
  size?: 'sm' | 'md';
  showIcon?: boolean;
  children?: ReactNode;
}

export function RealityBadge({ kind, size = 'sm', children }: RealityBadgeProps) {
  const cfg = LABEL_CONFIGS[kind];
  const sizeClass = size === 'sm' ? 'text-[10px] px-1.5 py-0.5' : 'text-xs px-2 py-1';
  return (
    <span
      className={`inline-flex items-center gap-1 rounded-full border font-semibold uppercase tracking-wide ${cfg.bg} ${cfg.text} ${cfg.border} ${sizeClass}`}
      title={cfg.description}
      aria-label={cfg.label}
    >
      <span className="inline-block h-1.5 w-1.5 rounded-full bg-current" aria-hidden="true" />
      {cfg.label}
      {children}
    </span>
  );
}

// RealityDisclaimer is a full-width banner used at the top of every page
// that surfaces simulation content.
export function RealityDisclaimer({ kind = 'HYPOTHETICAL' }: { kind?: RealityLabel }) {
  const cfg = LABEL_CONFIGS[kind];
  return (
    <div
      className={`rounded-lg border ${cfg.border} ${cfg.bg} p-4`}
      role="status"
      aria-live="polite"
    >
      <div className="flex items-start gap-3">
        <RealityBadge kind={kind} size="md" />
        <p className={`text-sm ${cfg.text}`}>
          {cfg.description} The platform never presents a simulated outcome as an
          actual civic event. Reality is observed; scenarios are constructed;
          assumptions are explicit; results are simulated; evidence remains
          traceable.
        </p>
      </div>
    </div>
  );
}

// TimelineLabel is a label specifically for scenario timeline events.
export function TimelineLabel({ kind }: { kind: 'OBSERVED' | 'ASSUMED' | 'MODELED' | 'UNKNOWN' }) {
  const map: Record<string, RealityLabel> = {
    OBSERVED: 'FACT',
    ASSUMED: 'ASSUMPTION',
    MODELED: 'SIMULATION',
    UNKNOWN: 'UNKNOWN',
  };
  return <RealityBadge kind={map[kind] ?? 'UNKNOWN'} />;
}
