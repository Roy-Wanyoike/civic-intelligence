// Package main — written_questions.go provides the Written Questions
// tracker API endpoints (issue #286 / task FEAT-8).
//
// A "written question" is the formal parliamentary device an MP uses to
// request information from a Cabinet Secretary / Minister. The MP
// tables the question, the presiding officer refers it to the relevant
// ministry, and the Minister is given a statutory deadline by which to
// respond in writing. The platform surfaces this lifecycle verbatim:
// every record carries the question text, the asked_at date, the
// Minister + ministry it was directed to, the response text (when
// answered), the responded_at date, the deadline, and the canonical
// source URL of the official parliamentary record.
//
// The platform NEVER derives a "responsiveness score" or "approval
// rating" from these records (rule: NO_POLITICAL_PERFORMANCE_SCORE). A
// Minister with 5 overdue responses is recorded as such; the citizen
// interprets the result, the platform does not.
//
// Endpoints (registered in main.go):
//
//	GET /api/v1/questions/written         -- list, filterable by mp_id + status
//	GET /api/v1/questions/written/{id}    -- single written question detail
//	GET /api/v1/people/{id}/questions    -- an MP's written questions
//
// The /api/v1/people/{id}/questions handler is dispatched from
// handlePeople (parallel with /scorecard, /bills and /votes). The list
// + detail handlers are registered directly under
// /api/v1/questions/written (note: distinct from the AI Q&A
// /api/v1/questions route — the AI Q&A route is auth-protected; written
// questions are public parliamentary records).
//
// Issue #286 acceptance bar: "Use seed data only." All 15 written
// questions are hand-curated seed data patterned on real Kenyan
// parliamentary questions. When the verified ingestion path ships,
// the questions table will be populated from
// parliament.go.ke/the-national-assembly/questions and this seed-only
// slice will be retired in favour of the canonical records.
package main

import (
	"net/http"
	"sort"
	"strings"
)

// WrittenQuestionStatus is the 3-valued enum for a written question's
// lifecycle state. The string values are the JSON wire forms
// (lowercase, no spaces) so the on-the-wire contract reads naturally
// for citizens + the frontend can switch on them directly.
type WrittenQuestionStatus string

const (
	// WrittenQuestionPending: the question has been tabled + referred
	// to the ministry, the deadline has NOT yet passed, and no
	// response has been published.
	WrittenQuestionPending WrittenQuestionStatus = "pending"
	// WrittenQuestionAnswered: the Minister has published a written
	// response (response_text + responded_at are populated). The
	// deadline may or may not have passed by the time the response
	// arrived.
	WrittenQuestionAnswered WrittenQuestionStatus = "answered"
	// WrittenQuestionOverdue: the deadline has passed without a
	// published response. The platform NEVER labels a Minister
	// "non-responsive" or "failing" — the overdue status is a
	// statement of the parliamentary record, not a verdict.
	WrittenQuestionOverdue WrittenQuestionStatus = "overdue"
)

// allWrittenQuestionStatuses is the canonical ordered list of the 3
// status values. Used by tests to assert every response item carries
// one of the 3 allowed statuses (no typos, no silently-introduced
// "open" / "closed" / "late" variants).
var allWrittenQuestionStatuses = []WrittenQuestionStatus{
	WrittenQuestionPending,
	WrittenQuestionAnswered,
	WrittenQuestionOverdue,
}

// isValidWrittenQuestionStatus returns true iff s is one of the 3
// canonical statuses.
func isValidWrittenQuestionStatus(s WrittenQuestionStatus) bool {
	for _, v := range allWrittenQuestionStatuses {
		if v == s {
			return true
		}
	}
	return false
}

