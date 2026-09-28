// Package main — votes.go provides the MP voting-records API endpoints
// (issue #284 / task FEAT-6).
//
// The platform surfaces raw, per-division vote records (aye / nay /
// abstain / absent) for each MP, with every record carrying the source
// URL of the official Hansard / Votes-and-Proceedings entry it was
// derived from. The platform NEVER derives a "pass / fail" verdict or a
// "performance score" from the counts — they are surfaced raw (rule:
// NO_POLITICAL_PERFORMANCE_SCORE).
//
// Endpoints (registered in main.go):
//
//      GET /api/v1/people/{id}/votes   -- an MP's voting history
//      GET /api/v1/bills/{id}/votes    -- division summary for a single Bill
//
// The /people/{id}/votes handler is dispatched from handlePeople
// (parallel with /scorecard and /bills). The /bills/{id}/votes handler
// is dispatched from makeBillDetailHandler's sub-route switch (parallel
// with /timeline, /changes, /summary, /follow).
//
// Issue #284 acceptance bar: "Use seed data only. Do NOT parse actual
// PDFs." All 30 vote records are hand-curated seed data. When the
// verified ingestion path ships, the votes table will carry a FK to the
// bills table and this seed-only Bill-ID space (bill-vote-001..006)
// will be retired in favour of the canonical Bill IDs from
// kenya_seed.SampleBills.
package main

import (
        "net/http"
        "sort"
        "strings"
)

// VoteKind is the 4-valued enum for an MP's vote on a single division.
// The string values are the JSON wire forms (lowercase, no spaces) so
// the on-the-wire contract reads naturally for citizens + the frontend
// can switch on them directly.
type VoteKind string

const (
        // VoteAye: the MP voted in favour of the motion / Bill at this
        // division.
        VoteAye VoteKind = "aye"
        // VoteNay: the MP voted against the motion / Bill at this division.
        VoteNay VoteKind = "nay"
        // VoteAbstain: the MP was present but abstained from the vote.
        VoteAbstain VoteKind = "abstain"
        // VoteAbsent: the MP was not present for the division (recorded as
        // absent in the Votes-and-Proceedings). Distinct from abstain
        // because the MP did not participate at all.
        VoteAbsent VoteKind = "absent"
)

// allVoteKinds is the canonical ordered list of the 4 VoteKind values.
// Used by tests to assert every response item carries one of the 4
// allowed kinds (no typos, no silently-introduced "present" / "no"
// / "yes" variants).
var allVoteKinds = []VoteKind{VoteAye, VoteNay, VoteAbstain, VoteAbsent}

// isValidVoteKind returns true iff v is one of the 4 canonical kinds.
func isValidVoteKind(v VoteKind) bool {
        for _, k := range allVoteKinds {
                if k == v {
                        return true
                }
        }
        return false
}

// VoteRecord is a single MP's vote on a single Bill at a single division.
// Both the /people/{id}/votes and /bills/{id}/votes endpoints return
// the same VoteRecord shape so the frontend can render a vote row
// identically on either page (the per-MP view lists all of an MP's
// votes; the per-Bill view lists every MP's vote on that Bill).
//
// Every field is non-empty when sourced from a verified primary
// source. The platform does NOT fabricate vote records — every row
// cites its source_url.
type VoteRecord struct {
        PersonID   string   `json:"person_id"`
        PersonName string   `json:"person_name"`
        Vote       VoteKind `json:"vote"`
        BillID     string   `json:"bill_id"`
        BillTitle  string   `json:"bill_title"`
        Division   string   `json:"division"`
        Date       string   `json:"date"`
        SourceURL  string   `json:"source_url"`
}

// sampleVoteBill is the per-Bill metadata the seed slice attaches to
// every vote on that Bill. Kept as a separate struct (rather than
// inlining the fields on each VoteRecord) so the seed matrix can carry
// a single source of truth for the Bill title / division / date /
// source URL — every vote on the same Bill inherits the same metadata,
// which keeps the seed data drift-free.
type sampleVoteBill struct {
        BillID    string
        Title     string
        Division  string
        Date      string // ISO 8601 (YYYY-MM-DD) — Votes-and-Proceedings date
        SourceURL string
}

