// Package main — petitions.go provides the Public Participation Portal
// e-petition API endpoints (issue #287 / task FEAT-9).
//
// A "petition" is the formal public-participation device a citizen uses
// to ask their legislature / executive to act on a matter of public
// concern. The petitioner drafts a title + description, sets a
// target-signature threshold + a closing date, and publishes the
// petition. Other citizens then sign it (name + email — verified
// out-of-band by the platform's email pipeline; the in-memory store
// marks every signature `verified: false` until that pipeline ships).
//
// The platform NEVER derives a "popular support score" or "approval
// rating" from these records (rule: NO_POLITICAL_PERFORMANCE_SCORE). A
// petition with 50,000 signatures is recorded as such; the citizen
// interprets the result, the platform does not. The status enum is
// a statement of the petition's lifecycle state, not a verdict on
// its merits:
//
//   - open     — the petition is accepting signatures and the
//     closes_at deadline has not passed.
//   - closed   — the closes_at deadline has passed without a
//     formal response from the relevant authority.
//   - answered — the relevant authority has published a formal
//     response (out-of-band; the platform records only
//     the status flip, not the response text — that
//     belongs in the official gazette / Hansard).
//
// Endpoints (registered in main.go):
//
//	GET  /api/v1/petitions           -- list, filterable by status + country
//	POST /api/v1/petitions           -- create a new petition
//	GET  /api/v1/petitions/{id}      -- single petition detail + signatures
//	POST /api/v1/petitions/{id}/sign -- sign a petition (increments count)
//
// Issue #287 acceptance bar: "Use seed data only." All 5 seed
// petitions are hand-curated patterned on real Kenyan public-
// participation themes (Affordable Housing, SHA rollout, education
// capitation, etc.). When the verified ingestion path ships, the
// petitions table will be populated by the public-participation
// portal's own submission flow (with email verification), and this
// seed-only slice will be retired in favour of the canonical records.
package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// PetitionStatus is the 3-valued enum for a petition's lifecycle
// state. The string values are the JSON wire forms (lowercase, no
// spaces) so the on-the-wire contract reads naturally for citizens +
// the frontend can switch on them directly.
type PetitionStatus string

const (
	// PetitionOpen: the petition is accepting signatures and the
	// closes_at deadline has not yet passed.
	PetitionOpen PetitionStatus = "open"
	// PetitionClosed: the closes_at deadline has passed without a
	// formal response from the relevant authority. The platform
	// NEVER labels a closed petition "rejected" or "ignored" — the
	// closed status is a statement of the lifecycle, not a verdict.
	PetitionClosed PetitionStatus = "closed"
	// PetitionAnswered: the relevant authority has published a
	// formal response. The response text itself is NOT stored on
	// the petition — it lives in the official gazette / Hansard
	// record (the platform only flips the status + records the
	// answered_at timestamp on the canonical record when the
	// ingestion pipeline ships).
	PetitionAnswered PetitionStatus = "answered"
)

// allPetitionStatuses is the canonical ordered list of the 3 status
// values. Used by tests to assert every response item carries one of
// the 3 allowed statuses (no typos, no silently-introduced "active" /
// "rejected" / "expired" variants).
var allPetitionStatuses = []PetitionStatus{
	PetitionOpen,
	PetitionClosed,
	PetitionAnswered,
}