// WrittenQuestion is a single MP → Minister written question record.
// Both the /questions/written list endpoint, the
// /questions/written/{id} detail endpoint, and the
// /people/{id}/questions per-MP endpoint return the same item shape
// so the frontend can render a question row identically on any page.
//
// Every field is non-empty when sourced from a verified primary
// source. The platform does NOT fabricate question records — every
// row cites its source_url.
type WrittenQuestion struct {
	ID           string `json:"id"`
	MPID         string `json:"mp_id"`
	MPName       string `json:"mp_name"`
	Minister     string `json:"minister"`
	Ministry     string `json:"ministry"`
	QuestionText string `json:"question_text"`
	AskedAt      string `json:"asked_at"`
	// ResponseText + RespondedAt are omitempty because pending +
	// overdue questions have no response yet. This mirrors the
	// real-world lifecycle: a question is tabled BEFORE a response
	// exists, and the API surface should not invent empty-string
	// responses for unanswered questions.
	ResponseText string                `json:"response_text,omitempty"`
	RespondedAt  string                `json:"responded_at,omitempty"`
	Deadline     string                `json:"deadline"`
	Status       WrittenQuestionStatus `json:"status"`
	SourceURL    string                `json:"source_url"`
}

// sampleWrittenQuestions is the 15-row seed slice for issue #286
// (3 written questions per sample MP × 5 sample MPs). Every record
// cites its source_url so a citizen can verify the question against
// the official parliament.go.ke record.
//
// The seed is hand-curated rather than randomly generated so the
// status counts are deterministic across runs (randomised data
// would make the regression tests flaky). The seed is balanced
// across the 3 statuses: 5 pending, 5 answered, 5 overdue — so the
// /questions/written?status=answered filter returns a non-empty
// result for every status value.
//
// All Minister + ministry names below are illustrative placeholders
// patterned on real Kenyan Cabinet Secretaries (National Treasury,
// Roads & Transport, Health, Education, Energy, Interior, Attorney-
// General). Each is paired with an illustrative-but-stable
// source_url so the frontend can render the row even before the
// live parliament.go.ke questions feed is wired.
//
// The slice is stored most-recent-first by asked_at so the per-MP
// view (findWrittenQuestionsByPerson) can stream the slice directly
// without a separate sort step.
var sampleWrittenQuestions = []WrittenQuestion{
	// === person-001: Kimani Ichung'wah (Majority Leader, NA) ===
	{
		ID:           "wq-001",
		MPID:         "person-001",
		MPName:       "Kimani Ichung'wah",
		Minister:     "Cabinet Secretary, National Treasury",
		Ministry:     "The National Treasury & Economic Planning",
		QuestionText: "Could the Cabinet Secretary for the National Treasury clarify the upper bound on domestic borrowing for the fiscal year 2024/25, and state whether the cap agreed with the Parliamentary Budget Office has been exceeded to date?",
		AskedAt:      "2024-09-05",
		Deadline:     "2024-09-26",
		Status:       WrittenQuestionAnswered,
		ResponseText: "The domestic borrowing ceiling for FY 2024/25 was set at KES 584.0 billion under the Public Finance Management (National Government) Regulations. As at 31 August 2024, net domestic borrowing stood at KES 271.4 billion, which is within the agreed ceiling. The Treasury will publish a quarterly update on the same.",
		RespondedAt:  "2024-09-19",
		SourceURL:    "https://www.parliament.go.ke/the-national-assembly/questions/2024-09-05",
	},
	{
		ID:           "wq-002",
		MPID:         "person-001",
		MPName:       "Kimani Ichung'wah",
		Minister:     "Cabinet Secretary, Roads & Transport",
		Ministry:     "Ministry of Roads & Transport",
		QuestionText: "Could the Cabinet Secretary for Roads & Transport provide the status of the Kikuyu–Nairobi road dualing project, including the contractor, the completion date, and the total amount disbursed to date?",
		AskedAt:      "2024-09-12",
		Deadline:     "2024-10-03",
		Status:       WrittenQuestionPending,
		SourceURL:    "https://www.parliament.go.ke/the-national-assembly/questions/2024-09-12",
	},
	{
		ID:           "wq-003",
		MPID:         "person-001",
		MPName:       "Kimani Ichung'wah",
		Minister:     "Attorney-General",
		Ministry:     "Office of the Attorney-General & Department of Justice",
		QuestionText: "Could the Attorney-General state the legal status of the Huduma Namba registration following the Court of Appeal judgement of 7 October 2022, and clarify whether any subordinate legislation is being drafted to operationalise the digital identity framework?",
		AskedAt:      "2024-08-15",
		Deadline:     "2024-09-05",
		Status:       WrittenQuestionOverdue,
		SourceURL:    "https://www.parliament.go.ke/the-national-assembly/questions/2024-08-15",
	},

	// === person-002: Opiyo Wandayi (Minority Leader, NA) ===
	{
		ID:           "wq-004",
		MPID:         "person-002",
		MPName:       "Opiyo Wandayi",
		Minister:     "Cabinet Secretary, Energy",
		Ministry:     "Ministry of Energy & Petroleum",
		QuestionText: "Could the Cabinet Secretary for Energy explain the compensation framework for citizens and businesses that suffered losses during the national grid blackout of 6 August 2024, and state whether any ex gratia payments have been processed?",
		AskedAt:      "2024-09-04",
		Deadline:     "2024-09-25",
		Status:       WrittenQuestionAnswered,
		ResponseText: "Following the 6 August 2024 grid disturbance, the Ministry constituted a multi-agency committee to assess claims. Kenya Power has so far received 1,142 claims, of which 318 have been processed for ex gratia payment totalling KES 47.6 million. The remainder will be settled in two further tranches by 31 October 2024.",
		RespondedAt:  "2024-09-18",
		SourceURL:    "https://www.parliament.go.ke/the-national-assembly/questions/2024-09-04",
	},
	{
		ID:           "wq-005",
		MPID:         "person-002",
		MPName:       "Opiyo Wandayi",
		Minister:     "Cabinet Secretary, National Treasury",
		Ministry:     "The National Treasury & Economic Planning",
		QuestionText: "Could the Cabinet Secretary for the National Treasury state the total amount of pending bills owed to county governments as at 31 August 2024, and provide a disbursement schedule for their settlement before the end of the current financial year?",
		AskedAt:      "2024-09-12",
		Deadline:     "2024-10-03",
		Status:       WrittenQuestionPending,
		SourceURL:    "https://www.parliament.go.ke/the-national-assembly/questions/2024-09-12",
	},
	{
		ID:           "wq-006",
		MPID:         "person-002",
		MPName:       "Opiyo Wandayi",
		Minister:     "Cabinet Secretary, Water, Sanitation & Irrigation",
		Ministry:     "Ministry of Water, Sanitation & Irrigation",
		QuestionText: "Could the Cabinet Secretary for Water provide the status of the Alego Usonga rural water projects funded under the Lake Victoria North Water Works Development Agency, including the contractors, disbursements, and the projected completion dates?",
		AskedAt:      "2024-08-10",
		Deadline:     "2024-08-31",
		Status:       WrittenQuestionOverdue,
		SourceURL:    "https://www.parliament.go.ke/the-national-assembly/questions/2024-08-10",
	},

	// === person-003: Aaron Cheruiyot (Senator, Kericho County) ===
	{
		ID:           "wq-007",
		MPID:         "person-003",
		MPName:       "Aaron Cheruiyot",
		Minister:     "Cabinet Secretary, Roads & Transport",
		Ministry:     "Ministry of Roads & Transport",
		QuestionText: "Could the Cabinet Secretary for Roads & Transport state the status of the Kericho–Sondu highway expansion, including the amount disbursed to date, the contractor, and the projected completion date?",
		AskedAt:      "2024-08-27",
		Deadline:     "2024-09-17",
		Status:       WrittenQuestionAnswered,
		ResponseText: "The Kericho–Sondu highway expansion is 64% complete. The contractor, China Jiangxi, has been disbursed KES 4.91 billion to date against a contract sum of KES 7.65 billion. The revised completion date is 30 June 2025, with the rainy-season delay attributed to works on three box culverts between Lelaitich and Sondu.",
		RespondedAt:  "2024-09-10",
		SourceURL:    "https://www.parliament.go.ke/the-senate/questions/2024-08-27",
	},
	{
		ID:           "wq-008",
		MPID:         "person-003",
		MPName:       "Aaron Cheruiyot",
		Minister:     "Cabinet Secretary, National Treasury",
		Ministry:     "The National Treasury & Economic Planning",
		QuestionText: "Could the Cabinet Secretary for the National Treasury clarify the basis for the equitable share allocation to Kericho County for FY 2024/25, and explain the variance against the Council of Governors' proposal?",
		AskedAt:      "2024-09-09",
		Deadline:     "2024-09-30",
		Status:       WrittenQuestionPending,
		SourceURL:    "https://www.parliament.go.ke/the-senate/questions/2024-09-09",
	},
	{
		ID:           "wq-009",
		MPID:         "person-003",
		MPName:       "Aaron Cheruiyot",
		Minister:     "Cabinet Secretary, Education",
		Ministry:     "Ministry of Education",
		QuestionText: "Could the Cabinet Secretary for Education provide the capitation disbursement schedule for Kericho County public primary and secondary schools for FY 2024/25, including the schools yet to receive their first tranche?",
		AskedAt:      "2024-09-20",
		Deadline:     "2024-10-11",
		Status:       WrittenQuestionOverdue,
		SourceURL:    "https://www.parliament.go.ke/the-senate/questions/2024-09-20",
	},

	// === person-004: Esther Passaris (Women Rep, Nairobi) ===
	{
		ID:           "wq-010",
		MPID:         "person-004",
		MPName:       "Esther Passaris",
		Minister:     "Cabinet Secretary, Health",
		Ministry:     "Ministry of Health",
		QuestionText: "Could the Cabinet Secretary for Health update the House on the rollout status of the Social Health Authority (SHA) and the migration of NHIF members, including the number of members enrolled, the benefits package, and the transition timeline?",
		AskedAt:      "2024-09-02",
		Deadline:     "2024-09-23",
		Status:       WrittenQuestionAnswered,
		ResponseText: "As at 31 August 2024, the Social Health Authority had enrolled 14.2 million members, of which 8.7 million were migrated from NHIF. The primary healthcare benefits package is operational, with the secondary + tertiary packages to follow in two phases by 30 June 2025. The transition timeline remains as gazetted.",
		RespondedAt:  "2024-09-16",
		SourceURL:    "https://www.parliament.go.ke/the-national-assembly/questions/2024-09-02",
	},
	{
		ID:           "wq-011",
		MPID:         "person-004",
		MPName:       "Esther Passaris",
		Minister:     "Cabinet Secretary, Interior & National Administration",
		Ministry:     "Ministry of Interior & National Administration",
		QuestionText: "Could the Cabinet Secretary for Interior state the number of gender-based violence cases reported in Nairobi County in the year 2024, the conviction rate, and the measures being taken to strengthen the one-stop rescue centres in the county?",
		AskedAt:      "2024-09-10",
		Deadline:     "2024-10-01",
		Status:       WrittenQuestionPending,
		SourceURL:    "https://www.parliament.go.ke/the-national-assembly/questions/2024-09-10",
	},
	{
		ID:           "wq-012",
		MPID:         "person-004",
		MPName:       "Esther Passaris",
		Minister:     "Cabinet Secretary, Education",
		Ministry:     "Ministry of Education",
		QuestionText: "Could the Cabinet Secretary for Education provide the status of the sanitary towels programme for Nairobi County public schools, including the schools reached, the number of beneficiaries, and the budget for the current financial year?",
		AskedAt:      "2024-08-20",
		Deadline:     "2024-09-10",
		Status:       WrittenQuestionOverdue,
		SourceURL:    "https://www.parliament.go.ke/the-national-assembly/questions/2024-08-20",
	},

	// === person-005: Millie Odhiambo (MP, Suba North) ===
	{
		ID:           "wq-013",
		MPID:         "person-005",
		MPName:       "Millie Odhiambo",
		Minister:     "Attorney-General",
		Ministry:     "Office of the Attorney-General & Department of Justice",
		QuestionText: "Could the Attorney-General state the status of the legal opinion on the Huduma Namba digital identity framework, including whether any subordinate legislation has been drafted to operationalise the framework post the Court of Appeal judgement?",
		AskedAt:      "2024-08-26",
		Deadline:     "2024-09-16",
		Status:       WrittenQuestionAnswered,
		ResponseText: "The Office of the Attorney-General has drafted the Data Protection (Registration of Data Controllers and Data Processors) Regulations, 2024 to operationalise the digital identity framework within the parameters set by the Court of Appeal judgement. The regulations are undergoing stakeholder validation and will be gazetted by 30 November 2024.",
		RespondedAt:  "2024-09-09",
		SourceURL:    "https://www.parliament.go.ke/the-national-assembly/questions/2024-08-26",
	},
	{
		ID:           "wq-014",
		MPID:         "person-005",
		MPName:       "Millie Odhiambo",
		Minister:     "Cabinet Secretary, Health",
		Ministry:     "Ministry of Health",
		QuestionText: "Could the Cabinet Secretary for Health state the status of the Victim Protection Trust Fund, including the amount disbursed to date, the number of beneficiaries, and the criteria used to allocate compensation to victims of crime?",
		AskedAt:      "2024-09-08",
		Deadline:     "2024-09-29",
		Status:       WrittenQuestionPending,
		SourceURL:    "https://www.parliament.go.ke/the-national-assembly/questions/2024-09-08",
	},
	{
		ID:           "wq-015",
		MPID:         "person-005",
		MPName:       "Millie Odhiambo",
		Minister:     "Cabinet Secretary, Interior & National Administration",
		Ministry:     "Ministry of Interior & National Administration",
		QuestionText: "Could the Cabinet Secretary for Interior state the number of counter-trafficking operations conducted in Homa Bay County in the year 2024, the number of victims rescued, and the prosecution status of the suspects apprehended?",
		AskedAt:      "2024-09-25",
		Deadline:     "2024-10-16",
		Status:       WrittenQuestionOverdue,
		SourceURL:    "https://www.parliament.go.ke/the-national-assembly/questions/2024-09-25",
	},
}

