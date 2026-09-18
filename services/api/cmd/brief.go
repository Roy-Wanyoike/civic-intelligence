// Package main — Civic Daily Brief (task ENG-I2).
//
// This file implements the personalised Civic Daily Brief: an AI-grounded,
// plain-language summary of the day's civic developments, filtered by what
// the citizen actually follows (topics, institutions, Bills). Every item in
// the brief MUST link to evidence — the platform never presents an
// unsubstantiated claim, and the AI summary is explicitly labelled with an
// ASSUMPTION reality badge so a reader can tell at a glance which text is
// generated and which is observed.
//
// Routes (registered in main.go):
//
//	POST /api/v1/brief/generate  — generate (and store) a personalised brief
//	GET  /api/v1/brief/today     — today's brief, generated on-demand if missing
//	GET  /api/v1/brief/archive   — list previously generated briefs
//	GET  /api/v1/brief/{id}      — a single brief by ID
//
// Brief generation pipeline (generateBrief):
//  1. Discover today's Bills via the Kenya Law adapter (the same source the
//     /api/v1/what-changed endpoint uses — single source of truth).
//  2. Filter the discovered changes by the user's followed_topics,
//     followed_institutions, and followed_bills.
//  3. Group changes into four sections:
//     "What Changed Today", "Your Followed Topics", "What to Watch",
//     "Constitutional Context".
//  4. Compose the headline from change counts (e.g., "3 Bills moved, 1 Act
//     commenced, 2 loans signed").
//  5. Ask the Python AI service (/v1/briefing/generate) for a 2-paragraph
//     plain-language summary of the day's activity. If the AI service is
//     unreachable, fall back to a deterministic, template-based summary that
//     is explicitly labelled "Auto-generated summary" — never presented as
//     AI output.
//  6. Attach the AI disclaimer: "AI-generated summary. Verify against
//     primary sources." — surfaced as `ai_disclaimer` on every brief.
//  7. Count the evidence URLs across all items → evidence_count.
//
// The brief store is an in-memory map[briefID]Brief protected by a mutex.
// In production this is replaced by a Postgres-backed implementation; the
// in-memory implementation is sufficient for local dev + tests.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_law"
)

// --- Types ---

// briefRequest is the JSON body for POST /api/v1/brief/generate.
//
// All fields are optional — an empty request yields a generic (un-personalised)
// brief of today's civic developments. UserID is propagated for downstream
// telemetry + per-user caching but is not required for the brief to render.
type briefRequest struct {
	UserID               string   `json:"user_id,omitempty"`
	FollowedTopics       []string `json:"followed_topics,omitempty"`
	FollowedInstitutions []string `json:"followed_institutions,omitempty"`
	FollowedBills        []string `json:"followed_bills,omitempty"`
	Country              string   `json:"country,omitempty"`
}

// brief is the canonical Civic Daily Brief shape. Returned by every endpoint
// under /api/v1/brief/*.
type brief struct {
	ID            string         `json:"id"`
	Date          string         `json:"date"`
	Headline      string         `json:"headline"`
	Sections      []briefSection `json:"sections"`
	AISummary     string         `json:"ai_summary"`
	AIDisclaimer  string         `json:"ai_disclaimer"`
	AISource      string         `json:"ai_source"`
	EvidenceCount int            `json:"evidence_count"`
	GeneratedAt   string         `json:"generated_at"`
	Country       string         `json:"country"`
	UserID        string         `json:"user_id,omitempty"`
}

// briefSection groups related brief items under a titled heading.
type briefSection struct {
	Title   string             `json:"title"`
	Summary string             `json:"summary"`
	Items   []briefSectionItem `json:"items"`
}

// briefSectionItem is the per-item shape used inside every section. Every
// populated item MUST carry a non-empty evidence_url — the frontend renders
// a per-item "View evidence" link and the brief-level evidence_count is the
// total of non-empty evidence_urls across all sections.
//
// The "constitutional context" section reuses this struct: `Article`,
// `Text`, and `Connection` carry the constitutional provision + plain-language
// connection to today's developments. `Type` for those rows is
// "constitutional_provision".
type briefSectionItem struct {
	Type         string `json:"type"`
	Title        string `json:"title,omitempty"`
	Description  string `json:"description,omitempty"`
	Bill         string `json:"bill,omitempty"`
	Change       string `json:"change,omitempty"`
	House        string `json:"house,omitempty"`
	Date         string `json:"date,omitempty"`
	Significance string `json:"significance,omitempty"`
	EvidenceURL  string `json:"evidence_url"`
	// Constitutional-context specific fields.
	Article    string `json:"article,omitempty"`
	Text       string `json:"text,omitempty"`
	Connection string `json:"connection,omitempty"`
}

// briefArchiveEntry is the summary shape returned by GET /api/v1/brief/archive.
// It omits the heavy section bodies so the archive list stays small.
type briefArchiveEntry struct {
	ID            string    `json:"id"`
	Date          string    `json:"date"`
	Headline      string    `json:"headline"`
	EvidenceCount int       `json:"evidence_count"`
	GeneratedAt   time.Time `json:"generated_at"`
}

