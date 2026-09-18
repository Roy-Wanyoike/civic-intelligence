// Package kenya_seed — DTO→domain mappers for Kenya seed data.
//
// These helpers convert the plain DTOs in acts.go into the legislation
// service's domain types. The kenya_seed package can import
// services/legislation (the public root) because adapters/kenya already
// depends on services/legislation (for the government package). This
// avoids any circular module dependency.
package kenya_seed

import (
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation"
)

// MapActs converts a slice of SeedAct DTOs into the legislation service's
// Act type. The `now` parameter is used as a fallback CreatedAt for any
// seed act that doesn't supply one.
func MapActs(seeds []SeedAct, now time.Time) []legislation.Act {
	out := make([]legislation.Act, 0, len(seeds))
	for _, sa := range seeds {
		out = append(out, MapAct(sa, now))
	}
	return out
}

// MapAct converts a single SeedAct DTO into the legislation service's Act
// type. The seed DTO stores Status as the API response string (e.g.
// "in_force", "amended") — these are cast directly to ActStatus so the
// original API contract is preserved end-to-end.
func MapAct(sa SeedAct, now time.Time) legislation.Act {
	createdAt := sa.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	updatedAt := sa.UpdatedAt
	if updatedAt.IsZero() {
		updatedAt = createdAt
	}
	return legislation.Act{
		ID:                legislation.ID(sa.ID),
		BillID:            legislation.ID(sa.BillID),
		CountryID:         legislation.ID(sa.CountryID),
		ActNumber:         sa.ActNumber,
		ActName:           sa.ActName,
		GazetteRef:        sa.GazetteRef,
		CommencementDate:  sa.CommencementDate,
		AssentedAt:        sa.AssentedAt,
		SourceDocumentID:  sa.SourceDocumentID,
		Description:       sa.Description,
		PublicationDate:   sa.PublicationDate,
		Status:            legislation.ActStatus(sa.Status),
		SourceURL:         sa.SourceURL,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}
}

// MapPostAssentEvents converts a slice of SeedPostAssentEvent DTOs into the
// legislation service's PostAssentEvent type.
func MapPostAssentEvents(seeds []SeedPostAssentEvent, now time.Time) []legislation.PostAssentEvent {
	out := make([]legislation.PostAssentEvent, 0, len(seeds))
	for _, se := range seeds {
		out = append(out, MapPostAssentEvent(se, now))
	}
	return out
}

// MapPostAssentEvent converts a single SeedPostAssentEvent DTO into the
// legislation service's PostAssentEvent type.
func MapPostAssentEvent(se SeedPostAssentEvent, now time.Time) legislation.PostAssentEvent {
	createdAt := se.CreatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	ev := legislation.PostAssentEvent{
		ID:          legislation.ID(se.ID),
		ActID:       legislation.ID(se.ActID),
		EventType:   legislation.PostAssentEventType(se.EventType),
		EventDate:   se.EventDate,
		Title:       se.Title,
		Description: se.Description,
		SourceURL:   se.SourceURL,
		CreatedAt:   createdAt,
	}
	if se.BillID != nil {
		bid := legislation.ID(*se.BillID)
		ev.BillID = &bid
	}
	return ev
}
