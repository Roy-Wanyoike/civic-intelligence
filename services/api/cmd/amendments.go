// Package main provides the per-Bill amendment tracker (issue #293 /
// FEAT-15).
//
// An Amendment is a single proposed change to a Bill — raised by an MP
// (or, in committee-of-the-whole form, by a committee) during the
// Bill's committee stage or third reading. Each amendment carries a
// lifecycle (proposed → accepted | rejected) and a one-line summary of
// the substantive change. The full text diff between the parent Bill
// version and the amendment's clause text is surfaced by the documents
// service's `/compare` capability (issue #102) — this endpoint returns
// only the amendment metadata.
//
// Endpoint (dispatched from makeBillDetailHandler in main.go, same as
// /timeline, /summary, /changes):
//
//	GET /api/v1/bills/{id}/amendments   -- amendments proposed against one Bill
//
// The amendments slice is seed-only for FEAT-15. Live amendment
// ingestion (Hansard committee-stage parsing) is pending issue #19 +
// the documents service's immutable Bill version storage (ADR-0011).
// Until that lands, the handler returns seed data with `source: "seed"`
// so callers can distinguish seed amendments from future live ones.
//
// The seed slice covers 3 sample Bills (4+3+3 = 10 amendments) drawn
// from kenya_seed.SampleBills so the amendments tab always renders a
// realistic mix of proposed + accepted + rejected amendments during
// FEAT-15 development.
package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_seed"
)

// AmendmentStatus is the lifecycle state of a proposed amendment. The
// three values match the parliamentary committee-stage vocabulary:
//
//   - proposed:  the amendment has been tabled but not yet voted on
//   - accepted:  the amendment was carried (majority vote) and folded
//     into the Bill's current working text
//   - rejected:  the amendment was defeated; the Bill's text is unchanged
//
// Withdrawn amendments are NOT modelled here — once tabled, an
// amendment stays on the record even if the sponsor later withdraws
// support (parliamentary convention).
type AmendmentStatus string

const (
	AmendmentStatusProposed AmendmentStatus = "proposed"
	AmendmentStatusAccepted AmendmentStatus = "accepted"
	AmendmentStatusRejected AmendmentStatus = "rejected"
)

// Amendment is the JSON shape returned by /api/v1/bills/{id}/amendments.
// Every field is populated from seed data during FEAT-15; the live
// ingestion path (issue #19) will populate the same shape from parsed
// Hansard committee-stage records.
//
// Field semantics:
//
//   - id:           stable platform ID (e.g., "amend-001")
//   - bill_id:      the parent Bill's SourceID (NOT a slug — the full ID
//     returned by /api/v1/bills/{id} so callers can correlate
//     amendments + Bills without an extra lookup)
//   - title:        one-line description of the substantive change
//     (e.g., "Clause 14(2) — increase contribution rate to 8%")
//   - proposed_by:  the platform-internal person ID of the sponsor (e.g.,
//     "person-001"). Cross-references /api/v1/people/{id}/scorecard.
//   - proposed_at:  RFC3339 timestamp of the committee sitting at which the
//     amendment was tabled.
//   - status:       proposed | accepted | rejected
//   - summary:      plain-language description of what the amendment would
//     change + why the sponsor moved it.
//   - source_url:   link to the official record (Hansard, Order Paper, or
//     kenyalaw.org Bill page). Seed amendments point at the
//     parent Bill's kenyalaw.org URL until the Hansard parser
//     can mint per-sitting URLs.
type Amendment struct {
	ID         string          `json:"id"`
	BillID     string          `json:"bill_id"`
	Title      string          `json:"title"`
	ProposedBy string          `json:"proposed_by"`
	ProposedAt string          `json:"proposed_at"`
	Status     AmendmentStatus `json:"status"`
	Summary    string          `json:"summary"`
	SourceURL  string          `json:"source_url"`
}

// sampleAmendmentSpec is the raw input used to build a seed Amendment.
// billIndex is the offset into kenya_seed.SampleBills; the actual
// SourceID is resolved at init time so the seed amendments stay
// attached to the (truncation-prone) ID minted by the seed Bills slice
// — callers always pass the same ID /api/v1/bills/{id} returns.
//
// billURL is the parent Bill's kenyalaw.org URL; copied into the
// amendment's source_url because per-sitting Hansard URLs are not yet
// minted by the ingestion worker (issue #19).
type sampleAmendmentSpec struct {
	id         string
	billIndex  int
	title      string
	proposedBy string
	proposedAt string
	status     AmendmentStatus
	summary    string
}