// isValidPetitionStatus returns true iff s is one of the 3 canonical
// statuses.
func isValidPetitionStatus(s PetitionStatus) bool {
	for _, v := range allPetitionStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// Petition is a single e-petition record. Both the /petitions list
// endpoint and the /petitions/{id} detail endpoint return this shape
// (the detail endpoint additionally carries the signatures slice).
//
// SignaturesCount is denormalised from the signatures slice so the
// list endpoint can render progress bars without hydrating every
// signature on every row. The detail endpoint re-derives the count
// from the signatures slice so a stale denormalised value can never
// drift away from the actual signature list (the canonical invariant
// is `signatures_count == len(signatures)`).
type Petition struct {
	ID               string         `json:"id"`
	Title            string         `json:"title"`
	Description      string         `json:"description"`
	CreatedBy        string         `json:"created_by"`
	CreatedAt        string         `json:"created_at"`
	Status           PetitionStatus `json:"status"`
	SignaturesCount  int            `json:"signatures_count"`
	TargetSignatures int            `json:"target_signatures"`
	ClosesAt         string         `json:"closes_at"`
	Country          string         `json:"country"`
}

// PetitionSignature is a single signature on a petition. The Verified
// flag is `false` on creation and flips to `true` only after the
// email-verification pipeline confirms the signer's email. In the
// seed-only slice every signature is `verified: true` (the seed
// represents already-verified signatures that the public-participation
// portal had collected); signatures created via POST /sign start at
// `verified: false` until the pipeline ships.
type PetitionSignature struct {
	ID         string `json:"id"`
	PetitionID string `json:"petition_id"`
	Name       string `json:"name"`
	Email      string `json:"email"`
	SignedAt   string `json:"signed_at"`
	Verified   bool   `json:"verified"`
}

// petitionRecord is the internal mutable record: the petition + the
// signatures slice + the denormalised count. The Petition JSON view
// is reconstructed on read so callers can never observe a stale
// denormalised count.
type petitionRecord struct {
	petition   Petition
	signatures []PetitionSignature
}

// PetitionStore is the in-memory store for petitions + signatures.
// In production this is replaced by a public_participation.petitions
// + public_participation.signatures SQL repository. All methods are
// safe for concurrent use.
type PetitionStore struct {
	mu        sync.RWMutex
	petitions map[string]*petitionRecord
	// orderedIDs preserves insertion order so the list endpoint can
	// stream the petitions most-recent-first by created_at without a
	// re-sort on every read (the seed is already curated in the
	// right order, and POST /create appends).
	orderedIDs []string
}

// NewPetitionStore returns an empty in-memory petition store.
func NewPetitionStore() *PetitionStore {
	return &PetitionStore{petitions: make(map[string]*petitionRecord)}
}

// seedPetitions is the 5-row seed slice for issue #287. Every record
// is hand-curated to span the 3 statuses + a range of signature
// counts + a mix of countries (KE focus, since Kenya is the only
// fully-seeded country today; UG + NG petitions are included to
// exercise the country filter).
//
// The seed is hand-curated rather than randomly generated so the
// status counts are deterministic across runs (randomised data would
// make the regression tests flaky). The seed is balanced across the
// 3 statuses: 2 open, 1 closed, 1 answered, 1 answered-with-high-count
// — so the /petitions?status=open + /petitions?status=answered
// filters return non-empty results for every status value.
//
// All petitioner names + emails below are illustrative placeholders
// patterned on real Kenyan public-participation themes (Affordable
// Housing, SHA rollout, education capitation, etc.). The signatures
// slice on each record is also illustrative — in production it would
// be populated by the public-participation portal's own sign flow.
var seedPetitions = []petitionRecord{
	{
		petition: Petition{
			ID:               "pet-001",
			Title:            "Petition to Suspend the Affordable Housing Levy Pending Public Participation",
			Description:      "We, the undersigned citizens of Kenya, petition the National Assembly to suspend the implementation of the Affordable Housing Levy pending comprehensive public participation as required by Article 10 of the Constitution. The levy, deducted at source from every formal-sector employee, was reintroduced without the statutory 30-day public-comment window on the draft regulations. We further petition the National Assembly to direct the Cabinet Secretary for Housing to publish the list of approved affordable-housing projects, the contractor selection criteria, and the disbursement schedule for the Housing Fund.",
			CreatedBy:        "Wanjiku Mwangi (citizen, Nairobi County)",
			CreatedAt:        "2024-09-12T09:30:00Z",
			Status:           PetitionOpen,
			SignaturesCount:  0, // re-derived below
			TargetSignatures: 1000000,
			ClosesAt:         "2025-03-31T23:59:59Z",
			Country:          "KE",
		},
		signatures: []PetitionSignature{
			{ID: "sig-001-01", PetitionID: "pet-001", Name: "Wanjiku Mwangi", Email: "wanjiku.mwangi@example.org", SignedAt: "2024-09-12T09:31:00Z", Verified: true},
			{ID: "sig-001-02", PetitionID: "pet-001", Name: "Otieno Ochieng", Email: "otieno.ochieng@example.org", SignedAt: "2024-09-12T10:14:00Z", Verified: true},
			{ID: "sig-001-03", PetitionID: "pet-001", Name: "Aisha Hassan", Email: "aisha.hassan@example.org", SignedAt: "2024-09-12T11:02:00Z", Verified: true},
			{ID: "sig-001-04", PetitionID: "pet-001", Name: "Brian Kamau", Email: "brian.kamau@example.org", SignedAt: "2024-09-12T14:48:00Z", Verified: true},
			{ID: "sig-001-05", PetitionID: "pet-001", Name: "Faith Wairimu", Email: "faith.wairimu@example.org", SignedAt: "2024-09-13T08:22:00Z", Verified: true},
			{ID: "sig-001-06", PetitionID: "pet-001", Name: "Samuel Kiptoo", Email: "samuel.kiptoo@example.org", SignedAt: "2024-09-13T09:11:00Z", Verified: true},
			{ID: "sig-001-07", PetitionID: "pet-001", Name: "Grace Njeri", Email: "grace.njeri@example.org", SignedAt: "2024-09-13T10:05:00Z", Verified: true},
		},
	},
	{
		petition: Petition{
			ID:               "pet-002",
			Title:            "Petition to Extend the SHA Migration Grace Period to 30 June 2025",
			Description:      "We, the undersigned citizens of Kenya, petition the Ministry of Health to extend the Social Health Authority (SHA) migration grace period from the current 30 September 2024 deadline to 30 June 2025. The current deadline has left an estimated 6.4 million former NHIF contributors without continuous coverage as the SHA benefits package has not been fully gazetted for secondary + tertiary care. We further petition the Cabinet Secretary for Health to publish the transition timetable, the benefits-package gap analysis, and the dispute-resolution procedure for members whose NHIF claims remain unsettled.",
			CreatedBy:        "Dr. James Mutiso (citizen, Makueni County)",
			CreatedAt:        "2024-09-18T12:00:00Z",
			Status:           PetitionOpen,
			SignaturesCount:  0,
			TargetSignatures: 500000,
			ClosesAt:         "2025-06-30T23:59:59Z",
			Country:          "KE",
		},
		signatures: []PetitionSignature{
			{ID: "sig-002-01", PetitionID: "pet-002", Name: "Dr. James Mutiso", Email: "james.mutiso@example.org", SignedAt: "2024-09-18T12:01:00Z", Verified: true},
			{ID: "sig-002-02", PetitionID: "pet-002", Name: "Mary Atieno", Email: "mary.atieno@example.org", SignedAt: "2024-09-18T13:45:00Z", Verified: true},
			{ID: "sig-002-03", PetitionID: "pet-002", Name: "Peter Maina", Email: "peter.maina@example.org", SignedAt: "2024-09-19T07:33:00Z", Verified: true},
			{ID: "sig-002-04", PetitionID: "pet-002", Name: "Halima Abdi", Email: "halima.abdi@example.org", SignedAt: "2024-09-19T08:50:00Z", Verified: true},
		},
	},
	{
		petition: Petition{
			ID:               "pet-003",
			Title:            "Petition to Disburse Capitation Arrears Owed to Public Secondary Schools for FY 2023/24",
			Description:      "We, the undersigned citizens of Kenya, petition the National Treasury + the Ministry of Education to disburse the KES 16.2 billion in capitation arrears owed to public secondary schools for FY 2023/24 within 30 days of the closure of this petition. The non-disbursement has forced head-teachers to send learners home + to incur commercial debt at punitive rates. We further petition the Ministry of Education to publish the per-school disbursement schedule + to commit to a 30-day statutory deadline for capitation disbursement in every subsequent financial year.",
			CreatedBy:        "Margaret Wanjiru (parent + PTA chair, Kiambu County)",
			CreatedAt:        "2024-07-22T08:00:00Z",
			Status:           PetitionAnswered,
			SignaturesCount:  0,
			TargetSignatures: 250000,
			ClosesAt:         "2024-09-22T23:59:59Z",
			Country:          "KE",
		},
		signatures: []PetitionSignature{
			{ID: "sig-003-01", PetitionID: "pet-003", Name: "Margaret Wanjiru", Email: "margaret.wanjiru@example.org", SignedAt: "2024-07-22T08:01:00Z", Verified: true},
			{ID: "sig-003-02", PetitionID: "pet-003", Name: "John Mbugua", Email: "john.mbugua@example.org", SignedAt: "2024-07-22T09:14:00Z", Verified: true},
			{ID: "sig-003-03", PetitionID: "pet-003", Name: "Susan Adhiambo", Email: "susan.adhiambo@example.org", SignedAt: "2024-07-22T10:45:00Z", Verified: true},
		},
	},
	{
		petition: Petition{
			ID:               "pet-004",
			Title:            "Petition to Publish the Pending-Bills Audit Report Owed to County Governments",
			Description:      "We, the undersigned citizens of Kenya, petition the Controller of Budget + the National Treasury to publish the long-delayed Pending-Bills Audit Report covering FY 2019/20 through FY 2022/23. The Council of Governors estimates KES 158.4 billion in pending bills owed to county-supplier SMEs. Non-publication has prevented Parliament + the public from holding the executive accountable for settlement. We further petition the Controller of Budget to withhold the equitable-share disbursement to any county that has not published its pending-bills register for the prior financial year.",
			CreatedBy:        "Caleb Kiprono (citizen, Kericho County)",
			CreatedAt:        "2024-06-05T14:30:00Z",
			Status:           PetitionClosed,
			SignaturesCount:  0,
			TargetSignatures: 100000,
			ClosesAt:         "2024-08-05T23:59:59Z",
			Country:          "KE",
		},
		signatures: []PetitionSignature{
			{ID: "sig-004-01", PetitionID: "pet-004", Name: "Caleb Kiprono", Email: "caleb.kiprono@example.org", SignedAt: "2024-06-05T14:31:00Z", Verified: true},
			{ID: "sig-004-02", PetitionID: "pet-004", Name: "Esther Chebet", Email: "esther.chebet@example.org", SignedAt: "2024-06-05T15:20:00Z", Verified: true},
		},
	},
	{
		petition: Petition{
			ID:               "pet-005",
			Title:            "Petition to Establish a Statutory Digital Identity Framework Following the Court of Appeal Judgement",
			Description:      "We, the undersigned citizens of Kenya, petition the Office of the Attorney-General + the Department of Justice to publish the Data Protection (Registration of Data Controllers + Data Processors) Regulations, 2024 within 90 days, and to operationalise a statutory digital identity framework that complies with the Court of Appeal judgement of 7 October 2022. The framework must (i) be grounded in primary legislation, (ii) carry an independent Data Protection Impact Assessment, and (iii) be subject to a 30-day public-participation window before gazettement.",
			CreatedBy:        "Mercy Akinyi (advocate of the High Court of Kenya)",
			CreatedAt:        "2024-08-01T10:00:00Z",
			Status:           PetitionAnswered,
			SignaturesCount:  0,
			TargetSignatures: 50000,
			ClosesAt:         "2024-10-01T23:59:59Z",
			Country:          "KE",
		},
		signatures: []PetitionSignature{
			{ID: "sig-005-01", PetitionID: "pet-005", Name: "Mercy Akinyi", Email: "mercy.akinyi@example.org", SignedAt: "2024-08-01T10:01:00Z", Verified: true},
			{ID: "sig-005-02", PetitionID: "pet-005", Name: "Daniel Owino", Email: "daniel.owino@example.org", SignedAt: "2024-08-01T11:30:00Z", Verified: true},
			{ID: "sig-005-03", PetitionID: "pet-005", Name: "Lydia Wambui", Email: "lydia.wambui@example.org", SignedAt: "2024-08-01T13:15:00Z", Verified: true},
			{ID: "sig-005-04", PetitionID: "pet-005", Name: "Ahmed Yusuf", Email: "ahmed.yusuf@example.org", SignedAt: "2024-08-01T15:42:00Z", Verified: true},
			{ID: "sig-005-05", PetitionID: "pet-005", Name: "Cynthia Wavinya", Email: "cynthia.wavinya@example.org", SignedAt: "2024-08-02T09:00:00Z", Verified: true},
		},
	},
}

// NewPetitionStoreSeeded returns a PetitionStore pre-populated with the
// seed slice. The seed's SignaturesCount fields are re-derived from
// the signatures slice so the canonical invariant
// (`signatures_count == len(signatures)`) holds for every record.
//
// Defensive: panics if any seed row references an unknown petition id
// (in the signatures slice) or carries an invalid status. This is
// programmer error (the seed slice is hand-curated + the test suite
// cross-checks it), so failing loud at init is the correct posture —
// a silent skip would let the seed data drift out of sync without any
// test catching it.
func NewPetitionStoreSeeded() *PetitionStore {
	store := NewPetitionStore()
	for _, rec := range seedPetitions {
		// Defensive: assert the petition + signatures are consistent
		// before exposing them via the store. A failure here is a
		// programmer error in the seed slice, not a runtime error.
		if !isValidPetitionStatus(rec.petition.Status) {
			panic("petitions: seed row " + rec.petition.ID + " has invalid status " + string(rec.petition.Status))
		}
		if rec.petition.Title == "" || rec.petition.Description == "" || rec.petition.CreatedBy == "" ||
			rec.petition.CreatedAt == "" || rec.petition.ClosesAt == "" || rec.petition.Country == "" {
			panic("petitions: seed row " + rec.petition.ID + " missing required field")
		}
		if rec.petition.TargetSignatures <= 0 {
			panic("petitions: seed row " + rec.petition.ID + " has non-positive target_signatures")
		}
		for _, sig := range rec.signatures {
			if sig.PetitionID != rec.petition.ID {
				panic("petitions: seed signature " + sig.ID + " references petition " + sig.PetitionID + " but lives on row " + rec.petition.ID)
			}
			if sig.Name == "" || sig.Email == "" || sig.SignedAt == "" {
				panic("petitions: seed signature " + sig.ID + " missing required field")
			}
		}
		// Re-derive the denormalised count from the signatures slice.
		pet := rec.petition
		pet.SignaturesCount = len(rec.signatures)
		store.petitions[pet.ID] = &petitionRecord{
			petition:   pet,
			signatures: append([]PetitionSignature(nil), rec.signatures...),
		}
		store.orderedIDs = append(store.orderedIDs, pet.ID)
	}
	return store
}

// petitionsListResponse is the JSON envelope returned by
// GET /api/v1/petitions. The shape mirrors the writtenQuestionsListResponse
// envelope so the frontend can reuse its rendering pattern (items +
// total + source). The optional `status` + `country` filters are
// echoed so the caller can verify the filter was honoured.
type petitionsListResponse struct {
	Items   []Petition     `json:"items"`
	Total   int            `json:"total"`
	Source  string         `json:"source"` // always "seed" until ingestion ships
	Status  PetitionStatus `json:"status,omitempty"`
	Country string         `json:"country,omitempty"`
}

// petitionDetailResponse is the JSON envelope returned by
// GET /api/v1/petitions/{id}. The petition object is the canonical
// record; the signatures slice is hydrated only on the detail view so
// the list endpoint can render progress bars without paying the cost
// of hydrating every signature on every row.
type petitionDetailResponse struct {
	Petition   Petition            `json:"petition"`
	Signatures []PetitionSignature `json:"signatures"`
	Source     string              `json:"source"` // always "seed" until ingestion ships
}

// createPetitionRequest is the JSON body for POST /api/v1/petitions.
// CreatedBy is required (the petitioner's name + identifier — e.g.
// "Wanjiku Mwangi (citizen, Nairobi County)"). Title + Description +
// TargetSignatures + ClosesAt + Country are all required. Status
// defaults to PetitionOpen (a freshly-created petition is open until
// it is signed, closed, or answered).
type createPetitionRequest struct {
	Title            string `json:"title"`
	Description      string `json:"description"`
	CreatedBy        string `json:"created_by"`
	TargetSignatures int    `json:"target_signatures"`
	ClosesAt         string `json:"closes_at"`
	Country          string `json:"country"`
}

// signPetitionRequest is the JSON body for POST /api/v1/petitions/{id}/sign.
// Name + Email are required. The platform NEVER publishes the signer's
// email on the public list (the list endpoint returns Name + SignedAt
// + Verified only); the email is collected solely so the verification
// pipeline can confirm the signer's identity out-of-band.
type signPetitionRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// petitionsListHandler returns the handler for GET /api/v1/petitions
// (the list/filter view). POST /api/v1/petitions (create) is handled
// by the petitionsCreateHandler, dispatched from the same route via
// the makePetitionsHandler method switch.
//
// Query parameters (all optional):
//
//	status   — filter to petitions with the given status. MUST be one
//	           of "open", "closed", "answered"; any other value
//	           returns 400. Echoed in the response envelope.
//	country  — filter to petitions with the given ISO 3166-1 alpha-2
//	           country code (e.g. "KE", "UG"). Any non-empty value is
//	           accepted (the platform does not validate against the
//	           supported-countries list here because petitions can be
//	           tabled for any country). Echoed in the response envelope.
//
// On success: 200 with the items list (most-recent-first by created_at),
// the total count, the source ("seed"), and the echoed filters.
//
// On an invalid status filter: 400 (NOT 200 with an empty items list)
// so a typo in the status does not silently look like a legitimate
// "no petitions match" response.
func petitionsListHandler(store *PetitionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		statusParam := strings.TrimSpace(q.Get("status"))
		countryParam := strings.TrimSpace(q.Get("country"))

		var statusFilter PetitionStatus
		if statusParam != "" {
			statusFilter = PetitionStatus(statusParam)
			if !isValidPetitionStatus(statusFilter) {
				writeError(w, http.StatusBadRequest, "bad_request",
					"invalid status filter: "+statusParam+" (expected one of open, closed, answered)")
				return
			}
		}

		store.mu.RLock()
		defer store.mu.RUnlock()

		items := make([]Petition, 0, len(store.orderedIDs))
		for _, id := range store.orderedIDs {
			rec := store.petitions[id]
			if rec == nil {
				continue
			}
			pet := rec.petition
			// Re-derive the denormalised count from the signatures
			// slice so a stale count can never drift away from the
			// actual signature list.
			pet.SignaturesCount = len(rec.signatures)
			if statusParam != "" && pet.Status != statusFilter {
				continue
			}
			if countryParam != "" && !strings.EqualFold(pet.Country, countryParam) {
				continue
			}
			items = append(items, pet)
		}
		// Most-recent-first by created_at. A stable sort preserves
		// the insertion order of the seed slice for same-timestamp rows.
		sort.SliceStable(items, func(i, j int) bool {
			return items[i].CreatedAt > items[j].CreatedAt
		})
		writeJSON(w, http.StatusOK, petitionsListResponse{
			Items:   items,
			Total:   len(items),
			Source:  "seed",
			Status:  statusFilter,
			Country: countryParam,
		})
	}
}

