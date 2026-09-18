package domain

import (
	"testing"
	"time"
)

// TestPresidentialAssentStatus_Values verifies the assent status enum.
func TestPresidentialAssentStatus_Values(t *testing.T) {
	cases := []struct {
		v   PresidentialAssentStatus
		exp string
	}{
		{AssentStatusAssented, "ASSENTED"},
		{AssentStatusReturned, "RETURNED"},
		{AssentStatusWithheld, "WITHHELD"},
		{AssentStatusUnknown, "UNKNOWN"},
	}
	for _, c := range cases {
		if string(c.v) != c.exp {
			t.Errorf("expected %s; got %s", c.exp, c.v)
		}
	}
}

// TestActAudit_DetectsMissingCommencement verifies that an Act without a
// commencement date is flagged as NOT_VERIFIED — NOT as "never commenced".
// Spec section 37: "No commencement notice found" does NOT mean "The Act
// never commenced." Instead use COMMENCEMENT_STATUS: NOT VERIFIED.
func TestActAudit_DetectsMissingCommencement(t *testing.T) {
	now := time.Now()
	pubDate := now.Add(-30 * 24 * time.Hour)
	assentDate := now.Add(-60 * 24 * time.Hour)
	act := Act{
		ID:              "act-1",
		ActNumber:       "Act No. 13 of 2024",
		AssentedAt:      assentDate,
		PublicationDate: &pubDate,
		CommencementDate: nil, // not set
		Status:          ActStatusPublished,
	}

	audit := AuditForAct(act, nil)

	if audit.AssentRecorded != true {
		t.Error("expected assent recorded")
	}
	if audit.ActPublished != true {
		t.Error("expected act published")
	}
	if audit.CommencementNotice != false {
		t.Error("expected commencement notice to be missing")
	}
	if !contains(audit.AuditStatuses, ActAuditNotVerified) {
		t.Error("expected ActAuditNotVerified in statuses")
	}
	if !contains(audit.AuditStatuses, ActAuditPartiallyTracked) {
		t.Error("expected ActAuditPartiallyTracked in statuses")
	}
	if !contains(audit.AuditStatuses, ActAuditDataGap) {
		t.Error("expected ActAuditDataGap in statuses")
	}
	if !containsString(audit.DataGaps, "commencement_notice_not_found") {
		t.Error("expected commencement_notice_not_found data gap")
	}
}

// TestActAudit_MarksCompleteWhenAllStepsPresent verifies the happy path.
func TestActAudit_MarksCompleteWhenAllStepsPresent(t *testing.T) {
	now := time.Now()
	pubDate := now.Add(-30 * 24 * time.Hour)
	commDate := now.Add(-15 * 24 * time.Hour)
	assentDate := now.Add(-60 * 24 * time.Hour)
	act := Act{
		ID:               "act-1",
		ActNumber:        "Act No. 13 of 2024",
		AssentedAt:       assentDate,
		PublicationDate:  &pubDate,
		CommencementDate: &commDate,
		Status:           ActStatusCommenced,
	}

	audit := AuditForAct(act, nil)

	if !contains(audit.AuditStatuses, ActAuditAssentConfirmed) {
		t.Error("expected ActAuditAssentConfirmed")
	}
	if !contains(audit.AuditStatuses, ActAuditPublicationConfirmed) {
		t.Error("expected ActAuditPublicationConfirmed")
	}
	if !contains(audit.AuditStatuses, ActAuditCommencementConfirmed) {
		t.Error("expected ActAuditCommencementConfirmed")
	}
	if !contains(audit.AuditStatuses, ActAuditComplete) {
		t.Error("expected ActAuditComplete")
	}
	if len(audit.DataGaps) != 0 {
		t.Errorf("expected no data gaps; got %v", audit.DataGaps)
	}
}

// TestActAudit_TracksPostAssentEvents verifies that events (regulations,
// court challenges, amendments, repeal) are tracked.
func TestActAudit_TracksPostAssentEvents(t *testing.T) {
	now := time.Now()
	pubDate := now.Add(-30 * 24 * time.Hour)
	commDate := now.Add(-15 * 24 * time.Hour)
	assentDate := now.Add(-60 * 24 * time.Hour)
	act := Act{
		ID:               "act-1",
		AssentedAt:       assentDate,
		PublicationDate:  &pubDate,
		CommencementDate: &commDate,
		Status:           ActStatusAmended,
	}
	events := []PostAssentEvent{
		{ID: "ev-1", ActID: "act-1", EventType: PostAssentEventRegulation, EventDate: now.Add(-10 * 24 * time.Hour), Title: "Regulation issued"},
		{ID: "ev-2", ActID: "act-1", EventType: PostAssentEventCourtChallenge, EventDate: now.Add(-5 * 24 * time.Hour), Title: "Court challenge filed"},
		{ID: "ev-3", ActID: "act-1", EventType: PostAssentEventJudicialDecision, EventDate: now.Add(-3 * 24 * time.Hour), Title: "High Court decision"},
		{ID: "ev-4", ActID: "act-1", EventType: PostAssentEventAmendment, EventDate: now.Add(-2 * 24 * time.Hour), Title: "Amendment Act"},
	}

	audit := AuditForAct(act, events)

	if !audit.RegulationsIssued {
		t.Error("expected regulations issued")
	}
	if !audit.CourtChallenged {
		t.Error("expected court challenged")
	}
	if !audit.JudicialDecision {
		t.Error("expected judicial decision")
	}
	if !audit.Amended {
		t.Error("expected amended")
	}
	if !contains(audit.AuditStatuses, ActAuditRegulationsTracked) {
		t.Error("expected ActAuditRegulationsTracked")
	}
	if !contains(audit.AuditStatuses, ActAuditJudicialHistoryTracked) {
		t.Error("expected ActAuditJudicialHistoryTracked")
	}
	if !contains(audit.AuditStatuses, ActAuditAmendmentsTracked) {
		t.Error("expected ActAuditAmendmentsTracked")
	}
}

// TestActAudit_DetectsRepeal verifies the repeal path.
func TestActAudit_DetectsRepeal(t *testing.T) {
	now := time.Now()
	pubDate := now.Add(-365 * 24 * time.Hour)
	commDate := now.Add(-350 * 24 * time.Hour)
	assentDate := now.Add(-400 * 24 * time.Hour)
	act := Act{
		ID:               "act-1",
		AssentedAt:       assentDate,
		PublicationDate:  &pubDate,
		CommencementDate: &commDate,
		Status:           ActStatusRepealed,
	}
	events := []PostAssentEvent{
		{ID: "ev-1", ActID: "act-1", EventType: PostAssentEventRepeal, EventDate: now.Add(-10 * 24 * time.Hour), Title: "Repeal Act"},
	}

	audit := AuditForAct(act, events)

	if !audit.Repealed {
		t.Error("expected repealed")
	}
	if !contains(audit.AuditStatuses, ActAuditRepealStatusTracked) {
		t.Error("expected ActAuditRepealStatusTracked")
	}
}

func contains(s []ActAuditStatus, v ActAuditStatus) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func containsString(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