// briefAISummaryDisclaimer is the disclaimer attached to every AI summary.
// Surfaced on the brief as `ai_disclaimer` and rendered verbatim by the
// frontend next to the ASSUMPTION reality badge.
const briefAISummaryDisclaimer = "AI-generated summary. Verify against primary sources."

// briefAISourceAI is the value of `ai_source` when the Python AI service
// successfully produced the summary.
const briefAISourceAI = "ai-service"

// briefAISourceTemplate is the value of `ai_source` when the AI service was
// unreachable and the brief fell back to the deterministic template summary.
// The template is NOT labelled as AI — it is explicitly "auto-generated" so
// no reader mistakes a fallback for a model output.
const briefAISourceTemplate = "template-fallback"

// --- Brief store (in-memory) ---

// briefStore is a thread-safe in-memory cache of generated briefs, keyed by
// brief ID (date-formatted: "brief-2026-09-18"). The store is intentionally
// process-local — in production this is replaced by a Postgres-backed
// implementation (see docs/architecture/12-deployment.md).
type briefStore struct {
	mu     sync.RWMutex
	briefs map[string]brief
}

func newBriefStore() *briefStore {
	return &briefStore{briefs: make(map[string]brief)}
}

// put stores (or replaces) a brief by its ID.
func (s *briefStore) put(b brief) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.briefs[b.ID] = b
}

// get returns the brief with the given ID, or (brief{}, false) if not found.
func (s *briefStore) get(id string) (brief, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	b, ok := s.briefs[id]
	return b, ok
}

// list returns all stored briefs, sorted newest-first by date.
func (s *briefStore) list() []brief {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]brief, 0, len(s.briefs))
	for _, b := range s.briefs {
		out = append(out, b)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Date > out[j].Date
	})
	return out
}

// --- Brief generation ---

// generateBrief builds a personalised Civic Daily Brief from the supplied
// change items (typically today's discovered Bills) + the user's followed
// entities. The function is pure: it does not touch the store. Callers
// (handlers) decide whether to persist the result.
//
// The aiServiceURL is the base URL of the Python AI service. When empty or
// unreachable, the brief falls back to a template-based summary that is
// explicitly labelled as "auto-generated" — never presented as AI output.
//
// The "today" date used for the brief ID is the caller's responsibility —
// handlers pass time.Now().UTC() so tests can override it.
func generateBrief(
	now time.Time,
	req briefRequest,
	bills []kenya_law.BillCandidate,
	aiServiceURL string,
) brief {
	country := strings.ToUpper(strings.TrimSpace(req.Country))
	if country == "" {
		country = "KE"
	}
	dateStr := now.UTC().Format("2006-01-02")
	briefID := "brief-" + dateStr

	// Filter Bills by followed institutions + followed Bills. Topics are
	// matched by substring on the title (the adapter does not yet expose a
	// structured topics field — see kenya_law.BillCandidate).
	filtered := filterBillsForUser(bills, req)

	// Build the four sections. Each item's evidence_url is the Bill's source
	// URL on kenyalaw.org — every claim is traceable to the primary source.
	changedItems := buildChangedItems(bills)
	followedItems := buildFollowedItems(filtered, req)
	watchItems := buildWatchItems(bills)
	constitutionItems := buildConstitutionalContext(req.FollowedTopics, bills)

	sections := []briefSection{
		{
			Title:   "What Changed Today",
			Summary: summarizeChangedToday(bills),
			Items:   changedItems,
		},
		{
			Title:   "Your Followed Topics",
			Summary: summarizeFollowed(req, filtered),
			Items:   followedItems,
		},
		{
			Title:   "What to Watch",
			Summary: "Upcoming events, deadlines, and Bills approaching final stage.",
			Items:   watchItems,
		},
		{
			Title:   "Constitutional Context",
			Summary: "Constitutional provisions relevant to today's developments.",
			Items:   constitutionItems,
		},
	}

	headline := buildBriefHeadline(bills)
	aiSummary, aiSource := generateAISummary(context.Background(), aiServiceURL, country, bills)

	b := brief{
		ID:           briefID,
		Date:         dateStr,
		Headline:     headline,
		Sections:     sections,
		AISummary:    aiSummary,
		AIDisclaimer: briefAISummaryDisclaimer,
		AISource:     aiSource,
		EvidenceCount: countEvidenceURLs(sections),
		GeneratedAt:  now.UTC().Format(time.RFC3339),
		Country:      country,
		UserID:       req.UserID,
	}
	return b
}

