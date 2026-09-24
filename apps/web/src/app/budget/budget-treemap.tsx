'use client';

import { useEffect, useMemo, useRef, useState, type ReactNode } from 'react';
import { hierarchy, treemap, treemapSquarify } from 'd3-hierarchy';
import type { BudgetAllocation } from '@/lib/budget-api';

/**
 * BudgetTreemap — interactive D3 treemap of national budget allocations
 * (issue #285 / FEAT-7).
 *
 * Cells are sized by allocation amount (larger allocation = larger cell)
 * and coloured by category:
 *   - recurrent     → forest green  (`#0c3b2e`, colors.forest)
 *   - development   → acacia gold   (`#d4a017`, colors.acacia)
 *
 * Hover any cell to reveal a tooltip with the ministry name, allocation
 * in KES billions, and the percentage share of the total budget.
 *
 * Responsive behaviour (spec §41): below 768px the treemap is replaced
 * by an accessible HTML <table> — a treemap's labels become unreadable
 * on small viewports, so the table is the better mobile UX. The table
 * is also the screen-reader-friendly alternative (the SVG treemap
 * carries an `aria-label` summarising the chart, but the table is the
 * authoritative machine-readable form).
 *
 * The treemap layout uses D3's `treemapSquarify` ratio algorithm
 * (D3's default + the published "best" treemap algorithm) so cells
 * stay close to square — this keeps ministry names readable.
 *
 * No external chart library beyond D3 is used (issue #285 acceptance:
 * "Do NOT use external chart libs beyond D3"). The SVG cells are
 * rendered manually from the layout positions D3 computes.
 */
interface BudgetTreemapProps {
  items: BudgetAllocation[];
}

// Canonical Civic Intelligence palette — see apps/web/src/lib/design-tokens.ts.
// Hard-coded here (rather than imported) so the component stays
// self-contained + the design tokens module isn't pulled into the
// budget page bundle for one-off reads.
const CATEGORY_COLORS: Record<BudgetAllocation['category'], string> = {
  recurrent: '#0c3b2e', // forest green — services, salaries, debt servicing
  development: '#d4a017', // acacia gold — capital projects, infrastructure
};

const CATEGORY_LABELS: Record<BudgetAllocation['category'], string> = {
  recurrent: 'Recurrent',
  development: 'Development',
};

// Treemap viewport. The SVG uses viewBox so it scales responsively with
// its container; the aspect ratio is locked at 16:9 (matches the debt
// trend chart's aspect ratio for visual consistency across Finance).
const TREEMAP_WIDTH = 960;
const TREEMAP_HEIGHT = 540;

// Minimum cell dimension (in px) below which the ministry name is
// suppressed. Below this, the label would visually collide with the
// cell boundary — the tooltip still surfaces the ministry name on
// hover so no information is lost.
const MIN_CELL_FOR_LABEL = 90;

// Hook: tracks whether the viewport matches the supplied media query.
// Re-evaluates on resize so the treemap / table swap is live.
function useMediaQuery(query: string): boolean {
  const [matches, setMatches] = useState(false);
  useEffect(() => {
    if (typeof window === 'undefined' || !window.matchMedia) return;
    const mql = window.matchMedia(query);
    const update = () => setMatches(mql.matches);
    update();
    mql.addEventListener('change', update);
    return () => mql.removeEventListener('change', update);
  }, [query]);
  return matches;
}

interface TreemapNode {
  x: number;
  y: number;
  width: number;
  height: number;
  data: BudgetAllocation;
}

