// Package main provides government API endpoints.
//
// Endpoints (registered in main.go):
//
//	GET /api/v1/governments       -- list administrations
//	GET /api/v1/governments/{id}  -- get an administration (also handles the
//	                                 /terms sub-resource internally; there is
//	                                 no separately registered /terms route)
//	GET /api/v1/constitution      -- get the constitution metadata
//	GET /api/v1/transitions       -- list government transitions
//
// The following endpoints are NOT registered, even though earlier versions of
// this file documented them:
//
//	GET /api/v1/governments/{id}/terms        -- handled internally by the
//	                                            governments/{id} detail handler
//	GET /api/v1/constitution/articles         -- not registered
//	GET /api/v1/constitution/articles/{id}    -- not registered
package main

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_seed"
	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/government"
)

// governmentData is the in-memory store seeded from kenya_seed.
var governmentData = struct {
	constitution government.Constitution
	presidents   []government.President
	admins       []government.Administration
	terms        []government.PresidentialTerm
	transitions  []government.GovernmentTransition
}{
	constitution: kenya_seed.ConstitutionOfKenya2010,
	presidents:   kenya_seed.KenyaPresidents,
	admins:       kenya_seed.KenyaAdministrations,
	terms:        kenya_seed.KenyaPresidentialTerms,
	transitions:  kenya_seed.KenyaGovernmentTransitions,
}

func makeGovernmentsListHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		admins := governmentData.admins
		// Sort by start date descending (most recent first).
		sort.Slice(admins, func(i, j int) bool {
			return admins[i].StartDate.After(admins[j].StartDate)
		})
		// Hydrate president names.
		out := make([]map[string]any, 0, len(admins))
		for _, a := range admins {
			presidentName := ""
			for _, p := range governmentData.presidents {
				if p.ID == a.PresidentID {
					presidentName = p.DisplayName
					break
				}
			}
			out = append(out, map[string]any{
				"id":               a.ID,
				"name":             a.Name,
				"president_id":     a.PresidentID,
				"president_name":   presidentName,
				"start_date":       a.StartDate,
				"end_date":         a.EndDate,
				"is_current":       a.EndDate == nil,
				"country_code":     a.CountryCode,
				"source_url":       a.SourceURL,
			})
		}
		writeJSON(w, 200, map[string]any{
			"administrations": out,
			"count":           len(out),
		})
	}
}

func makeGovernmentDetailHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/governments/")
		// Sub-resource: /terms
		if strings.HasSuffix(id, "/terms") {
			adminID := strings.TrimSuffix(id, "/terms")
			makeGovernmentTermsHandler(w, r, government.ID(adminID))
			return
		}
		var admin *government.Administration
		for i := range governmentData.admins {
			if governmentData.admins[i].ID == government.ID(id) {
				admin = &governmentData.admins[i]
				break
			}
		}
		if admin == nil {
			writeError(w, http.StatusNotFound, "not_found", "administration not found")
			return
		}
		var president *government.President
		for i := range governmentData.presidents {
			if governmentData.presidents[i].ID == admin.PresidentID {
				president = &governmentData.presidents[i]
				break
			}
		}
		// Collect terms for this administration.
		terms := []government.PresidentialTerm{}
		for _, t := range governmentData.terms {
			if t.AdministrationID == admin.ID {
				terms = append(terms, t)
			}
		}
		sort.Slice(terms, func(i, j int) bool {
			return terms[i].TermNumber < terms[j].TermNumber
		})
		writeJSON(w, 200, map[string]any{
			"administration": admin,
			"president":       president,
			"terms":           terms,
		})
	}
}

func makeGovernmentTermsHandler(w http.ResponseWriter, r *http.Request, adminID government.ID) {
	terms := []government.PresidentialTerm{}
	for _, t := range governmentData.terms {
		if t.AdministrationID == adminID {
			terms = append(terms, t)
		}
	}
	sort.Slice(terms, func(i, j int) bool {
		return terms[i].TermNumber < terms[j].TermNumber
	})
	writeJSON(w, 200, map[string]any{
		"administration_id": adminID,
		"terms":              terms,
		"count":              len(terms),
	})
}

func makeConstitutionHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		c := governmentData.constitution
		writeJSON(w, 200, map[string]any{
			"constitution":   c,
			"disclaimer":      "Constitution is authoritative source material. The platform does not reinterpret constitutional text.",
			"reality_layer":   "FACT",
		})
	}
}

func makeTransitionsHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeError(w, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		ts := governmentData.transitions
		sort.Slice(ts, func(i, j int) bool {
			return ts[i].TransitionDate.After(ts[j].TransitionDate)
		})
		// Hydrate president names.
		out := make([]map[string]any, 0, len(ts))
		for _, t := range ts {
			outPresident := ""
			inPresident := ""
			for _, p := range governmentData.presidents {
				if t.OutgoingPresidentID != nil && p.ID == *t.OutgoingPresidentID {
					outPresident = p.DisplayName
				}
				if p.ID == t.IncomingPresidentID {
					inPresident = p.DisplayName
				}
			}
			out = append(out, map[string]any{
				"id":                    t.ID,
				"transition_date":      t.TransitionDate,
				"outgoing_admin_id":    t.OutgoingAdminID,
				"incoming_admin_id":    t.IncomingAdminID,
				"outgoing_president":   outPresident,
				"incoming_president":   inPresident,
				"source_url":            t.SourceURL,
			})
		}
		writeJSON(w, 200, map[string]any{
			"transitions": out,
			"count":       len(out),
		})
	}
}

// jsonEncode is a local helper to avoid collisions with other files.
var _ = json.Marshal