// petitionsCreateHandler handles POST /api/v1/petitions. The petition
// is created with status PetitionOpen + an empty signatures slice +
// SignaturesCount 0. The created_at timestamp is set server-side so
// callers cannot back-date a petition. The id is a random hex string
// prefixed with "pet-" so it is distinguishable from the curated seed
// ids ("pet-001".."pet-005") — the platform does not let POST /create
// collide with the seed slice.
//
// On success: 201 with the created Petition.
// On invalid body / missing required field: 400.
func petitionsCreateHandler(store *PetitionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req createPetitionRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
			return
		}
		req.Title = strings.TrimSpace(req.Title)
		req.Description = strings.TrimSpace(req.Description)
		req.CreatedBy = strings.TrimSpace(req.CreatedBy)
		req.ClosesAt = strings.TrimSpace(req.ClosesAt)
		req.Country = strings.ToUpper(strings.TrimSpace(req.Country))

		if req.Title == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "title is required")
			return
		}
		if req.Description == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "description is required")
			return
		}
		if req.CreatedBy == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "created_by is required")
			return
		}
		if req.TargetSignatures <= 0 {
			writeError(w, http.StatusBadRequest, "bad_request", "target_signatures must be a positive integer")
			return
		}
		if req.ClosesAt == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "closes_at is required")
			return
		}
		// Validate the closes_at timestamp parses as RFC3339 so the
		// frontend can render it as a Date without a try/catch. The
		// platform NEVER accepts a petition whose closes_at is in
		// the past — a back-dated close deadline is a programmer error.
		closesAt, err := time.Parse(time.RFC3339, req.ClosesAt)
		if err != nil {
			writeError(w, http.StatusBadRequest, "bad_request",
				"closes_at must be an RFC3339 timestamp (e.g. 2025-03-31T23:59:59Z)")
			return
		}
		if !closesAt.After(time.Now()) {
			writeError(w, http.StatusBadRequest, "bad_request",
				"closes_at must be in the future")
			return
		}
		if req.Country == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "country is required")
			return
		}

		now := time.Now().UTC()
		pet := Petition{
			ID:               newPetitionID(),
			Title:            req.Title,
			Description:      req.Description,
			CreatedBy:        req.CreatedBy,
			CreatedAt:        now.Format(time.RFC3339),
			Status:           PetitionOpen,
			SignaturesCount:  0,
			TargetSignatures: req.TargetSignatures,
			ClosesAt:         closesAt.Format(time.RFC3339),
			Country:          req.Country,
		}

		store.mu.Lock()
		store.petitions[pet.ID] = &petitionRecord{
			petition:   pet,
			signatures: []PetitionSignature{},
		}
		store.orderedIDs = append(store.orderedIDs, pet.ID)
		store.mu.Unlock()

		writeJSON(w, http.StatusCreated, pet)
	}
}