// sampleVoteBills is the 6-Bill seed slice for issue #284. The Bill IDs
// live in a separate ID space (bill-vote-001..006) from
// kenya_seed.SampleBills deliberately: the verified ingestion path for
// vote records has not shipped yet, so a citizen should NOT be able to
// follow a vote record back to a /bills/{id} detail page that pretends
// to know more about the Bill than the seed data actually carries.
// When the votes table is wired with a FK to the bills table, this
// seed-only ID space will be retired in favour of the canonical IDs.
//
// Dates are stored most-recent-first so the per-MP view
// (findVotesByPerson) can stream the matrix directly without a
// separate sort step.
//
// All Bill titles below are illustrative placeholders patterned on
// real Kenyan parliamentary divisions (Affordable Housing, PFM, CARB,
// etc.). Each is paired with a real division stage + an
// illustrative-but-stable source URL so the frontend can render the
// row even before the live Votes-and-Proceedings feed is wired.
var sampleVoteBills = []sampleVoteBill{
        {
                BillID:    "bill-vote-001",
                Title:     "Affordable Housing Bill",
                Division:  "Third Reading",
                Date:      "2024-09-10",
                SourceURL: "https://www.parliament.go.ke/the-national-assembly/votes-and-proceedings/2024-09-10",
        },
        {
                BillID:    "bill-vote-002",
                Title:     "Public Finance Management (Amendment) Bill",
                Division:  "Second Reading",
                Date:      "2024-08-28",
                SourceURL: "https://www.parliament.go.ke/the-national-assembly/votes-and-proceedings/2024-08-28",
        },
        {
                BillID:    "bill-vote-003",
                Title:     "Supplementary Appropriation Bill",
                Division:  "Committee Stage",
                Date:      "2024-08-21",
                SourceURL: "https://www.parliament.go.ke/the-national-assembly/votes-and-proceedings/2024-08-21",
        },
        {
                BillID:    "bill-vote-004",
                Title:     "County Allocation of Revenue Bill",
                Division:  "Third Reading",
                Date:      "2024-07-17",
                SourceURL: "https://www.parliament.go.ke/the-senate/votes-and-proceedings/2024-07-17",
        },
        {
                BillID:    "bill-vote-005",
                Title:     "Statute Law (Miscellaneous Amendments) Bill",
                Division:  "Second Reading",
                Date:      "2024-06-26",
                SourceURL: "https://www.parliament.go.ke/the-national-assembly/votes-and-proceedings/2024-06-26",
        },
        {
                BillID:    "bill-vote-006",
                Title:     "Sexual Offences (Amendment) Bill",
                Division:  "Second Reading",
                Date:      "2024-05-15",
                SourceURL: "https://www.parliament.go.ke/the-national-assembly/votes-and-proceedings/2024-05-15",
        },
}

// voteSeedRow is a single (person_id, bill_id, vote) entry in the seed
// matrix. The Bill metadata (title / division / date / source_url) is
// looked up from sampleVoteBills, and the person name is looked up
// from sampleScorecards, so neither has to be repeated per row.
type voteSeedRow struct {
        PersonID string
        BillID   string
        Vote     VoteKind
}