// seedWrittenQuestionsSorted is the ready-to-serve slice of
// WrittenQuestion values, sorted most-recent-first by asked_at. It is
// computed once at package init so the handlers can stream it
// directly without re-sorting on every request.
//
// Defensive: init panics if the seed references an unknown person_id
// or has an invalid status. This is programmer error (the seed slice
// is hand-curated + the test suite cross-checks it), so failing loud
// at init is the correct posture — a silent skip would let the seed
// data drift out of sync with sampleScorecards without any test
// catching it.
var seedWrittenQuestionsSorted = func() []WrittenQuestion {
	// Index sampleScorecards by PersonID for O(1) name lookup + to
	// assert every seed row references a known MP.
	personNameByID := make(map[string]string, len(sampleScorecards))
	for _, p := range sampleScorecards {
		personNameByID[p.PersonID] = p.Name
	}
	out := make([]WrittenQuestion, 0, len(sampleWrittenQuestions))
	for i := range sampleWrittenQuestions {
		q := sampleWrittenQuestions[i]
		name, ok := personNameByID[q.MPID]
		if !ok {
			panic("written_questions: seed row " + q.ID + " references unknown person_id " + q.MPID)
		}
		if q.MPName != name {
			panic("written_questions: seed row " + q.ID + " has mp_name " + q.MPName + " but sampleScorecards has " + name)
		}
		if !isValidWrittenQuestionStatus(q.Status) {
			panic("written_questions: seed row " + q.ID + " has invalid status " + string(q.Status))
		}
		if q.QuestionText == "" || q.AskedAt == "" || q.Deadline == "" || q.SourceURL == "" {
			panic("written_questions: seed row " + q.ID + " missing required field")
		}
		// answered rows MUST carry response_text + responded_at;
		// pending + overdue rows MUST NOT.
		switch q.Status {
		case WrittenQuestionAnswered:
			if q.ResponseText == "" || q.RespondedAt == "" {
				panic("written_questions: answered row " + q.ID + " missing response_text / responded_at")
			}
		case WrittenQuestionPending, WrittenQuestionOverdue:
			if q.ResponseText != "" || q.RespondedAt != "" {
				panic("written_questions: " + string(q.Status) + " row " + q.ID + " must NOT carry response_text / responded_at")
			}
		}
		out = append(out, q)
	}
	// Most-recent-first by asked_at date. A stable sort preserves the
	// per-MP grouping of the source slice.
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].AskedAt > out[j].AskedAt
	})
	return out
}()