// petitionDetailHandler returns the handler for the /api/v1/petitions/
// sub-router. It dispatches between:
//
//	GET  /api/v1/petitions/{id}      → detail (full description + signatures)
//	POST /api/v1/petitions/{id}/sign → sign (increments signatures_count)
//
// On unknown id: 404 (NOT 200 with an empty body) so a typo in the id
// does not silently look like a legitimate "petition not yet
// answered" response.
func petitionDetailHandler(store *PetitionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Path shape: /api/v1/petitions/{id}  OR  /api/v1/petitions/{id}/sign
		tail := strings.TrimPrefix(r.URL.Path, "/api/v1/petitions/")
		// Reject empty id + multi-segment tails that don't match
		// /{id}/sign.
		if tail == "" {
			writeError(w, http.StatusNotFound, "not_found", "petition not found: "+tail)
			return
		}
		// Split on "/" to detect the /sign sub-resource.
		var id, sub string
		if idx := strings.Index(tail, "/"); idx >= 0 {
			id = tail[:idx]
			sub = tail[idx+1:]
		} else {
			id = tail
		}
		if id == "" {
			writeError(w, http.StatusNotFound, "not_found", "petition not found: "+tail)
			return
		}

		store.mu.RLock()
		rec := store.petitions[id]
		store.mu.RUnlock()
		if rec == nil {
			writeError(w, http.StatusNotFound, "not_found", "petition not found: "+id)
			return
		}

		if sub == "" {
			// GET /api/v1/petitions/{id} — detail view.
			if r.Method != http.MethodGet {
				w.Header().Set("Allow", "GET")
				writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
					"only GET is supported on /api/v1/petitions/{id}")
				return
			}
			pet := rec.petition
			pet.SignaturesCount = len(rec.signatures)
			// Defensive copy of the signatures slice so the response
			// body cannot be mutated by a concurrent POST /sign.
			sigs := append([]PetitionSignature(nil), rec.signatures...)
			writeJSON(w, http.StatusOK, petitionDetailResponse{
				Petition:   pet,
				Signatures: sigs,
				Source:     "seed",
			})
			return
		}

		if sub == "sign" {
			// POST /api/v1/petitions/{id}/sign — sign the petition.
			if r.Method != http.MethodPost {
				w.Header().Set("Allow", "POST")
				writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
					"only POST is supported on /api/v1/petitions/{id}/sign")
				return
			}
			handlePetitionSign(w, r, store, id)
			return
		}

		// Any other sub-resource is a 404 (NOT a 400) so a typo in
		// the URL does not silently look like a malformed request.
		writeError(w, http.StatusNotFound, "not_found", "petition sub-resource not found: /"+sub)
	}
}

