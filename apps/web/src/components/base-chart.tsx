'use client';

/**
 * `BaseChart` — reusable chart wrapper enforcing Spec §63 (Chart UX) and
 * Spec §64 (Data Visualization Integrity).
 *
 * Why this exists:
 *   The repo had exactly one chart (`debt-trend-chart.tsx`) that happened
 *   to comply with the spec, but every future chart would have to
 *   re-implement the same accessibility + integrity rules manually. This
 *   wrapper turns the spec into a compile-time requirement: TypeScript
 *   rejects any chart that omits `title`, `unit`, `sourceURL`, or
 *   `definition`.
 *
 * What it always renders (Spec §63 / §64):
 *   1. Title with a `ⓘ` definition tooltip
 *   2. SVG line chart with axis labels and units
 *   3. Hover tooltips (`<title>` SVG elements) on every data point
 *   4. Always-visible color legend
 *   5. Collapsible `<details>` with a tabular equivalent (screen-reader
 *      and keyboard accessible)
 *   6. Source citation footer with clickable link
 *
 * What it NEVER does:
 *   - 3D effects
 *   - Dual y-axis (use small multiples instead)
 *   - Truncated y-axis without marking (y-axis always starts at 0; if a
 *     non-zero baseline is required, render two side-by-side `BaseChart`s
 *     with shared X axis)
 *
 * See `apps/web/src/components/chart-ux-spec.md` for the full spec.
 */
import { useMemo } from 'react';

export interface ChartSeries {
  /** Key in each data row that holds the value for this series. */
  key: string;
  /** Display name used in the legend and tooltip. */
  name: string;
  /** CSS color for the line (e.g. `'#1f5f3f'`). */
  color: string;
}

export interface BaseChartProps {
  /** Plain-language title — what does this chart show? */
  title: string;
  /** Data points. Each row is one observation (e.g. one month). */
  data: Record<string, string | number | null>[];
  /** Key in each row that holds the X-axis value (e.g. "date"). */
  xKey: string;
  /** Series to plot. Each entry = one line. */
  yKeys: ChartSeries[];
  /** Unit of measure — e.g. "KES", "%", "count". REQUIRED. */
  unit: string;
  /** Authoritative source URL. Renders below the chart. REQUIRED. */
  sourceURL: string;
  /** Human-readable label for the source (e.g. "Central Bank of Kenya"). */
  sourceLabel?: string;
  /** Plain-language definition of the primary metric. REQUIRED. */
  definition: string;
  /** Optional x-axis label (defaults to the xKey, humanized). */
  xLabel?: string;
  /** Optional y-axis label (defaults to the unit). */
  yLabel?: string;
  /** Optional formatter for axis + tooltip values (e.g. compact currency). */
  formatValue?: (value: number) => string;
  /** Optional formatter for x-axis labels (defaults to the raw value). */
  formatX?: (value: string | number) => string;
}