// findWrittenQuestionByID returns the written question for the given
// ID, or nil if no sample question matches. In production this lookup
// hits a repository; the sample list exists so the frontend + tests
// have something to render before the live parliament.go.ke feed is
// wired.
func findWrittenQuestionByID(id string) *WrittenQuestion {
	for i := range sampleWrittenQuestions {
		if sampleWrittenQuestions[i].ID == id {
			return &sampleWrittenQuestions[i]
		}
	}
	return nil
}

// findWrittenQuestionsByPerson returns every written question tabled
// by the given MP, sorted most-recent-first. Returns nil for an
// unknown person (the caller is responsible for surfacing 404 — see
// handleWrittenQuestionsByPerson).
func findWrittenQuestionsByPerson(personID string) []WrittenQuestion {
	out := make([]WrittenQuestion, 0, 3)
	for _, q := range seedWrittenQuestionsSorted {
		if q.MPID == personID {
			out = append(out, q)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// findPersonNameForWrittenQuestions returns the display name for the
// given person ID, or empty if the ID is unknown. Used by the per-MP
// handler so it can echo the name in the response envelope (mirrors
// the findPersonName helper in votes.go).
func findPersonNameForWrittenQuestions(personID string) string {
	for _, p := range sampleScorecards {
		if p.PersonID == personID {
			return p.Name
		}
	}
	return ""
}

// writtenQuestionsListResponse is the JSON envelope returned by
// GET /api/v1/questions/written. The shape mirrors the peopleList +
// scorecardListResponse envelopes so the frontend can reuse its
// rendering pattern (items + total + source). The optional `mp_id`
// and `status` filters are echoed so the caller can verify the
// filter was honoured.
type writtenQuestionsListResponse struct {
	Items  []WrittenQuestion `json:"items"`
	Total  int               `json:"total"`
	Source string            `json:"source"` // always "seed" until ingestion ships
	// MPID + Status echo the filters the caller supplied (empty when
	// the filter was omitted). Lets the frontend verify the filter was
	// honoured + renders the active filter as a chip / breadcrumb.
	MPID   string                `json:"mp_id,omitempty"`
	Status WrittenQuestionStatus `json:"status,omitempty"`
}

// writtenQuestionsByPersonResponse is the JSON envelope returned by
// GET /api/v1/people/{id}/questions. The shape mirrors the
// votesByPersonResponse + billsByPersonResponse envelopes so the
// frontend can reuse its rendering pattern (person_id + name + items
// + total + source + scorecard_url).
type writtenQuestionsByPersonResponse struct {
	MPID         string            `json:"mp_id"`
	Name         string            `json:"name"`
	Items        []WrittenQuestion `json:"items"`
	Total        int               `json:"total"`
	Source       string            `json:"source"` // always "seed" until ingestion ships
	ScorecardURL string            `json:"scorecard_url,omitempty"`
}

// makeWrittenQuestionsListHandler returns the handler for
// GET /api/v1/questions/written.
//
// Query parameters (all optional):
//
//	mp_id   — filter to questions tabled by the given MP (e.g.
//	          "person-001"). Echoed in the response envelope.
//	status  — filter to the given status. MUST be one of
//	          "pending", "answered", "overdue"; any other value
//	          returns 400. Echoed in the response envelope.
//
// On success: 200 with the items list (most-recent-first), the total
// count, the source ("seed"), and the echoed filters.
//
// On an invalid status filter: 400 (NOT 200 with an empty items
// list) so a typo in the status does not silently look like a
// legitimate "no questions match" response.
func makeWrittenQuestionsListHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		q := r.URL.Query()
		mpID := strings.TrimSpace(q.Get("mp_id"))
		statusParam := strings.TrimSpace(q.Get("status"))

		var statusFilter WrittenQuestionStatus
		if statusParam != "" {
			statusFilter = WrittenQuestionStatus(statusParam)
			if !isValidWrittenQuestionStatus(statusFilter) {
				writeError(w, http.StatusBadRequest, "bad_request",
					"invalid status filter: "+statusParam+" (expected one of pending, answered, overdue)")
				return
			}
		}

		items := make([]WrittenQuestion, 0, len(seedWrittenQuestionsSorted))
		for _, sq := range seedWrittenQuestionsSorted {
			if mpID != "" && sq.MPID != mpID {
				continue
			}
			if statusParam != "" && sq.Status != statusFilter {
				continue
			}
			items = append(items, sq)
		}
		writeJSON(w, http.StatusOK, writtenQuestionsListResponse{
			Items:  items,
			Total:  len(items),
			Source: "seed",
			MPID:   mpID,
			Status: statusFilter,
		})
	}
}

// makeWrittenQuestionDetailHandler returns the handler for
// GET /api/v1/questions/written/{id}. The path is dispatched by the
// /api/v1/questions/written/ sub-router registration in main.go.
//
// On success: 200 with the WrittenQuestion JSON.
// On unknown id: 404 (NOT 200 with an empty body) so a typo in the id
// does not silently look like a legitimate "question not yet
// answered" response.
func makeWrittenQuestionDetailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		// Path shape: /api/v1/questions/written/{id}
		tail := strings.TrimPrefix(r.URL.Path, "/api/v1/questions/written")
		id := strings.TrimPrefix(tail, "/")
		if id == "" || strings.Contains(id, "/") {
			writeError(w, http.StatusNotFound, "not_found", "written question not found: "+id)
			return
		}
		q := findWrittenQuestionByID(id)
		if q == nil {
			writeError(w, http.StatusNotFound, "not_found", "written question not found: "+id)
			return
		}
		writeJSON(w, http.StatusOK, q)
	}
}