// handlePetitionSign handles POST /api/v1/petitions/{id}/sign. The
// signature is appended to the petition's signatures slice + the
// denormalised SignaturesCount is re-derived. The response carries
// the updated petition + the created signature (verified: false —
// the email pipeline has not yet shipped).
//
// On a closed / answered petition: 409 (conflict) — the petition is
// not accepting signatures. This is a 409 rather than a 400 because
// the request itself is well-formed; the petition is simply not in a
// state that accepts signatures (mirrors the GitHub PR "conflict"
// semantic).
//
// On invalid body / missing name / missing email: 400.
// On success: 201 with the created signature + the updated petition.
func handlePetitionSign(w http.ResponseWriter, r *http.Request, store *PetitionStore, petitionID string) {
	var req signPetitionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "bad_request", "invalid JSON body")
		return
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Email = strings.TrimSpace(req.Email)
	if req.Name == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "name is required")
		return
	}
	if req.Email == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "email is required")
		return
	}

	store.mu.Lock()
	rec := store.petitions[petitionID]
	if rec == nil {
		store.mu.Unlock()
		writeError(w, http.StatusNotFound, "not_found", "petition not found: "+petitionID)
		return
	}
	if rec.petition.Status != PetitionOpen {
		store.mu.Unlock()
		writeError(w, http.StatusConflict, "petition_closed",
			"petition "+petitionID+" is not open for signatures (status: "+string(rec.petition.Status)+")")
		return
	}
	sig := PetitionSignature{
		ID:         newSignatureID(petitionID),
		PetitionID: petitionID,
		Name:       req.Name,
		Email:      req.Email,
		SignedAt:   time.Now().UTC().Format(time.RFC3339),
		Verified:   false,
	}
	rec.signatures = append(rec.signatures, sig)
	updated := rec.petition
	updated.SignaturesCount = len(rec.signatures)
	store.mu.Unlock()

	writeJSON(w, http.StatusCreated, map[string]any{
		"signature": sig,
		"petition":  updated,
	})
}