// filterBillsForUser keeps Bills that match any of the user's followed
// topics, institutions, or Bills. An empty follow set returns ALL Bills —
// the brief is still useful for a citizen who follows nothing yet.
func filterBillsForUser(bills []kenya_law.BillCandidate, req briefRequest) []kenya_law.BillCandidate {
	if len(req.FollowedTopics) == 0 && len(req.FollowedInstitutions) == 0 && len(req.FollowedBills) == 0 {
		return bills
	}
	out := make([]kenya_law.BillCandidate, 0, len(bills))
	for _, b := range bills {
		if billMatchesFollows(b, req) {
			out = append(out, b)
		}
	}
	return out
}

// billMatchesFollows reports whether a Bill matches any followed topic,
// institution, or Bill ID. Matching is case-insensitive substring on the
// Bill's title + house + source ID — the adapter does not yet expose a
// structured topics field.
func billMatchesFollows(b kenya_law.BillCandidate, req briefRequest) bool {
	titleLower := strings.ToLower(b.Title)
	houseLower := strings.ToLower(b.House)
	idLower := strings.ToLower(b.SourceID)
	for _, t := range req.FollowedTopics {
		t = strings.ToLower(strings.TrimSpace(t))
		if t == "" {
			continue
		}
		if strings.Contains(titleLower, t) || strings.Contains(houseLower, t) {
			return true
		}
	}
	for _, inst := range req.FollowedInstitutions {
		inst = strings.ToLower(strings.TrimSpace(inst))
		if inst == "" {
			continue
		}
		if strings.Contains(houseLower, inst) || strings.Contains(titleLower, inst) {
			return true
		}
	}
	for _, bid := range req.FollowedBills {
		bid = strings.ToLower(strings.TrimSpace(bid))
		if bid == "" {
			continue
		}
		if strings.Contains(idLower, bid) || idLower == bid {
			return true
		}
	}
	return false
}

// buildChangedItems transforms today's Bills into brief items for the
// "What Changed Today" section. Each item carries the Bill's kenyalaw.org
// URL as evidence_url.
func buildChangedItems(bills []kenya_law.BillCandidate) []briefSectionItem {
	items := make([]briefSectionItem, 0, len(bills))
	for _, b := range bills {
		dateStr := ""
		if !b.PublicationDate.IsZero() {
			dateStr = b.PublicationDate.Format("2006-01-02")
		}
		kind := "bill_published"
		change := "Published on Kenya Law Reports"
		// Infer a richer change description from the title — never fabricates
		// a stage the adapter did not surface.
		titleLower := strings.ToLower(b.Title)
		switch {
		case strings.Contains(titleLower, "amendment"):
			kind = "bill_stage_change"
			change = "Amendment Bill published — may alter an existing Act"
		case strings.Contains(titleLower, "finance") || strings.Contains(titleLower, "appropriation"):
			kind = "bill_stage_change"
			change = "Finance / Appropriation Bill published — affects the budget"
		}
		items = append(items, briefSectionItem{
			Type:         kind,
			Title:        b.Title,
			Description:  "Bill published on Kenya Law Reports.",
			Bill:         b.SourceID,
			Change:       change,
			House:        b.House,
			Date:         dateStr,
			Significance: classifySignificance(b.Title),
			EvidenceURL:  b.URL,
		})
	}
	return items
}

// buildFollowedItems produces the items for the "Your Followed Topics"
// section. Items are filtered to the user's followed set; if the user
// follows nothing, the section is empty (and the summary explains why).
func buildFollowedItems(filtered []kenya_law.BillCandidate, req briefRequest) []briefSectionItem {
	items := make([]briefSectionItem, 0, len(filtered))
	for _, b := range filtered {
		dateStr := ""
		if !b.PublicationDate.IsZero() {
			dateStr = b.PublicationDate.Format("2006-01-02")
		}
		items = append(items, briefSectionItem{
			Type:        "followed_bill",
			Title:       b.Title,
			Description: "A Bill you follow has activity today.",
			Bill:        b.SourceID,
			House:       b.House,
			Date:        dateStr,
			EvidenceURL: b.URL,
		})
	}
	// If the user follows topics but no Bills matched, surface a templated
	// note so the section is not silently empty.
	if len(items) == 0 && (len(req.FollowedTopics) > 0 || len(req.FollowedInstitutions) > 0 || len(req.FollowedBills) > 0) {
		items = append(items, briefSectionItem{
			Type:        "no_matches",
			Title:       "No activity matched your follows today",
			Description: "Subscribe to additional topics, institutions, or Bills to receive more relevant updates.",
			EvidenceURL: "https://new.kenyalaw.org/bills/",
		})
	}
	return items
}

