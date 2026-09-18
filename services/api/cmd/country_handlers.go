// Package main — country-scoping helpers (task ENG-J1).
//
// This file provides the per-handler helpers that consume the country
// code placed on the request context by middleware.Country and filter
// the in-memory sample datasets accordingly.
//
// Resolution:
//
//	country := middleware.CountryFromContext(r.Context())
//
// If country == middleware.GlobalCountry ("ALL"), the helper returns the
// input slice UNCHANGED — that is the "dashboard / cross-country" view
// (used by /compare, /indicators, /dashboard, /graph). Otherwise the
// helper returns only the rows whose `country` field matches.
//
// When the middleware is not chained (e.g. existing unit tests that call
// a handler directly), CountryFromContext returns the default "KE" — so
// the existing KE-only tests continue to see KE-only data.
package main

import (
	"github.com/Roy-Wanyoike/civic-intelligence/services/api/internal/middleware"
)

// filterActsByCountry returns the subset of acts whose Country field
// matches the supplied country code. If country is empty or
// middleware.GlobalCountry ("ALL"), the input slice is returned
// unchanged (the "dashboard" view).
func filterActsByCountry(items []actResponse, country string) []actResponse {
	if country == "" || country == middleware.GlobalCountry {
		return items
	}
	out := make([]actResponse, 0, len(items))
	for _, a := range items {
		if a.Country == country {
			out = append(out, a)
		}
	}
	return out
}

// filterMapsByCountry returns the subset of map[string]any rows whose
// "country" field matches the supplied country code. Used by the
// samplePeople / sampleCommittees / sampleInstitutions handlers, all of
// which store their country on a "country" string key. If country is
// empty or middleware.GlobalCountry, the input slice is returned
// unchanged.
func filterMapsByCountry(items []map[string]any, country string) []map[string]any {
	if country == "" || country == middleware.GlobalCountry {
		return items
	}
	out := make([]map[string]any, 0, len(items))
	for _, m := range items {
		if c, ok := m["country"].(string); ok && c == country {
			out = append(out, m)
		}
	}
	return out
}
