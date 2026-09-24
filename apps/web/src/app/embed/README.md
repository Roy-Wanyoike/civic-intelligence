// Embeddable widget pages — issue #291.
//
// Routes under /embed/* are designed to be loaded in third-party sites
// via <iframe>. They render chrome-less (no navbar/footer/providers —
// see app/layout.tsx's `isEmbedRequest()` branch) and use minimal
// inline CSS so the host page's stylesheet can't leak in or out.
//
// The widget pages are server components — they fetch their data from
// the same Go BFF (`API_BASE_URL`) the main app uses, fall back to
// mock data when the BFF is unreachable (so the widget never renders
// an error stack trace inside someone else's page), and emit a
// "Powered by Civic Intelligence" link at the bottom of every render.
//
// Available widgets:
//   /embed/mp-finder     — constituency lookup → MP scorecard link
//   /embed/bill-tracker  — Bill current-stage timeline (?bill_id=…)
//   /embed/today         — today's parliament calendar events
//
// Documentation + copy-pasteable iframe snippets: /developers.
