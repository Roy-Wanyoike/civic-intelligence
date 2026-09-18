// Package main — countries.go exposes the /api/v1/countries endpoint, which
// returns metadata for every country adapter registered with the central
// registry (adapters/registry). The frontend Government Selector uses
// this endpoint to render its country dropdown and the X-Civic-Country
// header is validated against the same registry on every other API call.
//
// Adding a new country? Register it in adapters/registry (see
// CONTRIBUTING.md "Adding a New Country") and it shows up here
// automatically — no change to this file.
package main

import (
	"encoding/json"
	"net/http"
	"sort"

	"github.com/Roy-Wanyoike/civic-intelligence/adapters/registry"
)

// countriesResponse is the JSON shape returned by GET /api/v1/countries.
// The `countries` slice is sorted by ISO 3166-1 alpha-2 code so the output
// is stable across runs (and across concurrent registrations). The
// `count` field mirrors the slice length for client convenience.
type countriesResponse struct {
	Countries      []countryInfo `json:"countries"`
	Count          int           `json:"count"`
	SourceRegistry string        `json:"source_registry"`
}

// countryInfo is the JSON shape of a single country row. Field names match
// the TypeScript `SupportedCountry` interface in
// apps/web/src/lib/government-defaults.ts so the frontend can deserialise
// the response directly into that type.
type countryInfo struct {
	Code            string `json:"code"`
	Name            string `json:"name"`
	FlagEmoji       string `json:"flag_emoji"`
	ParliamentName  string `json:"parliament_name"`
	LegislatureType string `json:"legislature_type"`
}

// handleCountriesList returns the list of supported countries. Public +
// unauthenticated — the country list is metadata, not authenticated user
// data, so the frontend can fetch it eagerly on page load.
func handleCountriesList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.Header().Set("Allow", "GET")
		http.Error(w, `{"error":"method_not_allowed"}`, http.StatusMethodNotAllowed)
		return
	}
	// SupportedCountries returns the slice already sorted by code.
	infos := registry.SupportedCountries()
	out := make([]countryInfo, 0, len(infos))
	for _, info := range infos {
		out = append(out, countryInfo{
			Code:            info.Code,
			Name:            info.Name,
			FlagEmoji:       info.FlagEmoji,
			ParliamentName:  info.ParliamentName,
			LegislatureType: info.LegislatureType,
		})
	}
	// Defensive sort: registry already sorts, but sorting here too keeps
	// this handler independent of future registry refactors.
	sort.Slice(out, func(i, j int) bool { return out[i].Code < out[j].Code })

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=300")
	_ = json.NewEncoder(w).Encode(countriesResponse{
		Countries:      out,
		Count:          len(out),
		SourceRegistry: "adapters/registry",
	})
}

// isSupportedCountryCode reports whether the registry recognises the
// given country code. Used by other handlers to validate the
// X-Civic-Country header before dispatching to the appropriate adapter.
//
// Exported at package level so other handlers in cmd/ can call it
// (e.g., the bills + acts handlers that need to switch adapters based on
// the country header).
func isSupportedCountryCode(code string) bool {
	return registry.IsSupported(code)
}
