// Package legislation is the canonical civic-domain service. It owns the
// single source of civic truth: Bills, Acts, stages, events, committees,
// people, and the rules that govern their lifecycle.
//
// Architectural rules (NON-NEGOTIABLE — see ARCHITECTURE.md):
//   1. This package never imports infrastructure (database/sql, net/http,
//      NATS, OpenAI). Domain types are pure.
//   2. Country-specific knowledge (Kenya's stage names, two-house structure,
//      terminology) is ADAPTER DATA, never code in this package. The
//      BillStateMachine takes a []StageDefinition (country-supplied) as input.
//   3. AI cannot directly write to this service's tables. AI proposes
//      candidate facts; a separate validation step (in services/intelligence)
//      accepts/rejects them before they reach here.
package legislation

// Package documentation only. The domain types live in domain.go.