// sampleAmendmentSpecs is the 10-row seed slice distributed across 3
// sample Bills (issue #293). The three Bills are picked to span the
// finance + health policy areas (the two most active in the seed
// slice) so the amendments tab can render a realistic mix of proposed +
// already-disposed amendments.
//
// Bill 0 — The Local Authorities Provident Fund Amendment Bill 2026 (4 amendments)
// Bill 1 — The National Coroners Service Bill 2026                (3 amendments)
// Bill 3 — The Public Finance Management Amendment Bill 2026     (3 amendments)
//
// Source: derived from typical Committee-of-the-Whole-House amendment
// patterns for the listed Bills. The seed slice is intentionally small
// (10 rows) and powers the amendments tab UI during the FEAT-15
// development cycle. Live amendment ingestion (Hansard parsing) is
// pending issue #19.
var sampleAmendmentSpecs = []sampleAmendmentSpec{
	// --- Bill 0: The Local Authorities Provident Fund Amendment Bill 2026 ---
	{
		id:         "amend-001",
		billIndex:  0,
		title:      "Clause 14(2) — increase member contribution rate to 8%",
		proposedBy: "person-002",
		proposedAt: "2026-09-15T10:30:00Z",
		status:     AmendmentStatusProposed,
		summary:    "Raises the member contribution rate from 6% to 8% over three fiscal years, with the increment staged 6% → 7% → 8%.",
	},
	{
		id:         "amend-002",
		billIndex:  0,
		title:      "Clause 22 — extend board retirement age to 65",
		proposedBy: "person-004",
		proposedAt: "2026-09-15T11:05:00Z",
		status:     AmendmentStatusAccepted,
		summary:    "Extends the mandatory retirement age for the Fund's board members from 60 to 65, aligning it with the public-service ceiling.",
	},
	{
		id:         "amend-003",
		billIndex:  0,
		title:      "Clause 31 — quarterly investment reports",
		proposedBy: "person-001",
		proposedAt: "2026-09-16T09:15:00Z",
		status:     AmendmentStatusRejected,
		summary:    "Requires the Fund to publish quarterly (not annual) investment performance reports. Defeated on cost grounds.",
	},
	{
		id:         "amend-004",
		billIndex:  0,
		title:      "New Clause 34A — audited annual statements within 90 days",
		proposedBy: "person-005",
		proposedAt: "2026-09-16T14:40:00Z",
		status:     AmendmentStatusProposed,
		summary:    "Shortens the audited-statements publication window from 180 to 90 days after fiscal year-end, with statutory penalties for late filing.",
	},

	// --- Bill 1: The National Coroners Service Bill 2026 ---
	{
		id:         "amend-005",
		billIndex:  1,
		title:      "Clause 8 — independent coroner appointment panel",
		proposedBy: "person-004",
		proposedAt: "2026-09-20T10:00:00Z",
		status:     AmendmentStatusAccepted,
		summary:    "Replaces the executive appointment of the Chief Coroner with a 5-member independent panel chaired by the Public Service Commission.",
	},
	{
		id:         "amend-006",
		billIndex:  1,
		title:      "Clause 15 — mandatory inquest for unexplained deaths in custody",
		proposedBy: "person-005",
		proposedAt: "2026-09-20T11:25:00Z",
		status:     AmendmentStatusProposed,
		summary:    "Makes a public inquest mandatory (not discretionary) for any death in police or prison custody, with a 30-day statutory deadline.",
	},
	{
		id:         "amend-007",
		billIndex:  1,
		title:      "Clause 27 — victim family notification in 48 hours",
		proposedBy: "person-002",
		proposedAt: "2026-09-21T09:50:00Z",
		status:     AmendmentStatusRejected,
		summary:    "Requires next-of-kin notification within 48 hours of an inquestable death. Withdrawn after the AG flagged conflict with existing evidence-chain rules.",
	},

	// --- Bill 3: The Public Finance Management Amendment Bill 2026 ---
	{
		id:         "amend-008",
		billIndex:  3,
		title:      "Clause 12 — county budget statement within 14 days of approval",
		proposedBy: "person-001",
		proposedAt: "2026-09-12T10:15:00Z",
		status:     AmendmentStatusAccepted,
		summary:    "Tightens the county budget publication deadline from 21 to 14 days after assembly approval, with a late-filing penalty of 1% of the affected vote.",
	},
	{
		id:         "amend-009",
		billIndex:  3,
		title:      "Clause 18(3) — public participation quorum of 200",
		proposedBy: "person-004",
		proposedAt: "2026-09-12T14:00:00Z",
		status:     AmendmentStatusProposed,
		summary:    "Sets a minimum quorum of 200 members of the public for budget public-participation forums to be deemed validly held.",
	},
	{
		id:         "amend-010",
		billIndex:  3,
		title:      "Clause 25 — consolidated county borrowing registry",
		proposedBy: "person-002",
		proposedAt: "2026-09-13T09:30:00Z",
		status:     AmendmentStatusProposed,
		summary:    "Requires the National Treasury to publish a single, publicly searchable registry of every county government's outstanding loans + guarantees.",
	},
}