export function BudgetTreemap({ items }: BudgetTreemapProps) {
  const [tooltip, setTooltip] = useState<{ x: number; y: number; content: ReactNode } | null>(null);
  const containerRef = useRef<HTMLDivElement>(null);
  const isMobile = useMediaQuery('(max-width: 767px)');

  // Compute the treemap layout once per items change. The layout is
  // deterministic given the input, so memoising on `items` keeps the
  // SVG stable across re-renders that don't change the data (e.g. a
  // tooltip position update).
  const cells = useMemo<TreemapNode[]>(() => {
    if (!items || items.length === 0) return [];
    // d3-hierarchy requires a root node. We use a flat root with the
    // items as children — every child's `value` is its allocation so
    // the treemap algorithm partitions the canvas by allocation size.
    //
    // The datum type is a union: the root carries `children`, leaves
    // carry the ministry allocation fields. TypeScript can't narrow
    // the union through `hierarchy().sum()`, so we cast to a single
    // permissive datum type at the boundary — the runtime contract is
    // `sum()` returns a positive number for leaves + 0 for the root.
    type BudgetDatum = {
      name?: string;
      children?: BudgetAllocation[];
    } & Partial<BudgetAllocation>;
    const root = hierarchy<BudgetDatum>({ name: 'budget', children: items })
      .sum((d) => (d.amount_kes_billions ?? 0))
      .sort((a, b) => (b.value ?? 0) - (a.value ?? 0));
    const layout = treemap<BudgetDatum>()
      .tile(treemapSquarify.ratio(1.6))
      .size([TREEMAP_WIDTH, TREEMAP_HEIGHT])
      .paddingInner(2)
      .paddingOuter(2)
      .round(true);
    layout(root);
    // Flatten the layout into a render-friendly list of cells. The
    // root + intermediate nodes are filtered out (we only want leaves
    // — i.e. actual ministry allocations). Leaves carry a `data`
    // field that is a BudgetAllocation (the union narrowed at runtime
    // because leaves never have `children`).
    return (root.leaves() as unknown as { x0: number; y0: number; x1: number; y1: number; data: BudgetAllocation }[])
      .map((leaf) => ({
        x: leaf.x0,
        y: leaf.y0,
        width: leaf.x1 - leaf.x0,
        height: leaf.y1 - leaf.y0,
        data: leaf.data,
      }));
  }, [items]);

  // Tooltip positioning: clamped to the container so the tooltip never
  // overflows the viewport. Recomputed on every hover (the tooltip
  // position is relative to the container, not the cell, so a hover
  // near the right edge flips to the left of the cursor).
  const handleCellEnter = (cell: TreemapNode, event: React.MouseEvent) => {
    if (!containerRef.current) return;
    const rect = containerRef.current.getBoundingClientRect();
    const x = event.clientX - rect.left;
    const y = event.clientY - rect.top;
    // Flip the tooltip to the left of the cursor when within 220px of
    // the right edge — keeps the tooltip on-screen without a layout
    // pass.
    const flipX = x > rect.width - 220;
    const flipY = y > rect.height - 100;
    setTooltip({
      x: flipX ? x - 12 : x + 12,
      y: flipY ? y - 12 : y + 12,
      content: (
        <div className="space-y-1">
          <div className="font-semibold text-civic-ink">{cell.data.ministry}</div>
          <div className="text-civic-ink">
            KES {cell.data.amount_kes_billions.toLocaleString('en-KE', { maximumFractionDigits: 1 })}B
            <span className="ml-2 text-civic-stone">
              ({cell.data.percentage.toFixed(2)}% of budget)
            </span>
          </div>
          <div className="text-xs text-civic-stone">
            {CATEGORY_LABELS[cell.data.category]} allocation
          </div>
        </div>
      ),
    });
  };

  const handleCellLeave = () => setTooltip(null);

  // Mobile fallback: an accessible HTML table. Each row carries the
  // ministry name, category badge, allocation, and percentage. The
  // table is sortable by allocation (largest-first, matching the seed
  // slice's order) so the citizen sees the budget's biggest items at
  // a glance.
  if (isMobile || cells.length === 0) {
    return <BudgetTable items={items} />;
  }

  return (
    <div ref={containerRef} className="relative">
      {/* Treemap legend */}
      <div className="mb-3 flex flex-wrap gap-4 text-xs">
        {(Object.keys(CATEGORY_LABELS) as BudgetAllocation['category'][]).map((cat) => (
          <div key={cat} className="flex items-center gap-2">
            <span
              className="inline-block h-3 w-4 rounded-sm"
              style={{ backgroundColor: CATEGORY_COLORS[cat] }}
              aria-hidden="true"
            />
            <span className="text-civic-stone">{CATEGORY_LABELS[cat]}</span>
          </div>
        ))}
      </div>

      <svg
        role="img"
        aria-label={`Treemap of ${items.length} ministry budget allocations. Cells sized by allocation amount; coloured forest green for recurrent spending and acacia gold for development spending.`}
        viewBox={`0 0 ${TREEMAP_WIDTH} ${TREEMAP_HEIGHT}`}
        className="w-full"
      >
        {cells.map((cell, i) => {
          const color = CATEGORY_COLORS[cell.data.category];
          const showLabel = cell.width >= MIN_CELL_FOR_LABEL && cell.height >= 36;
          const showAmount = cell.width >= MIN_CELL_FOR_LABEL && cell.height >= 60;
          // Truncate the ministry name to fit the cell width. Each
          // character is ~6px at the label font size, so cap at
          // width/8 - 2 chars.
          const maxChars = Math.max(0, Math.floor(cell.width / 8) - 2);
          const ministry =
            cell.data.ministry.length > maxChars
              ? cell.data.ministry.slice(0, Math.max(0, maxChars - 1)) + '…'
              : cell.data.ministry;
          return (
            <g
              key={`${cell.data.ministry}-${i}`}
              onMouseEnter={(e) => handleCellEnter(cell, e)}
              onMouseMove={(e) => handleCellEnter(cell, e)}
              onMouseLeave={handleCellLeave}
              className="cursor-pointer"
            >
              <rect
                x={cell.x}
                y={cell.y}
                width={cell.width}
                height={cell.height}
                fill={color}
                stroke="#ffffff"
                strokeWidth={1}
                rx={2}
              />
              {showLabel && (
                <text
                  x={cell.x + 6}
                  y={cell.y + 16}
                  fontSize="11"
                  fontWeight="600"
                  fill="#ffffff"
                  className="pointer-events-none select-none"
                >
                  {ministry}
                </text>
              )}
              {showAmount && (
                <text
                  x={cell.x + 6}
                  y={cell.y + 32}
                  fontSize="10"
                  fill="#ffffff"
                  fillOpacity={0.85}
                  className="pointer-events-none select-none"
                >
                  KES {cell.data.amount_kes_billions.toLocaleString('en-KE', { maximumFractionDigits: 1 })}B
                </text>
              )}
              {showAmount && cell.height >= 80 && (
                <text
                  x={cell.x + 6}
                  y={cell.y + 46}
                  fontSize="9"
                  fill="#ffffff"
                  fillOpacity={0.7}
                  className="pointer-events-none select-none"
                >
                  {cell.data.percentage.toFixed(1)}%
                </text>
              )}
            </g>
          );
        })}
      </svg>

      {tooltip && (
        <div
          className="pointer-events-none absolute z-20 max-w-xs rounded-md border border-civic-border bg-civic-paper px-3 py-2 text-xs shadow-xl"
          style={{ left: tooltip.x, top: tooltip.y }}
          role="tooltip"
        >
          {tooltip.content}
        </div>
      )}
    </div>
  );
}