// voteSeedMatrix is the 30-row seed matrix: 5 sample MPs (person-001
// ..person-005, drawn from sampleScorecards) × 6 sample Bills
// (bill-vote-001..006, drawn from sampleVoteBills). Every Bill has all
// 4 vote kinds (aye / nay / abstain / absent) represented across the 5
// MPs so the /bills/{id}/votes summary surfaces non-zero counts in
// every cell — a citizen can see at a glance how the division split.
//
// The matrix is hand-curated rather than randomly generated so the
// counts are deterministic across runs (randomised data would make the
// regression tests flaky).
var voteSeedMatrix = []voteSeedRow{
        // Bill 1 — Affordable Housing Bill, Third Reading (2024-09-10)
        {PersonID: "person-001", BillID: "bill-vote-001", Vote: VoteAye},
        {PersonID: "person-002", BillID: "bill-vote-001", Vote: VoteNay},
        {PersonID: "person-003", BillID: "bill-vote-001", Vote: VoteAye},
        {PersonID: "person-004", BillID: "bill-vote-001", Vote: VoteAbstain},
        {PersonID: "person-005", BillID: "bill-vote-001", Vote: VoteAbsent},

        // Bill 2 — Public Finance Management (Amendment) Bill, Second Reading (2024-08-28)
        {PersonID: "person-001", BillID: "bill-vote-002", Vote: VoteAye},
        {PersonID: "person-002", BillID: "bill-vote-002", Vote: VoteNay},
        {PersonID: "person-003", BillID: "bill-vote-002", Vote: VoteAbstain},
        {PersonID: "person-004", BillID: "bill-vote-002", Vote: VoteAye},
        {PersonID: "person-005", BillID: "bill-vote-002", Vote: VoteAbsent},

        // Bill 3 — Supplementary Appropriation Bill, Committee Stage (2024-08-21)
        {PersonID: "person-001", BillID: "bill-vote-003", Vote: VoteNay},
        {PersonID: "person-002", BillID: "bill-vote-003", Vote: VoteAye},
        {PersonID: "person-003", BillID: "bill-vote-003", Vote: VoteAbsent},
        {PersonID: "person-004", BillID: "bill-vote-003", Vote: VoteAye},
        {PersonID: "person-005", BillID: "bill-vote-003", Vote: VoteAbstain},

        // Bill 4 — County Allocation of Revenue Bill, Third Reading (2024-07-17)
        {PersonID: "person-001", BillID: "bill-vote-004", Vote: VoteAye},
        {PersonID: "person-002", BillID: "bill-vote-004", Vote: VoteAbsent},
        {PersonID: "person-003", BillID: "bill-vote-004", Vote: VoteAbstain},
        {PersonID: "person-004", BillID: "bill-vote-004", Vote: VoteNay},
        {PersonID: "person-005", BillID: "bill-vote-004", Vote: VoteAbstain},

        // Bill 5 — Statute Law (Miscellaneous Amendments) Bill, Second Reading (2024-06-26)
        {PersonID: "person-001", BillID: "bill-vote-005", Vote: VoteNay},
        {PersonID: "person-002", BillID: "bill-vote-005", Vote: VoteAbstain},
        {PersonID: "person-003", BillID: "bill-vote-005", Vote: VoteAye},
        {PersonID: "person-004", BillID: "bill-vote-005", Vote: VoteNay},
        {PersonID: "person-005", BillID: "bill-vote-005", Vote: VoteAbsent},

        // Bill 6 — Sexual Offences (Amendment) Bill, Second Reading (2024-05-15)
        {PersonID: "person-001", BillID: "bill-vote-006", Vote: VoteAye},
        {PersonID: "person-002", BillID: "bill-vote-006", Vote: VoteNay},
        {PersonID: "person-003", BillID: "bill-vote-006", Vote: VoteAbstain},
        {PersonID: "person-004", BillID: "bill-vote-006", Vote: VoteAye},
        {PersonID: "person-005", BillID: "bill-vote-006", Vote: VoteAbsent},
}

// seedVoteRecords is the materialised, ready-to-serve slice of
// VoteRecord values. It is computed once at package init from the seed
// matrix + sampleScorecards + sampleVoteBills (the single sources of
// truth for MP names + Bill metadata respectively) so the handlers can
// stream it directly without rebuilding it on every request.
//
// The slice is sorted most-recent-first by Bill date so the per-MP view
// can be a simple linear filter over the slice (the /people/{id}/votes
// response contract is most-recent-first).
//
// Defensive: init panics if the matrix references an unknown
// person_id or bill_id. This is programmer error (the seed slice is
// hand-curated + the test suite cross-checks it), so failing loud at
// init is the correct posture — a silent skip would let the seed data
// drift out of sync with sampleScorecards / sampleVoteBills without
// any test catching it.
var seedVoteRecords = func() []VoteRecord {
        // Index sampleScorecards by PersonID for O(1) name lookup.
        personNameByID := make(map[string]string, len(sampleScorecards))
        for _, p := range sampleScorecards {
                personNameByID[p.PersonID] = p.Name
        }
        // Index sampleVoteBills by BillID for O(1) Bill-metadata lookup.
        billMetaByID := make(map[string]sampleVoteBill, len(sampleVoteBills))
        for _, b := range sampleVoteBills {
                billMetaByID[b.BillID] = b
        }
        out := make([]VoteRecord, 0, len(voteSeedMatrix))
        for _, row := range voteSeedMatrix {
                name, ok := personNameByID[row.PersonID]
                if !ok {
                        panic("votes: seed matrix references unknown person_id " + row.PersonID)
                }
                bill, ok := billMetaByID[row.BillID]
                if !ok {
                        panic("votes: seed matrix references unknown bill_id " + row.BillID)
                }
                if !isValidVoteKind(row.Vote) {
                        panic("votes: seed matrix has invalid VoteKind " + string(row.Vote))
                }
                out = append(out, VoteRecord{
                        PersonID:   row.PersonID,
                        PersonName: name,
                        Vote:       row.Vote,
                        BillID:     row.BillID,
                        BillTitle:  bill.Title,
                        Division:   bill.Division,
                        Date:       bill.Date,
                        SourceURL:  bill.SourceURL,
                })
        }
        // Most-recent-first by Bill date. sampleVoteBills is already stored
        // most-recent-first, but the seed matrix groups by Bill — so we
        // re-sort here to be robust against future re-orderings of either
        // slice. A stable sort preserves the matrix's per-Bill MP ordering.
        sort.SliceStable(out, func(i, j int) bool {
                return out[i].Date > out[j].Date
        })
        return out
}()