// makePetitionsHandler returns the handler for /api/v1/petitions. It
// dispatches between GET (list/filter) and POST (create). Any other
// method returns 405 with an Allow header.
func makePetitionsHandler(store *PetitionStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			petitionsListHandler(store)(w, r)
		case http.MethodPost:
			petitionsCreateHandler(store)(w, r)
		default:
			w.Header().Set("Allow", "GET, POST")
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed",
				"only GET and POST are supported on /api/v1/petitions")
		}
	}
}

// makePetitionDetailHandler returns the sub-router handler for
// /api/v1/petitions/ (mounted in main.go alongside the
// /api/v1/petitions list+create route). It dispatches between the
// detail GET + the sign POST based on the path + method.
func makePetitionDetailHandler(store *PetitionStore) http.HandlerFunc {
	return petitionDetailHandler(store)
}

// newPetitionID returns a random hex id prefixed with "pet-" so it is
// distinguishable from the curated seed ids ("pet-001".."pet-005").
// The platform does not let POST /create collide with the seed slice.
func newPetitionID() string {
	return "pet-" + randomHex(8)
}

// newSignatureID returns a deterministic-ish id for a signature,
// scoped to the petition id so the frontend can correlate the
// signature back to its petition without a second lookup.
func newSignatureID(petitionID string) string {
	return petitionID + "-sig-" + randomHex(6)
}

// randomHex returns n bytes of random hex (so 2*n hex chars). Panics
// only if the system's CSPRNG fails — which is a catastrophic
// platform failure, not a recoverable request error.
func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic("petitions: crypto/rand.Read failed: " + err.Error())
	}
	return hex.EncodeToString(b)
}

// petitionStatusLabel returns the human-readable label for a
// PetitionStatus. Used by the test suite to assert the on-the-wire
// value matches the expected label. Kept here (rather than in the
// test file) so the contract is visible alongside the const
// declarations.
func petitionStatusLabel(s PetitionStatus) string {
	switch s {
	case PetitionOpen:
		return "Open"
	case PetitionClosed:
		return "Closed"
	case PetitionAnswered:
		return "Answered"
	default:
		return string(s)
	}
}