/**
 * BudgetTable — the mobile + screen-reader-friendly fallback for the
 * treemap. Renders an accessible HTML table with one row per ministry
 * allocation, sorted largest-first (matching the seed slice's order).
 *
 * The category is rendered as a colour-coded badge so the citizen can
 * scan the table for the recurrent / development balance at a glance,
 * mirroring the treemap's colour encoding.
 */
function BudgetTable({ items }: BudgetTreemapProps) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <caption className="sr-only">
          Ministry budget allocations, sorted largest-first. Each row
          carries the ministry name, category, allocation in KES billions,
          and percentage share of the total budget.
        </caption>
        <thead className="bg-civic-mist text-civic-stone">
          <tr>
            <th scope="col" className="px-3 py-2 text-left font-semibold">Ministry</th>
            <th scope="col" className="px-3 py-2 text-left font-semibold">Category</th>
            <th scope="col" className="px-3 py-2 text-right font-semibold">KES (B)</th>
            <th scope="col" className="px-3 py-2 text-right font-semibold">Share</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-civic-border">
          {items.map((item) => (
            <tr key={item.ministry} className="hover:bg-civic-mist">
              <td className="px-3 py-2 text-civic-ink">{item.ministry}</td>
              <td className="px-3 py-2">
                <span
                  className="inline-flex items-center gap-1.5 rounded-full px-2 py-0.5 text-xs font-medium"
                  style={{
                    backgroundColor: `${CATEGORY_COLORS[item.category]}1a`, // 10% opacity
                    color: CATEGORY_COLORS[item.category],
                  }}
                >
                  <span
                    className="inline-block h-2 w-2 rounded-full"
                    style={{ backgroundColor: CATEGORY_COLORS[item.category] }}
                    aria-hidden="true"
                  />
                  {CATEGORY_LABELS[item.category]}
                </span>
              </td>
              <td className="px-3 py-2 text-right tabular-nums text-civic-ink">
                {item.amount_kes_billions.toLocaleString('en-KE', { maximumFractionDigits: 1 })}
              </td>
              <td className="px-3 py-2 text-right tabular-nums text-civic-stone">
                {item.percentage.toFixed(2)}%
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