// findVotesByPerson returns every vote record for the given MP, sorted
// most-recent-first. Returns nil for an unknown person (the caller is
// responsible for surfacing 404 — see handleVotesByPerson).
func findVotesByPerson(personID string) []VoteRecord {
        out := make([]VoteRecord, 0, len(seedVoteRecords))
        for _, v := range seedVoteRecords {
                if v.PersonID == personID {
                        out = append(out, v)
                }
        }
        if len(out) == 0 {
                return nil
        }
        return out
}

// findVotesByBill returns every vote record for the given Bill. The
// order matches the seed matrix's per-Bill MP ordering (person-001
// first, person-005 last) so the per-Bill view is stable across
// requests. Returns nil for an unknown Bill.
func findVotesByBill(billID string) []VoteRecord {
        out := make([]VoteRecord, 0, 5)
        for _, v := range seedVoteRecords {
                if v.BillID == billID {
                        out = append(out, v)
                }
        }
        if len(out) == 0 {
                return nil
        }
        return out
}

// findPersonName returns the display name for the given person ID, or
// empty if the ID is unknown. Mirrors the lookupPerson helper in
// main.go but only returns the name (the votes handler doesn't need
// the scorecard URL — the per-MP response surfaces it separately).
func findPersonName(personID string) string {
        for _, p := range sampleScorecards {
                if p.PersonID == personID {
                        return p.Name
                }
        }
        return ""
}

// findSampleVoteBill returns the metadata for the given Bill ID, or nil
// if the ID is unknown.
func findSampleVoteBill(billID string) *sampleVoteBill {
        for i := range sampleVoteBills {
                if sampleVoteBills[i].BillID == billID {
                        return &sampleVoteBills[i]
                }
        }
        return nil
}

// VoteCounts is the per-division tally returned by /bills/{id}/votes.
// The platform NEVER derives a "pass / fail" verdict from these counts
// — they are surfaced raw (rule: NO_POLITICAL_PERFORMANCE_SCORE). A
// Bill with 0 aye + 0 nay is recorded as such; the citizen interprets
// the result, the platform does not.
type VoteCounts struct {
        Aye     int `json:"aye"`
        Nay     int `json:"nay"`
        Abstain int `json:"abstain"`
        Absent  int `json:"absent"`
        Total   int `json:"total"`
}

// tallyVotes counts the vote kinds in the given slice + returns a
// VoteCounts struct. Total is always the sum of aye + nay + abstain +
// absent (i.e. len(records)).
func tallyVotes(records []VoteRecord) VoteCounts {
        var c VoteCounts
        for _, r := range records {
                switch r.Vote {
                case VoteAye:
                        c.Aye++
                case VoteNay:
                        c.Nay++
                case VoteAbstain:
                        c.Abstain++
                case VoteAbsent:
                        c.Absent++
                }
        }
        c.Total = len(records)
        return c
}

// votesByPersonResponse is the JSON envelope returned by
// GET /api/v1/people/{id}/votes. The shape mirrors the
// billsByPersonResponse envelope from issue #282 so the frontend can
// reuse its rendering pattern (person_id + name + items + total +
// source + scorecard_url).
type votesByPersonResponse struct {
        PersonID     string       `json:"person_id"`
        Name         string       `json:"name"`
        Items        []VoteRecord `json:"items"`
        Total        int          `json:"total"`
        Source       string       `json:"source"` // always "seed" until ingestion ships
        ScorecardURL string       `json:"scorecard_url,omitempty"`
}

