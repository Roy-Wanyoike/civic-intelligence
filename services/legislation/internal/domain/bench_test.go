package domain_test

import (
        "testing"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation/government"
        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/domain"
)

// BenchmarkBillStateMachine_ValidTransition measures the cost of a single
// valid stage transition lookup through the canonical StageGraphValidator
// (wrapped by BillStateMachine). The hot path for bills ingestion is
// ValidateTransition — every published event runs through this check.
//
// Reported allocs/op must be 0; the validator is a map lookup, not an
// allocation. Any regression here is a per-event tax on the ingestion
// pipeline.
func BenchmarkBillStateMachine_ValidTransition(b *testing.B) {
        stages := benchStages()
        sm := domain.NewBillStateMachine(stages)
        b.ResetTimer()
        b.ReportAllocs()
        for i := 0; i < b.N; i++ {
                if err := sm.ValidateTransition("FIRST_READING", "COMMITTEE"); err != nil {
                        b.Fatalf("unexpected error: %v", err)
                }
        }
}

// BenchmarkAuditForAct_CompleteAct measures AuditForAct against a fully
// populated Act with one of every post-assent event kind (regulation,
// court challenge, judicial decision, amendment, repeal). This is the
// worst-case audit shape the /acts/{id}/audit endpoint will encounter
// in production — every AuditStatus flag is true and every DataGap is
// empty. The benchmark establishes the upper bound for the audit
// computation; per-iteration cost must stay sub-millisecond.
func BenchmarkAuditForAct_CompleteAct(b *testing.B) {
        now := time.Now().UTC()
        pub := now.Add(-30 * 24 * time.Hour)
        comm := now.Add(-15 * 24 * time.Hour)
        assent := now.Add(-60 * 24 * time.Hour)
        act := domain.Act{
                ID:               "act-bench",
                ActNumber:        "Act No. 1 of 2024",
                AssentedAt:       assent,
                PublicationDate:  &pub,
                CommencementDate: &comm,
                Status:           domain.ActStatusCommenced,
        }
        events := []domain.PostAssentEvent{
                {ID: "ev-1", ActID: act.ID, EventType: domain.PostAssentEventRegulation, EventDate: now.Add(-10 * 24 * time.Hour)},
                {ID: "ev-2", ActID: act.ID, EventType: domain.PostAssentEventCourtChallenge, EventDate: now.Add(-5 * 24 * time.Hour)},
                {ID: "ev-3", ActID: act.ID, EventType: domain.PostAssentEventJudicialDecision, EventDate: now.Add(-3 * 24 * time.Hour)},
                {ID: "ev-4", ActID: act.ID, EventType: domain.PostAssentEventAmendment, EventDate: now.Add(-2 * 24 * time.Hour)},
                {ID: "ev-5", ActID: act.ID, EventType: domain.PostAssentEventRepeal, EventDate: now.Add(-1 * 24 * time.Hour)},
        }
        b.ResetTimer()
        b.ReportAllocs()
        for i := 0; i < b.N; i++ {
                audit := domain.AuditForAct(act, events)
                if audit.ActID != act.ID {
                        b.Fatalf("audit returned wrong act id: %s", audit.ActID)
                }
        }
}