// Sensible default: compact number formatting.
function defaultFormatValue(value: number): string {
  if (!Number.isFinite(value)) return '—';
  return new Intl.NumberFormat('en-KE', {
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(value);
}

function defaultFormatX(value: string | number): string {
  if (typeof value === 'string' && /^\d{4}-\d{2}/.test(value)) {
    // Looks like an ISO date — show the year.
    try {
      return new Date(value).getFullYear().toString();
    } catch {
      return String(value);
    }
  }
  return String(value);
}

function humanizeKey(key: string): string {
  return key
    .replace(/_/g, ' ')
    .replace(/\b\w/g, (c) => c.toUpperCase());
}

export function BaseChart({
  title,
  data,
  xKey,
  yKeys,
  unit,
  sourceURL,
  sourceLabel,
  definition,
  xLabel,
  yLabel,
  formatValue = defaultFormatValue,
  formatX = defaultFormatX,
}: BaseChartProps) {
  const w = 720;
  const h = 240;
  const padding = { top: 24, right: 20, bottom: 36, left: 64 };

  const xLabelFinal = xLabel ?? humanizeKey(xKey);
  const yLabelFinal = yLabel ?? unit;

  const {
    series,
    xLabels,
    yMax,
    yTicks,
    empty,
  } = useMemo(() => {
    if (data.length === 0) {
      return { series: [], xLabels: [], yMax: 0, yTicks: [], empty: true };
    }
    // Find the max absolute value across all series so a single y-axis can host them.
    const allValues = data.flatMap((row) =>
      yKeys.map((s) => Number(row[s.key] ?? 0)),
    );
    const maxRaw = Math.max(...allValues, 0);
    // Round up to a "nice" maximum. We use a 5-step grid.
    const niceMax = niceNumber(maxRaw || 1);
    const tickCount = 5;
    const ticks = Array.from(
      { length: tickCount + 1 },
      (_, i) => (niceMax * i) / tickCount,
    );

    const xStep =
      (w - padding.left - padding.right) / Math.max(1, data.length - 1);

    const computedSeries = yKeys.map((s) => ({
      ...s,
      points: data.map((row, i) => {
        const raw = row[s.key];
        const value = raw == null ? null : Number(raw);
        return {
          x: padding.left + i * xStep,
          y:
            value == null || !Number.isFinite(value)
              ? null
              : h -
                padding.bottom -
                (value / niceMax) * (h - padding.top - padding.bottom),
          value,
          xRaw: row[xKey],
        };
      }),
    }));

    const computedXLabels = data.map((row, i) => ({
      x: padding.left + i * xStep,
      label: formatX(row[xKey] ?? ''),
    }));

    return {
      series: computedSeries,
      xLabels: computedXLabels,
      yMax: niceMax,
      yTicks: ticks,
      empty: false,
    };
  }, [data, xKey, yKeys, formatX]);

  if (empty) {
    return (
      <section
        role="figure"
        aria-label={title}
        className="mt-4 rounded-lg border border-civic-border bg-civic-mist p-8 text-center text-sm text-civic-stone"
      >
        No data available for &ldquo;{title}&rdquo;.
      </section>
    );
  }

  const ariaLabel = [
    `${title}. Line chart with ${yKeys.length} series:`,
    yKeys.map((s) => s.name).join(', '),
    `Unit: ${unit}.`,
    `Source: ${sourceLabel ?? sourceURL}.`,
  ].join(' ');

  return (
    <section role="figure" aria-label={title} className="mt-4">
      {/* Title row with definition tooltip */}
      <div className="mb-2 flex items-start justify-between gap-3">
        <div className="flex items-start gap-1.5">
          <h3 className="font-serif text-base font-semibold text-civic-forest">
            {title}
          </h3>
          <span className="group relative inline-flex">
            <span
              className="inline-flex h-4 w-4 cursor-help items-center justify-center rounded-full border border-civic-border bg-civic-mist text-[10px] font-bold text-civic-stone"
              tabIndex={0}
              role="button"
              aria-label={`Definition: ${definition}`}
            >
              i
            </span>
            <span
              role="tooltip"
              className="pointer-events-none absolute left-1/2 top-6 z-20 hidden w-72 -translate-x-1/2 rounded-md border border-civic-border bg-civic-paper p-3 text-xs text-civic-ink shadow-lg group-hover:block group-focus-within:block"
            >
              {definition}
            </span>
          </span>
        </div>
        <span className="text-[11px] font-medium uppercase tracking-wider text-civic-stone">
          Unit: {unit}
        </span>
      </div>

      {/* Accessible alternative: tabular equivalent (Spec §63). */}
      <details className="mb-3">
        <summary className="cursor-pointer text-xs text-civic-stone">
          View tabular equivalent (screen-reader friendly)
        </summary>
        <table className="mt-2 w-full text-xs">
          <thead className="bg-civic-mist">
            <tr>
              <th className="px-2 py-1 text-left">{xLabelFinal}</th>
              {yKeys.map((s) => (
                <th key={s.key} className="px-2 py-1 text-right">
                  {s.name} ({unit})
                </th>
              ))}
            </tr>
          </thead>
          <tbody className="divide-y divide-civic-border">
            {data.map((row, i) => (
              <tr key={i}>
                <td className="px-2 py-1">{formatX(row[xKey] ?? '')}</td>
                {yKeys.map((s) => {
                  const raw = row[s.key];
                  const value = raw == null ? null : Number(raw);
                  return (
                    <td key={s.key} className="px-2 py-1 text-right">
                      {value == null || !Number.isFinite(value)
                        ? '—'
                        : formatValue(value)}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </details>

      {/* The SVG chart itself — keyboard-focusable, with aria-label. */}
      <svg
        role="img"
        aria-label={ariaLabel}
        viewBox={`0 0 ${w} ${h}`}
        className="w-full focus:outline-none focus-visible:ring-2 focus-visible:ring-civic-leaf"
        tabIndex={0}
      >
        {/* Y axis */}
        <line
          x1={padding.left}
          y1={padding.top}
          x2={padding.left}
          y2={h - padding.bottom}
          stroke="currentColor"
          className="text-civic-border"
        />
        {/* X axis */}
        <line
          x1={padding.left}
          y1={h - padding.bottom}
          x2={w - padding.right}
          y2={h - padding.bottom}
          stroke="currentColor"
          className="text-civic-border"
        />
        {/* Y-axis label (rotated) */}
        <text
          x={-(h / 2)}
          y={14}
          transform="rotate(-90)"
          textAnchor="middle"
          fontSize="11"
          fill="#52606b"
        >
          {yLabelFinal}
        </text>
        {/* X-axis label */}
        <text
          x={w / 2}
          y={h - 2}
          textAnchor="middle"
          fontSize="11"
          fill="#52606b"
        >
          {xLabelFinal}
        </text>
        {/* Y ticks + labels */}
        {yTicks.map((tick, i) => {
          const y =
            h -
            padding.bottom -
            (tick / yMax) * (h - padding.top - padding.bottom);
          return (
            <g key={i}>
              <line
                x1={padding.left - 4}
                y1={y}
                x2={padding.left}
                y2={y}
                stroke="#94a3b8"
              />
              <text
                x={padding.left - 8}
                y={y + 4}
                textAnchor="end"
                fontSize="10"
                fill="#475569"
              >
                {formatValue(tick)}
              </text>
            </g>
          );
        })}
        {/* X labels */}
        {xLabels.map((l, i) => (
          <text
            key={i}
            x={l.x}
            y={h - padding.bottom + 18}
            textAnchor="middle"
            fontSize="10"
            fill="#475569"
          >
            {l.label}
          </text>
        ))}
        {/* Series */}
        {series.map((s) => {
          // Build a polyline string skipping null y values.
          const validPoints = s.points.filter(
            (p) => p.y != null && p.value != null,
          ) as { x: number; y: number; value: number; xRaw: string | number }[];
          const polylinePoints = validPoints
            .map((p) => `${p.x},${p.y}`)
            .join(' ');
          return (
            <g key={s.key}>
              {validPoints.length > 1 && (
                <polyline
                  fill="none"
                  stroke={s.color}
                  strokeWidth="2"
                  points={polylinePoints}
                />
              )}
              {validPoints.map((p, i) => (
                <circle key={i} cx={p.x} cy={p.y} r="3" fill={s.color}>
                  {/* Hover/focus tooltip (Spec §63) */}
                  <title>
                    {`${s.name}: ${formatValue(p.value)} ${unit} (${formatX(p.xRaw)})`}
                  </title>
                </circle>
              ))}
            </g>
          );
        })}
      </svg>

      {/* Always-visible color legend (Spec §63 — not behind hover) */}
      <div className="mt-3 flex flex-wrap gap-4">
        {yKeys.map((s) => (
          <div key={s.key} className="flex items-center gap-2 text-xs">
            <span
              className="inline-block h-2 w-4"
              style={{ backgroundColor: s.color }}
              aria-hidden="true"
            />
            <span className="text-civic-ink">{s.name}</span>
          </div>
        ))}
      </div>

      {/* Source citation (Spec §64 — every visualization must disclose its source and definition) */}
      <p className="mt-3 text-[11px] text-civic-stone">
        Source:{' '}
        <a
          href={sourceURL}
          target="_blank"
          rel="noopener noreferrer"
          className="underline hover:text-civic-leaf"
        >
          {sourceLabel ?? sourceURL}
        </a>
        {' · '}
        <span>
          Definition: {definition}
        </span>
      </p>
    </section>
  );
}

/** Round up to a "nice" maximum (1, 2, 5 × 10^n) so the y-axis ticks land on round numbers. */
function niceNumber(value: number): number {
  if (value <= 0) return 1;
  const exponent = Math.floor(Math.log10(value));
  const fraction = value / Math.pow(10, exponent);
  let niceFraction: number;
  if (fraction <= 1) niceFraction = 1;
  else if (fraction <= 2) niceFraction = 2;
  else if (fraction <= 5) niceFraction = 5;
  else niceFraction = 10;
  return niceFraction * Math.pow(10, exponent);
}