// sampleAmendments is built once at package init from sampleAmendmentSpecs
// + the live kenya_seed.SampleBills slice. The bill_id field is resolved
// to the actual SourceID (which is truncation-prone — see
// kenya_seed.makeSampleBill) so amendments stay attached to a Bill
// across slug-length drift. source_url is similarly resolved to the
// parent Bill's kenyalaw.org URL until the Hansard parser can mint
// per-sitting URLs (issue #19).
var sampleAmendments = func() []Amendment {
	out := make([]Amendment, 0, len(sampleAmendmentSpecs))
	for _, s := range sampleAmendmentSpecs {
		var billID, billURL string
		if s.billIndex >= 0 && s.billIndex < len(kenya_seed.SampleBills) {
			billID = kenya_seed.SampleBills[s.billIndex].SourceID
			billURL = kenya_seed.SampleBills[s.billIndex].URL
		}
		out = append(out, Amendment{
			ID:         s.id,
			BillID:     billID,
			Title:      s.title,
			ProposedBy: s.proposedBy,
			ProposedAt: s.proposedAt,
			Status:     s.status,
			Summary:    s.summary,
			SourceURL:  billURL,
		})
	}
	return out
}()

// findAmendmentsByBillID returns the seed amendments attached to the
// given bill_id. Substring matching is intentionally NOT used —
// amendments are tied to a specific Bill version (ADR-0011), so a
// partial-ID match would risk surfacing amendments for the wrong Bill.
// Returns an empty (non-nil) slice when no amendments are found or when
// billID is empty.
func findAmendmentsByBillID(billID string) []Amendment {
	out := []Amendment{}
	if billID == "" {
		return out
	}
	for _, a := range sampleAmendments {
		if a.BillID == billID {
			out = append(out, a)
		}
	}
	return out
}

// handleBillAmendments returns the amendments proposed against a single Bill.
//
// Route: GET /api/v1/bills/{id}/amendments
//
// Behavior:
//   - If the Bill ID is empty, returns 400 (bad_request).
//   - If the Bill cannot be found via live discovery AND is not in the
//     seed slice, returns 404 (not_found) — never 503, per the issue #265
//     contract that a known Bill must always return 200.
//   - If the adapter is unreachable, falls back to the seed Bills slice
//     to verify the Bill exists (issue #265).
//   - Returns the amendments array (may be empty when the Bill exists
//     but has no recorded amendments) + `source: "seed"`.
//   - Amendments are seed data pending the Hansard ingestion path
//     (issue #19); the seed data is marked as such on the wire so
//     callers can distinguish it from future live amendments.
func handleBillAmendments(w http.ResponseWriter, r *http.Request, adapter BillsAdapter, billID string) {
	if billID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "bill ID required")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	// Verify the Bill exists — try live discovery first, then fall back
	// to the seed slice (issue #265: never 503 for a known Bill).
	bill, err := findBillByID(ctx, adapter, billID)
	if err != nil {
		if err == errEmptyBillID {
			writeError(w, http.StatusBadRequest, "bad_request", err.Error())
			return
		}
		log.Printf("amendments handler: adapter error, falling back to seed: %v", err)
		// Adapter unreachable — try the seed slice directly (avoids a
		// second failed crawl via findBillByID's live path).
		bill = kenya_seed.FindSampleBillByID(billID)
	}
	if bill == nil {
		// Live discovery succeeded but no match — last resort: try the
		// seed slice before declaring the Bill unknown.
		if seedBill := kenya_seed.FindSampleBillByID(billID); seedBill != nil {
			bill = seedBill
		}
	}
	if bill == nil {
		writeError(w, http.StatusNotFound, "not_found", "bill not found: "+billID)
		return
	}

	// Look up amendments keyed on the resolved Bill's SourceID (which
	// is what /api/v1/bills/{id} returns) so the amendments filter
	// never misses a Bill whose ID differs from what the caller passed.
	amendments := findAmendmentsByBillID(bill.SourceID)

	// Amendments are seed-only for FEAT-15 (issue #19 will wire the live
	// ingestion path). Mark the source accordingly so callers can tell
	// the data is from the seed slice even when the Bill itself was
	// discovered live.
	writeJSON(w, http.StatusOK, map[string]any{
		"bill_id":    bill.SourceID,
		"amendments": amendments,
		"total":      len(amendments),
		"source":     "seed",
		"note":       "Amendments are seed data pending live Hansard ingestion (issue #19)",
	})
}