// BenchmarkValidateAttribution measures the cost of the attribution
// integrity check used by /debt/loans and the ingestion pipeline. The
// benchmark uses a realistic 5-administration slice (Kenya's three
// post-2010 administrations plus two historical) so the linear scan
// inside ValidateAttribution exercises a non-trivial N. Any regression
// here is a per-loan tax on the debt dashboard.
func BenchmarkValidateAttribution(b *testing.B) {
        term2End := time.Date(2022, 9, 13, 0, 0, 0, 0, time.UTC)
        rutoStart := time.Date(2022, 9, 13, 0, 0, 0, 0, time.UTC)
        uhuruStart := time.Date(2013, 4, 9, 0, 0, 0, 0, time.UTC)
        kibakiStart := time.Date(2002, 12, 30, 0, 0, 0, 0, time.UTC)
        moiEnd := time.Date(2002, 12, 30, 0, 0, 0, 0, time.UTC)
        moiStart := time.Date(1978, 8, 22, 0, 0, 0, 0, time.UTC)
        admins := []government.Administration{
                {ID: "admin-ruto", Name: "Ruto Admin", StartDate: rutoStart},
                {ID: "admin-uhuru-2", Name: "Uhuru Term 2", StartDate: uhuruStart, EndDate: &term2End},
                {ID: "admin-kibaki-2", Name: "Kibaki Term 2", StartDate: time.Date(2007, 12, 30, 0, 0, 0, 0, time.UTC), EndDate: &uhuruStart},
                {ID: "admin-kibaki-1", Name: "Kibaki Term 1", StartDate: kibakiStart, EndDate: &moiEnd},
                {ID: "admin-moi", Name: "Moi Admin", StartDate: moiStart, EndDate: &moiEnd},
        }
        contractDate := time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)
        agreement := domain.BorrowingAgreement{
                ID:                          "loan-bench",
                GovernmentAdministrationID: "admin-uhuru-2",
                ContractDate:                &contractDate,
        }
        b.ResetTimer()
        b.ReportAllocs()
        for i := 0; i < b.N; i++ {
                if err := domain.ValidateAttribution(agreement, admins); err != nil {
                        b.Fatalf("unexpected attribution error: %v", err)
                }
        }
}

// benchStages returns a stage set with the same shape as the test
// stages in bill_state_machine_test.go but with the realistic depth
// of the Kenyan lifecycle (8 non-terminal + 3 terminal). Used by the
// transition benchmark so the validator exercises a non-trivial map.
func benchStages() []contracts.StageDefinition {
        return []contracts.StageDefinition{
                {Code: "INTRODUCTION", Name: "Introduction", AllowedNext: []string{"FIRST_READING", "REJECTED", "WITHDRAWN"}, Country: "TE"},
                {Code: "FIRST_READING", Name: "First Reading", AllowedNext: []string{"COMMITTEE", "REJECTED", "WITHDRAWN"}, Country: "TE"},
                {Code: "COMMITTEE", Name: "Committee", AllowedNext: []string{"SECOND_READING", "REJECTED", "WITHDRAWN"}, Country: "TE"},
                {Code: "SECOND_READING", Name: "Second Reading", AllowedNext: []string{"COMMITTEE_OF_WHOLE", "REJECTED", "WITHDRAWN"}, Country: "TE"},
                {Code: "COMMITTEE_OF_WHOLE", Name: "Committee of Whole House", AllowedNext: []string{"THIRD_READING", "REJECTED", "WITHDRAWN"}, Country: "TE"},
                {Code: "THIRD_READING", Name: "Third Reading", AllowedNext: []string{"OTHER_HOUSE", "REJECTED", "WITHDRAWN"}, Country: "TE"},
                {Code: "OTHER_HOUSE", Name: "Other House", AllowedNext: []string{"CONCURRENCE", "REJECTED", "WITHDRAWN"}, Country: "TE"},
                {Code: "CONCURRENCE", Name: "Concurrence", AllowedNext: []string{"PRESIDENTIAL_ASSENT", "REJECTED", "WITHDRAWN"}, Country: "TE"},
                {Code: "PRESIDENTIAL_ASSENT", Name: "Presidential Assent", AllowedNext: []string{"COMMENCEMENT"}, Country: "TE"},
                {Code: "COMMENCEMENT", Name: "Commencement", AllowedNext: nil, IsTerminal: true, Country: "TE"},
                {Code: "REJECTED", Name: "Rejected", AllowedNext: nil, IsTerminal: true, Country: "TE"},
                {Code: "WITHDRAWN", Name: "Withdrawn", AllowedNext: nil, IsTerminal: true, Country: "TE"},
        }
}