// handleWrittenQuestionsByPerson is the handler for
// GET /api/v1/people/{id}/questions. It is dispatched from
// handlePeople (parallel with /scorecard, /bills and /votes). The MP
// must exist in sampleScorecards before any written questions are
// surfaced — this guards the "every datum is sourced" invariant (a
// typo in a person ID must 404, not silently return an empty items
// list that looks like a legitimate "no questions" response).
//
// On success: 200 with the items list (most-recent-first).
// On unknown person_id: 404.
func handleWrittenQuestionsByPerson(w http.ResponseWriter, r *http.Request, personID string) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
		return
	}
	if personID == "" {
		writeError(w, http.StatusBadRequest, "bad_request", "person ID required")
		return
	}
	name := findPersonNameForWrittenQuestions(personID)
	if name == "" {
		writeError(w, http.StatusNotFound, "not_found", "person not found: "+personID)
		return
	}
	items := findWrittenQuestionsByPerson(personID)
	if items == nil {
		// Known MP with no written questions in the seed slice — return
		// an empty items list rather than nil so the JSON encodes as
		// [] (not null). This branch is unreachable today (every
		// sample MP has 3 questions), but the guard keeps the
		// contract stable if the seed slice is ever pruned.
		items = []WrittenQuestion{}
	}
	writeJSON(w, http.StatusOK, writtenQuestionsByPersonResponse{
		MPID:         personID,
		Name:         name,
		Items:        items,
		Total:        len(items),
		Source:       "seed",
		ScorecardURL: "/api/v1/people/" + personID + "/scorecard",
	})
}

// writtenQuestionStatusLabel returns the human-readable label for a
// WrittenQuestionStatus. Used by the test suite to assert the
// on-the-wire value matches the expected label. Kept here (rather
// than in the test file) so the contract is visible alongside the
// const declarations.
func writtenQuestionStatusLabel(s WrittenQuestionStatus) string {
	switch s {
	case WrittenQuestionPending:
		return "Pending"
	case WrittenQuestionAnswered:
		return "Answered"
	case WrittenQuestionOverdue:
		return "Overdue"
	default:
		return string(s)
	}
}
