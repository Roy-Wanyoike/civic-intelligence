// Package simulation is the canonical scenario & simulation service. It owns
// the lifecycle of every hypothetical exploration on the platform.
//
// Architectural rules (NON-NEGOTIABLE — see Phase 18 spec §1):
//   1. This package NEVER mutates canonical civic facts. A simulation may
//      inform research; it cannot modify reality.
//   2. Every simulation record is explicitly distinguishable from observed
//      civic facts via the RealityLayer enum.
//   3. Every assumption is explicit (AssumptionType), every model is
//      versioned, every run is reproducible, every result is labelled
//      SIMULATION.
//   4. The package never imports infrastructure (database/sql, net/http,
//      NATS, OpenAI). Domain types are pure Go.
//   5. No political recommendations. The system describes differences and
//      leaves decisions to users.
package simulation

// Package documentation only. Domain types live in internal/domain.