// votesByBillResponse is the JSON envelope returned by
// GET /api/v1/bills/{id}/votes. It carries the per-division tally
// (Counts) + the per-MP items list so the frontend can render both
// the headline summary (aye / nay / abstain / absent) and the
// detailed roll-call on a single page.
type votesByBillResponse struct {
        BillID    string       `json:"bill_id"`
        BillTitle string       `json:"bill_title"`
        Division  string       `json:"division"`
        Date      string       `json:"date"`
        SourceURL string       `json:"source_url"`
        Counts    VoteCounts   `json:"counts"`
        Items     []VoteRecord `json:"items"`
        Total     int          `json:"total"`
        Source    string       `json:"source"` // always "seed" until ingestion ships
}

// handleVotesByPerson is the handler for
// GET /api/v1/people/{id}/votes. It is dispatched from handlePeople
// (parallel with /scorecard and /bills). The MP must exist in
// sampleScorecards before any vote records are surfaced — this guards
// the "every datum is sourced" invariant (a typo in a person ID must
// 404, not silently return an empty items list that looks like a
// legitimate "no votes" response).
//
// On success: 200 with the items list (most-recent-first).
// On unknown person_id: 404.
func handleVotesByPerson(w http.ResponseWriter, r *http.Request, personID string) {
        if r.Method != http.MethodGet {
                writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                return
        }
        if personID == "" {
                writeError(w, http.StatusBadRequest, "bad_request", "person ID required")
                return
        }
        name := findPersonName(personID)
        if name == "" {
                writeError(w, http.StatusNotFound, "not_found", "person not found: "+personID)
                return
        }
        items := findVotesByPerson(personID)
        if items == nil {
                // Known MP with no vote records in the seed slice — return an
                // empty items list rather than nil so the JSON encodes as []
                // (not null). This branch is unreachable today (every sample
                // MP has 6 votes), but the guard keeps the contract stable
                // if the seed slice is ever pruned.
                items = []VoteRecord{}
        }
        writeJSON(w, http.StatusOK, votesByPersonResponse{
                PersonID:     personID,
                Name:         name,
                Items:        items,
                Total:        len(items),
                Source:       "seed",
                ScorecardURL: "/api/v1/people/" + personID + "/scorecard",
        })
}

// handleVotesByBill is the handler for GET /api/v1/bills/{id}/votes.
// It is dispatched from makeBillDetailHandler's sub-route switch
// (parallel with /timeline, /changes, /summary, /follow). The Bill
// must exist in sampleVoteBills before any vote records are surfaced
// — same "every datum is sourced" invariant as the per-MP handler.
//
// On success: 200 with the Counts tally + items list (per-MP roll
// call, in seed-matrix order).
// On unknown bill_id: 404.
//
// The platform NEVER derives a "pass / fail" verdict from the Counts —
// they are surfaced raw. A Bill with 0 aye + 0 nay is recorded as
// such; the citizen interprets the result.
func handleVotesByBill(w http.ResponseWriter, r *http.Request, billID string) {
        if r.Method != http.MethodGet {
                writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
                return
        }
        if billID == "" {
                writeError(w, http.StatusBadRequest, "bad_request", "bill ID required")
                return
        }
        bill := findSampleVoteBill(billID)
        if bill == nil {
                writeError(w, http.StatusNotFound, "not_found", "bill not found: "+billID)
                return
        }
        items := findVotesByBill(billID)
        if items == nil {
                items = []VoteRecord{}
        }
        writeJSON(w, http.StatusOK, votesByBillResponse{
                BillID:    bill.BillID,
                BillTitle: bill.Title,
                Division:  bill.Division,
                Date:      bill.Date,
                SourceURL: bill.SourceURL,
                Counts:    tallyVotes(items),
                Items:     items,
                Total:     len(items),
                Source:    "seed",
        })
}

// voteKindLabel returns the human-readable label for a VoteKind. Used
// by the test suite to assert the on-the-wire value matches the
// expected label. Kept here (rather than in the test file) so the
// contract is visible alongside the const declarations.
func voteKindLabel(v VoteKind) string {
        switch v {
        case VoteAye:
                return "Aye"
        case VoteNay:
                return "Nay"
        case VoteAbstain:
                return "Abstain"
        case VoteAbsent:
                return "Absent"
        default:
                return strings.Title(string(v))
        }
}