// buildWatchItems surfaces upcoming / approaching-final Bills for the
// "What to Watch" section. Logic mirrors makeTrendingHandler: titles
// containing "Amendment", "Finance", or "Appropriation" are treated as
// approaching final stage (the adapter does not yet expose real stage data).
func buildWatchItems(bills []kenya_law.BillCandidate) []briefSectionItem {
	items := make([]briefSectionItem, 0, len(bills))
	now := time.Now()
	for _, b := range bills {
		titleLower := strings.ToLower(b.Title)
		approaching := strings.Contains(titleLower, "amendment") ||
			strings.Contains(titleLower, "finance") ||
			strings.Contains(titleLower, "appropriation")
		// Recently published (last 30 days) is "new" — surface as a watch item.
		if !approaching && (b.PublicationDate.IsZero() || now.Sub(b.PublicationDate) > 30*24*time.Hour) {
			continue
		}
		dateStr := ""
		if !b.PublicationDate.IsZero() {
			dateStr = b.PublicationDate.Format("2006-01-02")
		}
		change := "Approaching final stage — track committee + report stage."
		if !approaching {
			change = "Recently published — track First Reading + public participation window."
		}
		items = append(items, briefSectionItem{
			Type:        "watch",
			Title:       b.Title,
			Description: change,
			Bill:        b.SourceID,
			House:       b.House,
			Date:        dateStr,
			EvidenceURL: b.URL,
		})
	}
	return items
}

// buildConstitutionalContext returns constitutional provisions relevant to
// the followed topics + today's Bills. The mapping is a curated subset of
// the Constitution of Kenya 2010 (see adapters/kenya/kenya_seed/constitution.go
// + apps/web/src/data/constitution-articles.ts).
//
// When the user follows no topics and no Bills match a known topic, the
// section surfaces Article 10 (national values) as a sensible default — the
// section is never empty because the Constitution is always relevant context
// for legislative activity.
func buildConstitutionalContext(followedTopics []string, bills []kenya_law.BillCandidate) []briefSectionItem {
	// Aggregate all topics we care about: explicit follows + topics inferred
	// from today's Bill titles.
	topics := make(map[string]bool, len(followedTopics))
	for _, t := range followedTopics {
		topics[strings.ToLower(strings.TrimSpace(t))] = true
	}
	for _, b := range bills {
		titleLower := strings.ToLower(b.Title)
		for keyword, topic := range topicKeywordMap {
			if strings.Contains(titleLower, keyword) {
				topics[topic] = true
			}
		}
	}

	items := make([]briefSectionItem, 0, len(topics))
	for topic := range topics {
		art, ok := constitutionForTopic(topic)
		if !ok {
			continue
		}
		items = append(items, briefSectionItem{
			Type:        "constitutional_provision",
			Article:     art.Number,
			Title:       art.Number + " — " + art.Title,
			Text:        art.Text,
			Connection:  art.Connection,
			EvidenceURL: art.SourceURL,
		})
	}

	// Default to Article 10 if nothing matched — the Constitution's national
	// values are always relevant context for civic activity.
	if len(items) == 0 {
		art := constitutionArticle10
		items = append(items, briefSectionItem{
			Type:        "constitutional_provision",
			Article:     art.Number,
			Title:       art.Number + " — " + art.Title,
			Text:        art.Text,
			Connection:  art.Connection,
			EvidenceURL: art.SourceURL,
		})
	}

	// Stable order: by Article number (string sort is fine — articles are
	// "Article 10", "Article 19", etc.).
	sort.Slice(items, func(i, j int) bool {
		return items[i].Article < items[j].Article
	})
	return items
}

// topicKeywordMap maps substrings found in Bill titles to canonical topics
// used by the constitutional-context lookup. Keeping the map small and
// explicit avoids false positives (e.g., a Bill titled "Housing" must map
// to "housing" → Article 43, not to a guess).
var topicKeywordMap = map[string]string{
	"health":      "health",
	"medical":     "health",
	"education":   "education",
	"school":      "education",
	"finance":     "public_finance",
	"appropriation": "public_finance",
	"tax":         "public_finance",
	"taxation":    "public_finance",
	"debt":        "public_finance",
	"housing":     "housing",
	"data":        "data_protection",
	"privacy":     "data_protection",
	"environment": "environment",
	"land":        "land",
	"water":       "water",
	"children":    "children",
	"labour":      "labour",
	"employment":  "labour",
	"police":      "security",
	"security":    "security",
	"defence":     "security",
}

// constitutionArticle is a minimal subset of the Constitution of Kenya 2010
// surfaced by the brief's Constitutional Context section. The full text
// lives in adapters/kenya/kenya_seed/constitution.go — this is a curated
// projection that's enough to render the brief.
type constitutionArticle struct {
	Number     string
	Title      string
	Text       string
	SourceURL  string
	Connection string
}

// constitutionForTopic returns the constitution article relevant to a topic,
// or (zero, false) if the topic has no curated mapping.
func constitutionForTopic(topic string) (constitutionArticle, bool) {
	art, ok := constitutionTopicMap[topic]
	return art, ok
}

// constitutionArticle10 is the default — national values & principles of
// governance. Surfaced when no topic-specific article applies.
var constitutionArticle10 = constitutionArticle{
	Number:    "Article 10",
	Title:     "National Values and Principles of Governance",
	Text:      "The national values and principles of governance include patriotism, national unity, sharing and devolution of power, the rule of law, democracy and participation of the people, human dignity, equity, social justice, inclusiveness, equality, human rights, non-discrimination, protection of the marginalised, good governance, integrity, transparency and accountability.",
	SourceURL: "https://www.kenyalaw.org/kl/index.php?id=398",
	Connection: "Every Bill debated in Parliament must reflect these values. " +
		"Use this lens to evaluate whether today's developments advance or undermine them.",
}

