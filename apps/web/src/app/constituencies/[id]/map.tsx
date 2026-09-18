interface Props {
  constituencyName: string;
}

/**
 * KenyaMapPlaceholder renders a simple SVG outline of Kenya with a label
 * marking the highlighted constituency. It is intentionally NOT a real
 * geographic map — the platform does not fabricate precise district
 * boundaries until the official IEBC boundary shapefile is wired in.
 *
 * The SVG path below is a stylised silhouette of Kenya, suitable for a
 * "you are here" affordance. It carries a `<title>` element so screen
 * readers announce the highlighted area, and a `<desc>` summarising what
 * the graphic shows.
 */
export function KenyaMapPlaceholder({ constituencyName }: Props) {
  return (
    <div
      aria-label={`Map placeholder highlighting ${constituencyName}`}
      className="flex flex-shrink-0 flex-col items-center justify-center rounded-lg border border-civic-border bg-civic-mist p-4 print:bg-civic-paper"
    >
      <svg
        viewBox="0 0 200 240"
        role="img"
        aria-labelledby="kenya-map-title kenya-map-desc"
        className="h-40 w-32 text-civic-forest"
        fill="currentColor"
        stroke="currentColor"
        strokeWidth={1}
      >
        <title id="kenya-map-title">Kenya outline with {constituencyName} highlighted</title>
        <desc id="kenya-map-desc">
          A stylised outline of Kenya. The highlighted pin marks the
          approximate location of {constituencyName}. This is a placeholder —
          the platform does not fabricate precise district boundaries.
        </desc>
        {/* Stylised Kenya silhouette — north at top, coast at right */}
        <path
          d="M 70 20
             L 130 18
             L 165 35
             L 180 70
             L 175 105
             L 185 130
             L 180 165
             L 165 195
             L 145 215
             L 120 222
             L 95 215
             L 75 195
             L 60 170
             L 50 140
             L 45 110
             L 50 80
             L 60 50
             Z"
          fill="none"
          stroke="currentColor"
          strokeWidth={2}
          className="text-civic-forest"
        />
        {/* Highlighted constituency marker — a filled pin */}
        <circle
          cx="115"
          cy="120"
          r="6"
          className="fill-civic-leaf"
          aria-hidden="true"
        />
        <circle
          cx="115"
          cy="120"
          r="12"
          fill="none"
          stroke="currentColor"
          strokeWidth={1}
          className="text-civic-leaf"
          opacity={0.4}
          aria-hidden="true"
        />
      </svg>
      <p className="mt-2 text-center text-[10px] text-civic-stone">
        <span className="font-semibold text-civic-ink">{constituencyName}</span>
        <br />
        Placeholder · not a precise district map
      </p>
    </div>
  );
}
