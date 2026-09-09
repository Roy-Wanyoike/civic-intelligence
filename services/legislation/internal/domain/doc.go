// Package legislation is the heart of the Civic Intelligence Platform. It
// owns the canonical civic domain: countries, institutions, legislatures,
// houses, committees, people, bills, bill versions, bill stages, amendments
// and acts.
//
// # Bounded context
//
// The legislation bounded context owns the canonical truth about bills and
// their lifecycle. Other services may PROPOSE changes (via events) but only
// this service may WRITE to the canonical tables.
//
// # What this service does NOT own
//
//   - Raw document bytes (ingestion owns them; we only reference by ID).
//   - Extracted document text (documents service owns it; we cite by ID).
//   - AI explanations (intelligence service owns them).
//   - Search indexes (search service owns them; we publish events it consumes).
//   - Notifications (notifications service owns them).
//   - Users (identity service owns them).
//
// # Country-agnosticism
//
// This service NEVER hard-codes Kenya-specific strings ("National Assembly",
// "Senate", "Second Reading"). All country-specific terminology comes from
// the LegislativeSourceAdapter injected at startup. The BillStateMachine
// validates transitions purely against the StageDefinitions the adapter
// provides.
package legislation