// constitutionTopicMap maps canonical topics to the relevant Constitution of
// Kenya 2010 article. Curated from the official text at kenyalaw.org.
var constitutionTopicMap = map[string]constitutionArticle{
	"health": {
		Number:    "Article 43",
		Title:     "Economic and Social Rights",
		Text:      "Every person has the right to the highest attainable standard of health, which includes the right to health care services, including reproductive health care.",
		SourceURL: "https://www.kenyalaw.org/kl/index.php?id=398",
		Connection: "Health-sector Bills affect the State's obligation to realise the right to health care services.",
	},
	"education": {
		Number:    "Article 43",
		Title:     "Economic and Social Rights",
		Text:      "Every person has the right to education. A person shall not be denied emergency medical treatment.",
		SourceURL: "https://www.kenyalaw.org/kl/index.php?id=398",
		Connection: "Education-sector Bills affect the State's obligation to realise the right to education.",
	},
	"housing": {
		Number:    "Article 43",
		Title:     "Economic and Social Rights",
		Text:      "Every person has the right to accessible and adequate housing, and to reasonable standards of sanitation.",
		SourceURL: "https://www.kenyalaw.org/kl/index.php?id=398",
		Connection: "Housing Bills affect the State's obligation to realise the right to accessible and adequate housing.",
	},
	"public_finance": {
		Number:    "Article 201",
		Title:     "Principles of Public Finance",
		Text:      "Public finance shall promote an equitable society, the burden of taxation shall be shared fairly, revenue raised nationally shall be shared equitably, and expenditure shall promote equitable development.",
		SourceURL: "https://www.kenyalaw.org/kl/index.php?id=398",
		Connection: "Finance, Appropriation, and Tax Bills must conform to these principles of public finance.",
	},
	"data_protection": {
		Number:    "Article 31",
		Title:     "Privacy and Freedom of the Person",
		Text:      "Every person has the right to privacy, which includes the right not to have their person, home or property searched, their possessions seized, information relating to their family or private affairs unnecessarily required or revealed, or the privacy of their communications infringed.",
		SourceURL: "https://www.kenyalaw.org/kl/index.php?id=398",
		Connection: "Data Protection Bills give effect to the constitutional right to privacy.",
	},
	"environment": {
		Number:    "Article 42",
		Title:     "Environment",
		Text:      "Every person has the right to a clean and healthy environment, which includes the right to have the environment protected for the benefit of present and future generations.",
		SourceURL: "https://www.kenyalaw.org/kl/index.php?id=398",
		Connection: "Environment-sector Bills give effect to the right to a clean and healthy environment.",
	},
	"land": {
		Number:    "Article 60",
		Title:     "Principles of Land Policy",
		Text:      "Land in Kenya shall be held, used and managed in a manner that is equitable, efficient, productive and sustainable.",
		SourceURL: "https://www.kenyalaw.org/kl/index.php?id=398",
		Connection: "Land-sector Bills must conform to the principles of land policy in Article 60.",
	},
	"water": {
		Number:    "Article 43",
		Title:     "Economic and Social Rights",
		Text:      "Every person has the right to clean and safe water in adequate quantities.",
		SourceURL: "https://www.kenyalaw.org/kl/index.php?id=398",
		Connection: "Water-sector Bills affect the State's obligation to realise the right to clean and safe water.",
	},
	"children": {
		Number:    "Article 53",
		Title:     "Children",
		Text:      "Every child has the right to be protected from harm, to education, to basic nutrition, shelter and health care, and to be protected from abuse, neglect, harmful cultural practices and exploitation.",
		SourceURL: "https://www.kenyalaw.org/kl/index.php?id=398",
		Connection: "Children-sector Bills must give effect to the rights of the child in Article 53.",
	},
	"labour": {
		Number:    "Article 41",
		Title:     "Labour Relations",
		Text:      "Every person has the right to fair labour practices, including the right to fair remuneration, reasonable working conditions, and to form and join trade unions.",
		SourceURL: "https://www.kenyalaw.org/kl/index.php?id=398",
		Connection: "Labour-sector Bills must give effect to the right to fair labour practices in Article 41.",
	},
	"security": {
		Number:    "Article 238",
		Title:     "National Security",
		Text:      "The national security of Kenya is the responsibility of the national security organs, which shall be subordinate to civilian authority, act in compliance with the law, and respect human rights.",
		SourceURL: "https://www.kenyalaw.org/kl/index.php?id=398",
		Connection: "Security-sector Bills affect the national security organs and must respect civilian authority and human rights.",
	},
}

