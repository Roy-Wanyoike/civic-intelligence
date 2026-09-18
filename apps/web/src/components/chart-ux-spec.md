# Chart UX Specification (Spec §63 & §64)

> Reference document for **every** chart on the Civic Intelligence Platform.
> Charts that violate this spec MUST NOT ship to production.

## 1. Source

* **Spec §63 — Chart UX**: every graph must work with mouse, keyboard, touch,
  and screen-reader alternatives. Charts must never hide important
  information exclusively behind hover.
* **Spec §64 — Data Visualization Integrity**: never allow misleading axes,
  missing units, missing time period, truncated axes that distort
  interpretation, unexplained transformations, unlabeled estimates, fake
  precision, unsupported trend claims. Every visualization must disclose
  its source and definition.

## 2. Mandatory chart elements

Every chart MUST render all of the following:

| # | Element | Requirement | Enforced by `BaseChart` |
|---|---------|-------------|--------------------------|
| 1 | **Title** | Human-readable, plain-language title that says *what* the chart shows. | ✅ `title` prop (required) |
| 2 | **Axis labels** | Both X and Y axes labeled with the metric they represent. Units included in the label (e.g. `Debt stock (KES, trillions)`). | ✅ `xLabel` + derived `yLabel` |
| 3 | **Units** | The unit of measure (`KES`, `%`, `count`, etc.) shown in the axis label AND in the title or subtitle. | ✅ `unit` prop (required) |
| 4 | **Source citation** | A clickable link to the authoritative source (CBK, Treasury, Kenya Law, etc.) rendered *below* the chart. | ✅ `sourceURL` prop (required) + "Source:" footer |
| 5 | **Definition tooltip** | A short, plain-language definition of what the chart's primary metric means (e.g. "Total public debt stock = domestic + external debt owed by the central government"). Renders as a `ⓘ` icon with hover/focus tooltip. | ✅ `definition` prop (required) |
| 6 | **Accessible tabular equivalent** | A `<details>` block containing the raw data as an HTML `<table>` — always present, collapsed by default. Screen readers can expand it. | ✅ Auto-rendered `<details>` with full data table |
| 7 | **Hover tooltips** | Each data point has a `<title>` SVG element so hover/focus reveals the exact value. | ✅ Auto-rendered per data point |
| 8 | **Color legend** | A legend mapping each series color to its name, rendered below the chart. Always visible (not behind hover). | ✅ Auto-rendered from `yKeys` |

## 3. Forbidden patterns

A chart MUST NOT:

1. **Be 3D** — no perspective, no extruded bars, no fake depth. 3D charts
   distort visual comparison of values.
2. **Use a dual y-axis** that misleads — dual axes invite spurious
   correlation. If two metrics must be shown together, use a small-multiples
   layout (two side-by-side charts with shared X axis) instead.
3. **Truncate the y-axis** without clear marking. The y-axis SHOULD start at
   zero for bar charts. For line charts where a non-zero baseline is
   defensible (e.g. zooming into a narrow band), the chart MUST:
   * Show a clear break mark (zig-zag) on the axis.
   * Include a subtitle saying "y-axis does not start at zero".
4. **Hide data behind hover** — hover tooltips ENHANCE the chart, they do
   not REPLACE the tabular equivalent. The data table is always rendered.
5. **Omit units** — a chart that shows "1.2T" with no unit is misleading.
   Always include units in axis labels and titles.
6. **Mix scales** — if multiple series share an axis, they must share a
   unit. Never plot e.g. "KES" and "%" on the same axis.
7. **Imply trends without data** — never draw a trend line unless you have
   enough points to support it (≥3) and label the regression method.
8. **Fake precision** — a number like `KES 1,234,567,890.12345` implies
   precision that doesn't exist. Use compact notation (`KES 1.2T`) unless
   the underlying data truly has that precision.

## 4. Interaction requirements (Spec §63)

| Input | Behaviour |
|-------|-----------|
| Mouse | Hover over a data point reveals exact value + date. |
| Keyboard | The chart container is `tabIndex={0}`. Arrow-left/right moves focus between data points; Enter/Space activates the tooltip. |
| Touch | Tap a data point to reveal its tooltip (persistent until tapped again). |
| Screen reader | The chart's `role="img"` has a descriptive `aria-label`. The `<details>` table provides the raw data in a linear, screen-reader-friendly form. |

## 5. `BaseChart` reference component

The reusable wrapper lives at
`apps/web/src/components/base-chart.tsx`.

### Props

```ts
interface BaseChartProps {
  /** Plain-language title — what does this chart show? */
  title: string;
  /** Data points. Each row is one observation (e.g. one month). */
  data: Record<string, string | number | null>[];
  /** Key in each row that holds the X-axis value (e.g. "date"). */
  xKey: string;
  /** Series to plot. Each entry = one line. */
  yKeys: {
    key: string;        // data row key
    name: string;        // legend + tooltip name
    color: string;       // CSS color
  }[];
  /** Unit of measure — e.g. "KES", "%", "count". REQUIRED. */
  unit: string;
  /** Authoritative source URL. Renders below the chart. REQUIRED. */
  sourceURL: string;
  /** Human-readable label for the source (e.g. "Central Bank of Kenya"). */
  sourceLabel?: string;
  /** Plain-language definition of the primary metric. REQUIRED. */
  definition: string;
  /** Optional x-axis label (defaults to the xKey humanized). */
  xLabel?: string;
  /** Optional y-axis label (defaults to unit). */
  yLabel?: string;
}
```

### Enforcement

`BaseChart` is **type-safe required** — the TypeScript compiler rejects
charts that omit `title`, `data`, `xKey`, `yKeys`, `unit`, `sourceURL`,
or `definition`. This makes the spec mechanically enforceable: you
cannot ship a chart that lacks any of the mandatory elements.

### Usage example

```tsx
<BaseChart
  title="Kenya Public Debt Stock (2013–2024)"
  data={timeline}
  xKey="date"
  yKeys={[
    { key: 'total_debt_stock', name: 'Total', color: '#1f5f3f' },
    { key: 'domestic_debt',    name: 'Domestic', color: '#0d6efd' },
    { key: 'external_debt',    name: 'External', color: '#dc3545' },
  ]}
  unit="KES"
  sourceURL="https://www.centralbank.go.ke/public-debt/"
  sourceLabel="Central Bank of Kenya"
  definition="Total public debt stock = domestic + external debt owed by the central government. Reported monthly."
/>
```

## 6. Migration plan

The existing `apps/web/src/app/debt/debt-trend-chart.tsx` already complies
with the spec — it renders a tabular equivalent, axis labels, hover
tooltips, and a legend. It should be refactored to delegate to
`BaseChart` in a follow-up PR so all charts share the same enforcement
point. New charts (impact analyses, scenarios, research visualizations)
MUST use `BaseChart` from day one.
