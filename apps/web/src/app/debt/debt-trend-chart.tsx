'use client';

import { useMemo } from 'react';

interface DebtTrendChartProps {
  points: { date: string; total_debt_stock?: number; domestic_debt?: number; external_debt?: number }[];
}

// Simple SVG-based line chart for the debt trend. No external chart library.
// Each point is rendered as a circle on an SVG polyline.
export function DebtTrendChart({ points }: DebtTrendChartProps) {
  const w = 720;
  const h = 240;
  const padding = { top: 20, right: 20, bottom: 30, left: 60 };

  const { series, xLabels, yMax, yTicks } = useMemo(() => {
    if (points.length === 0) {
      return { series: [], xLabels: [], yMax: 0, yTicks: [] };
    }
    const totals = points.map((p) => p.total_debt_stock ?? 0);
    const max = Math.max(...totals, 1);
    const niceMax = Math.ceil(max / 1e12) * 1e12; // round up to nearest trillion
    const tickCount = 5;
    const ticks = Array.from({ length: tickCount + 1 }, (_, i) => (niceMax * i) / tickCount);

    const xStep = (w - padding.left - padding.right) / Math.max(1, points.length - 1);

    const total = points.map((p, i) => ({
      x: padding.left + i * xStep,
      y: h - padding.bottom - ((p.total_debt_stock ?? 0) / niceMax) * (h - padding.top - padding.bottom),
      value: p.total_debt_stock,
      date: p.date,
    }));
    const domestic = points.map((p, i) => ({
      x: padding.left + i * xStep,
      y: h - padding.bottom - ((p.domestic_debt ?? 0) / niceMax) * (h - padding.top - padding.bottom),
      value: p.domestic_debt,
      date: p.date,
    }));
    const external = points.map((p, i) => ({
      x: padding.left + i * xStep,
      y: h - padding.bottom - ((p.external_debt ?? 0) / niceMax) * (h - padding.top - padding.bottom),
      value: p.external_debt,
      date: p.date,
    }));

    return {
      series: [
        { name: 'Total', color: '#1f5f3f', points: total },
        { name: 'Domestic', color: '#0d6efd', points: domestic },
        { name: 'External', color: '#dc3545', points: external },
      ],
      xLabels: points.map((p, i) => ({
        x: padding.left + i * xStep,
        label: new Date(p.date).getFullYear().toString(),
      })),
      yMax: niceMax,
      yTicks: ticks,
    };
  }, [points]);

  if (points.length === 0) {
    return (
      <div className="mt-4 rounded-lg border border-stone-200 bg-stone-50 p-8 text-center text-sm text-stone-600">
        No debt observations available.
      </div>
    );
  }

  return (
    <div className="mt-4">
      {/* Accessible alternative: tabular equivalent of the chart. */}
      <details className="mb-3">
        <summary className="cursor-pointer text-xs text-civic-stone">
          View tabular equivalent (screen-reader friendly)
        </summary>
        <table className="mt-2 w-full text-xs">
          <thead className="bg-stone-50">
            <tr>
              <th className="px-2 py-1 text-left">Date</th>
              <th className="px-2 py-1 text-right">Total</th>
              <th className="px-2 py-1 text-right">Domestic</th>
              <th className="px-2 py-1 text-right">External</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-stone-100">
            {points.map((p, i) => (
              <tr key={i}>
                <td className="px-2 py-1">{new Date(p.date).toLocaleDateString('en-KE', { year: 'numeric', month: 'short' })}</td>
                <td className="px-2 py-1 text-right">{p.total_debt_stock != null ? formatKES(p.total_debt_stock) : '—'}</td>
                <td className="px-2 py-1 text-right">{p.domestic_debt != null ? formatKES(p.domestic_debt) : '—'}</td>
                <td className="px-2 py-1 text-right">{p.external_debt != null ? formatKES(p.external_debt) : '—'}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </details>
      <svg
        role="img"
        aria-label="Line chart showing Kenya's public debt stock from 2013 to 2024, broken down by total, domestic, and external."
        viewBox={`0 0 ${w} ${h}`}
        className="w-full"
      >
        {/* Y axis */}
        <line x1={padding.left} y1={padding.top} x2={padding.left} y2={h - padding.bottom} stroke="#cbd5e1" />
        {/* X axis */}
        <line x1={padding.left} y1={h - padding.bottom} x2={w - padding.right} y2={h - padding.bottom} stroke="#cbd5e1" />
        {/* Y ticks + labels */}
        {yTicks.map((tick, i) => {
          const y = h - padding.bottom - (tick / yMax) * (h - padding.top - padding.bottom);
          return (
            <g key={i}>
              <line x1={padding.left - 4} y1={y} x2={padding.left} y2={y} stroke="#94a3b8" />
              <text x={padding.left - 8} y={y + 4} textAnchor="end" fontSize="10" fill="#475569">
                {formatKES(tick)}
              </text>
            </g>
          );
        })}
        {/* X labels */}
        {xLabels.map((l, i) => (
          <text key={i} x={l.x} y={h - padding.bottom + 18} textAnchor="middle" fontSize="10" fill="#475569">
            {l.label}
          </text>
        ))}
        {/* Series */}
        {series.map((s) => (
          <g key={s.name}>
            <polyline
              fill="none"
              stroke={s.color}
              strokeWidth="2"
              points={s.points.map((p) => `${p.x},${p.y}`).join(' ')}
            />
            {s.points.map((p, i) => (
              <circle key={i} cx={p.x} cy={p.y} r="3" fill={s.color}>
                <title>{`${s.name}: ${formatKES(p.value ?? 0)} (${new Date(p.date).toLocaleDateString('en-KE', { year: 'numeric', month: 'short' })})`}</title>
              </circle>
            ))}
          </g>
        ))}
      </svg>
      {/* Legend */}
      <div className="mt-3 flex flex-wrap gap-4">
        {series.map((s) => (
          <div key={s.name} className="flex items-center gap-2 text-xs">
            <span className="inline-block h-2 w-4" style={{ backgroundColor: s.color }} aria-hidden="true" />
            <span className="text-stone-700">{s.name}</span>
          </div>
        ))}
      </div>
    </div>
  );
}

function formatKES(amount: number): string {
  return new Intl.NumberFormat('en-KE', {
    style: 'currency',
    currency: 'KES',
    notation: 'compact',
    maximumFractionDigits: 1,
  }).format(amount);
}