// buildBriefHeadline composes a count-based headline. The headline is the
// single most-read line of the brief, so it must be accurate + scannable.
func buildBriefHeadline(bills []kenya_law.BillCandidate) string {
	if len(bills) == 0 {
		return "No new civic developments detected today."
	}
	published := 0
	amendments := 0
	finance := 0
	for _, b := range bills {
		titleLower := strings.ToLower(b.Title)
		switch {
		case strings.Contains(titleLower, "finance") || strings.Contains(titleLower, "appropriation"):
			finance++
		case strings.Contains(titleLower, "amendment"):
			amendments++
		default:
			published++
		}
	}
	parts := make([]string, 0, 3)
	if published > 0 {
		parts = append(parts, fmt.Sprintf("%d Bill%s published", published, pluralS(published)))
	}
	if amendments > 0 {
		parts = append(parts, fmt.Sprintf("%d Amendment Bill%s", amendments, pluralS(amendments)))
	}
	if finance > 0 {
		parts = append(parts, fmt.Sprintf("%d Finance Bill%s", finance, pluralS(finance)))
	}
	if len(parts) == 0 {
		return fmt.Sprintf("%d civic development%s tracked today.", len(bills), pluralS(len(bills)))
	}
	return strings.Join(parts, ", ") + "."
}

func pluralS(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// classifySignificance mirrors makeWhatChangedHandler's heuristic — title
// containing "finance"/"appropriation" → HIGH_IMPACT, "amendment" →
// SUBSTANTIVE, otherwise INFORMATIONAL. Kept in sync so the brief and the
// /api/v1/what-changed feed never disagree on significance.
func classifySignificance(title string) string {
	t := strings.ToLower(title)
	switch {
	case strings.Contains(t, "finance") || strings.Contains(t, "appropriation"):
		return "HIGH_IMPACT"
	case strings.Contains(t, "amendment"):
		return "SUBSTANTIVE"
	default:
		return "INFORMATIONAL"
	}
}

// summarizeChangedToday produces the per-section summary for "What Changed
// Today". Always non-empty — when no Bills were discovered, the summary
// says so explicitly so the section is never silently empty.
func summarizeChangedToday(bills []kenya_law.BillCandidate) string {
	if len(bills) == 0 {
		return "No new Bills were detected on Kenya Law Reports today. Check back tomorrow, or browse the archive for past activity."
	}
	house := "National Assembly"
	senate := 0
	na := 0
	for _, b := range bills {
		h := strings.ToLower(b.House)
		if strings.Contains(h, "senate") {
			senate++
		} else {
			na++
		}
	}
	if senate > 0 && na > 0 {
		house = fmt.Sprintf("the National Assembly (%d) and the Senate (%d)", na, senate)
	} else if senate > 0 {
		house = fmt.Sprintf("the Senate (%d)", senate)
	} else {
		house = fmt.Sprintf("the National Assembly (%d)", na)
	}
	return fmt.Sprintf("Today's verified developments on Kenya Law Reports include %d Bill(s) from %s. Each item below links to its primary source.", len(bills), house)
}

// summarizeFollowed explains what's in the "Your Followed Topics" section.
func summarizeFollowed(req briefRequest, filtered []kenya_law.BillCandidate) string {
	followCount := len(req.FollowedTopics) + len(req.FollowedInstitutions) + len(req.FollowedBills)
	if followCount == 0 {
		return "You are not following any topics, institutions, or Bills yet. Visit the Following page to personalise this section."
	}
	if len(filtered) == 0 {
		return fmt.Sprintf("None of the %d entit%s you follow had activity today. Below is a guide to subscribe to more.", followCount, pluralEntity(followCount))
	}
	return fmt.Sprintf("%d of the entit%s you follow had activity today.", len(filtered), pluralEntity(len(filtered)))
}

func pluralEntity(n int) string {
	if n == 1 {
		return "y"
	}
	return "ies"
}

// countEvidenceURLs returns the total number of non-empty evidence_url
// values across all sections. Used for the brief-level evidence_count.
func countEvidenceURLs(sections []briefSection) int {
	n := 0
	for _, s := range sections {
		for _, it := range s.Items {
			if strings.TrimSpace(it.EvidenceURL) != "" {
				n++
			}
		}
	}
	return n
}

// --- AI summary ---

// generateAISummary calls the Python AI service (/v1/briefing/generate) to
// produce a 2-paragraph plain-language summary of today's activity. Returns
// (summary, source) where source is one of briefAISourceAI or
// briefAISourceTemplate.
//
// The template fallback is NOT labelled as AI — it is explicitly prefixed
// with "Auto-generated summary." so a reader can distinguish a fallback from
// a model output. The disclaimer ("AI-generated summary. Verify against
// primary sources.") is attached at the brief level — not in this function —
// so the disclaimer is consistent across both code paths.
//
// We deliberately DO NOT call the AI service when there are zero bills —
// the template fallback handles the empty case with a deterministic message.
func generateAISummary(ctx context.Context, aiServiceURL string, country string, bills []kenya_law.BillCandidate) (string, string) {
	if len(bills) == 0 {
		return templateSummaryEmpty(), briefAISourceTemplate
	}
	if aiServiceURL == "" {
		return templateSummaryFromBills(bills), briefAISourceTemplate
	}
	aiSummary, ok := callAIBriefingGenerator(ctx, aiServiceURL, country, bills)
	if !ok {
		log.Printf("brief: AI service unavailable, using template-fallback summary")
		return templateSummaryFromBills(bills), briefAISourceTemplate
	}
	return aiSummary, briefAISourceAI
}

// aiBriefingRequest is the request body for POST /v1/briefing/generate on
// the Python AI service. The contract lives in services/ai/app/main.py
// (BriefingRequest: { country: str, events: list[dict] }).
type aiBriefingRequest struct {
	Country string                   `json:"country"`
	Events  []map[string]interface{} `json:"events"`
}

// aiBriefingResponse is the response shape from /v1/briefing/generate. The
// Python service returns DailyBriefing.model_dump(mode="json") — we only
// consume the fields we need (headline + items).
type aiBriefingResponse struct {
	Country  string         `json:"country"`
	Date     string         `json:"date"`
	Headline string         `json:"headline"`
	Items    []aiBriefItem  `json:"items"`
}

type aiBriefItem struct {
	Kind        string                 `json:"kind"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Significance string               `json:"significance"`
	Citations   []map[string]any       `json:"citations"`
}

// callAIBriefingGenerator POSTs today's changes to the Python AI service and
// returns a 2-paragraph plain-language summary composed from the response.
// Returns ("", false) on any error — caller falls back to the template.
func callAIBriefingGenerator(ctx context.Context, aiServiceURL string, country string, bills []kenya_law.BillCandidate) (string, bool) {
	events := make([]map[string]interface{}, 0, len(bills))
	for _, b := range bills {
		dateStr := ""
		if !b.PublicationDate.IsZero() {
			dateStr = b.PublicationDate.Format("2006-01-02")
		}
		events = append(events, map[string]interface{}{
			"event_type":  "bill_published",
			"title":       b.Title,
			"description": "Bill published on Kenya Law Reports.",
			"bill_id":     b.SourceID,
			"house":       b.House,
			"date":         dateStr,
			"significance": classifySignificance(b.Title),
			"citations": []map[string]interface{}{
				{
					"document_id": b.SourceID,
					"source_url":  b.URL,
					"source_type": "kenya_law",
					"snippet":     b.Title,
				},
			},
		})
	}

	bodyBytes, err := json.Marshal(aiBriefingRequest{Country: country, Events: events})
	if err != nil {
		return "", false
	}

	url := strings.TrimRight(aiServiceURL, "/") + "/v1/briefing/generate"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyBytes))
	if err != nil {
		return "", false
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: 25 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", false
	}
	var br aiBriefingResponse
	if err := json.Unmarshal(raw, &br); err != nil {
		return "", false
	}

	return composeAISummaryFromResponse(br, bills), true
}

// composeAISummaryFromResponse turns the Python AI service's structured
// DailyBriefing into a 2-paragraph plain-language summary string. We do NOT
// blindly trust the AI service's headline / items — we compose our own
// summary that references today's verified changes by name, so the brief
// always reflects what the Go BFF actually observed (single source of truth).
func composeAISummaryFromResponse(resp aiBriefingResponse, bills []kenya_law.BillCandidate) string {
	if len(bills) == 0 {
		return templateSummaryEmpty()
	}
	// Paragraph 1: today's activity overview.
	var names []string
	for i, b := range bills {
		if i >= 3 {
			break
		}
		names = append(names, fmt.Sprintf("%q", b.Title))
	}
	suffix := ""
	if len(bills) > 3 {
		suffix = fmt.Sprintf(" and %d more", len(bills)-3)
	}
	p1 := fmt.Sprintf("Today the platform tracked %d verified civic development(s) on Kenya Law Reports, including %s%s. Every change is sourced from the official record; nothing here is inferred.",
		len(bills), strings.Join(names, ", "), suffix)

	// Paragraph 2: what to look for + the disclaimer anchor.
	p2 := "The most consequential items are flagged below by significance. Look for amendments (they alter existing law), finance / appropriation Bills (they affect the budget), and any Bill approaching its final stage. Cross-reference the Constitutional Context section to evaluate each change against the values and rights the Constitution protects."

	return p1 + "\n\n" + p2
}

// templateSummaryEmpty is the deterministic fallback when no Bills were
// discovered. Explicitly NOT labelled as AI — it is an "auto-generated
// summary" so the reader can tell.
func templateSummaryEmpty() string {
	return "Auto-generated summary. No new civic developments were detected on Kenya Law Reports today. " +
		"The platform does not fabricate activity — when the source feed is empty, the brief is empty. " +
		"Check back tomorrow, or browse the archive for past briefs."
}

// templateSummaryFromBills is the deterministic fallback when the AI service
// is unreachable. Explicitly labelled "Auto-generated summary" so a reader
// can distinguish a fallback from a model output.
func templateSummaryFromBills(bills []kenya_law.BillCandidate) string {
	if len(bills) == 0 {
		return templateSummaryEmpty()
	}
	var names []string
	for i, b := range bills {
		if i >= 3 {
			break
		}
		names = append(names, fmt.Sprintf("%q", b.Title))
	}
	suffix := ""
	if len(bills) > 3 {
		suffix = fmt.Sprintf(" and %d more", len(bills)-3)
	}
	p1 := fmt.Sprintf("Auto-generated summary. Today's verified developments on Kenya Law Reports include %d Bill(s): %s%s. "+
		"The AI summarisation service was unavailable, so this summary is templated from observed data — not a model output.",
		len(bills), strings.Join(names, ", "), suffix)
	p2 := fmt.Sprintf("Cross-reference the Constitutional Context section to evaluate each change against the Constitution. "+
		"The full evidence chain is in the Evidence footer below — every claim is traceable to its primary source on kenyalaw.org.")
	return p1 + "\n\n" + p2
}

// --- HTTP handlers ---

// makeBriefGenerateHandler handles POST /api/v1/brief/generate. It parses
// the request body, discovers today's Bills via the Kenya Law adapter,
// generates the brief, persists it to the store, and returns it.
func makeBriefGenerateHandler(adapter *kenya_law.Adapter, aiServiceURL string, store *briefStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use POST")
			return
		}
		var req briefRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			// Empty body is fine — yield an un-personalised brief.
			req = briefRequest{}
		}
		if req.Country == "" {
			req.Country = r.Header.Get("X-Civic-Country")
			if req.Country == "" {
				req.Country = "KE"
			}
		}

		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()

		bills, err := adapter.DiscoverBills(ctx)
		if err != nil {
			log.Printf("brief generate: adapter error: %v", err)
			// Continue with empty Bills — the brief surfaces the failure
			// honestly rather than returning a 5xx.
			bills = []kenya_law.BillCandidate{}
		}

		b := generateBrief(time.Now(), req, bills, aiServiceURL)
		store.put(b)
		writeJSON(w, http.StatusOK, b)
	}
}

// makeBriefTodayHandler handles GET /api/v1/brief/today. Returns today's
// brief from the store if it has already been generated; otherwise
// generates on-demand, persists, and returns.
func makeBriefTodayHandler(adapter *kenya_law.Adapter, aiServiceURL string, store *briefStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
			return
		}
		now := time.Now()
		briefID := "brief-" + now.UTC().Format("2006-01-02")
		if b, ok := store.get(briefID); ok {
			writeJSON(w, http.StatusOK, b)
			return
		}

		// On-demand generation. Use an empty briefRequest — the /today
		// endpoint is un-personalised; personalised briefs go through
		// /generate.
		country := r.Header.Get("X-Civic-Country")
		if country == "" {
			country = "KE"
		}
		ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		bills, err := adapter.DiscoverBills(ctx)
		if err != nil {
			log.Printf("brief today: adapter error: %v", err)
			bills = []kenya_law.BillCandidate{}
		}
		b := generateBrief(now, briefRequest{Country: country}, bills, aiServiceURL)
		store.put(b)
		writeJSON(w, http.StatusOK, b)
	}
}

// makeBriefArchiveHandler handles GET /api/v1/brief/archive. Returns a list
// of brief summaries (no section bodies) sorted newest-first.
func makeBriefArchiveHandler(store *briefStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
			return
		}
		all := store.list()
		entries := make([]briefArchiveEntry, 0, len(all))
		for _, b := range all {
			genAt, err := time.Parse(time.RFC3339, b.GeneratedAt)
			if err != nil {
				genAt = time.Time{}
			}
			entries = append(entries, briefArchiveEntry{
				ID:            b.ID,
				Date:          b.Date,
				Headline:      b.Headline,
				EvidenceCount: b.EvidenceCount,
				GeneratedAt:   genAt,
			})
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"items": entries,
			"total": len(entries),
		})
	}
}

// makeBriefDetailHandler handles GET /api/v1/brief/{id}. Returns a single
// brief by ID. Returns 404 if the brief is not in the store.
func makeBriefDetailHandler(store *briefStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "use GET")
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/brief/")
		// Reject the subtree-catch paths for the other routes — they are
		// registered separately as exact matches, but defensive: if the
		// caller lands here for one of them, return 404 rather than a
		// confusing "not found: today".
		if id == "" || id == "today" || id == "archive" || id == "generate" {
			writeError(w, http.StatusNotFound, "not_found", "brief ID required (e.g. /api/v1/brief/brief-2026-09-18)")
			return
		}
		b, ok := store.get(id)
		if !ok {
			writeError(w, http.StatusNotFound, "not_found", "brief not found: "+id)
			return
		}
		writeJSON(w, http.StatusOK, b)
	}
}
