// Package legislation is the canonical civic-domain service. This file
// adds the Wire() constructors (issues #202, #203) that build in-memory
// ActRepository + DebtRepository instances and seed them with
// country-supplied data.
//
// Architectural note: the legislation service is country-agnostic. Country-
// specific seed data (e.g. Kenya Acts, Kenya debt snapshots) lives in
// adapters/kenya/kenya_seed. Because adapters/kenya already depends on
// services/legislation, the legislation service CANNOT import
// adapters/kenya back (that would be a circular module dependency). Wire()
// therefore accepts the seed data as parameters — the caller (typically the
// API service in main.go) sources the data from the country adapter and
// passes it in.
//
// This mirrors services/simulation/wire.go in spirit: a single entry point
// per bounded context that constructs the in-memory repository, seeds it,
// and returns it.
//
// Architectural invariants (Spec section 35, 37):
//   - The platform NEVER attributes sovereign borrowing personally to a
//     president. BorrowingAgreement.GovernmentAdministrationID is a
//     temporal linkage, not a personal blame axis.
//   - Summaries carry NO_POLITICAL_PERFORMANCE_SCORE — no "best borrower",
//     no debt performance ranking, no political score.
//   - Snapshots are immutable observations (Spec section 9). Changes in
//     debt stock can reflect FX, valuation, repayments, refinancing,
//     arrears, adjustments, disbursement timing — NOT just new borrowing.
package legislation

import (
        "context"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation/government"
        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/domain"
        "github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/infrastructure/memory"
)

// Re-export the Act-domain types consumers need. Using type aliases (not
// type definitions) so a legislation.Act IS a domain.Act — no conversion
// needed at the call site.
type (
        ID                      = domain.ID
        Act                     = domain.Act
        ActVersion              = domain.ActVersion
        ActStatus               = domain.ActStatus
        ActFilter               = domain.ActFilter
        ActRepository           = domain.ActRepository
        PostAssentEvent         = domain.PostAssentEvent
        PostAssentEventType     = domain.PostAssentEventType
        PresidentialAssentEvent = domain.PresidentialAssentEvent

        // Debt-domain type aliases (issue #203).
        FiscalYear               = domain.FiscalYear
        BorrowingAgreement       = domain.BorrowingAgreement
        DebtDisbursement         = domain.DebtDisbursement
        DebtRepayment            = domain.DebtRepayment
        DebtService              = domain.DebtService
        PublicDebtSnapshot       = domain.PublicDebtSnapshot
        BorrowingAuthorization   = domain.BorrowingAuthorization
        DebtEvidenceRef          = domain.DebtEvidenceRef
        FiscalReconciliationConflict = domain.FiscalReconciliationConflict
        GovernmentDebtSummary    = domain.GovernmentDebtSummary
        DebtRepository           = domain.DebtRepository
        DebtFilter               = domain.DebtFilter
        CreditorCategory         = domain.CreditorCategory
        DomesticOrExternal       = domain.DomesticOrExternal
        BorrowingAttribution     = domain.BorrowingAttribution
        DebtPurpose              = domain.DebtPurpose
        DebtTrendPoint           = domain.DebtTrendPoint

        // Administration type aliases used by ValidateAttribution.
        Administration = government.Administration
)

// NO_POLITICAL_PERFORMANCE_SCORE is re-exported from the domain so callers
// do not need to import internal/domain.
const NO_POLITICAL_PERFORMANCE_SCORE = domain.NO_POLITICAL_PERFORMANCE_SCORE

// SeedAct is a country-supplied seed record for an Act. The caller (typically
// the API service) sources these from the country adapter and passes them to
// Wire().
type SeedAct struct {
        ID               ID
        CountryID        ID
        BillID           ID
        ActNumber        string
        ActName         string
        GazetteRef       string
        CommencementDate *time.Time
        AssentedAt       time.Time
        SourceDocumentID string

        PublicationDate        *time.Time
        CommencementNoticeID   *ID
        Status                 ActStatus
        SourceURL              string
        Description            string
}

// SeedPostAssentEvent is a country-supplied seed record for a post-assent event.
type SeedPostAssentEvent struct {
        ID          ID
        ActID       ID
        BillID      *ID
        EventType   PostAssentEventType
        EventDate   time.Time
        Title       string
        Description string
        SourceURL   string
}

// mapSeedActToDomain converts a SeedAct to a domain.Act.
func mapSeedActToDomain(s SeedAct) domain.Act {
        return domain.Act{
                ID:                    domain.ID(s.ID),
                BillID:                domain.ID(s.BillID),
                CountryID:             domain.ID(s.CountryID),
                ActNumber:             s.ActNumber,
                ActName:               s.ActName,
                GazetteRef:            s.GazetteRef,
                CommencementDate:      s.CommencementDate,
                AssentedAt:            s.AssentedAt,
                SourceDocumentID:      s.SourceDocumentID,
                PublicationDate:       s.PublicationDate,
                CommencementNoticeID:  s.CommencementNoticeID,
                Status:                domain.ActStatus(s.Status),
                SourceURL:             s.SourceURL,
        }
}

// mapSeedEventToDomain converts a SeedPostAssentEvent to a domain.PostAssentEvent.
func mapSeedEventToDomain(s SeedPostAssentEvent) domain.PostAssentEvent {
        return domain.PostAssentEvent{
                ID:          domain.ID(s.ID),
                ActID:       domain.ID(s.ActID),
                BillID:      s.BillID,
                EventType:   domain.PostAssentEventType(s.EventType),
                EventDate:   s.EventDate,
                Title:       s.Title,
                Description: s.Description,
                SourceURL:   s.SourceURL,
        }
}

// Wire constructs an in-memory ActRepository and seeds it with the supplied
// acts and post-assent events. Callers (typically the API service in main.go)
// source the seed data from the country adapter and map it into the
// legislation.Act / legislation.PostAssentEvent types before calling Wire.
// Mirrors services/simulation/wire.go.
func Wire(seedActs []Act, seedEvents []PostAssentEvent) ActRepository {
        repo := memory.NewActRepo()
        ctx := context.Background()
        for _, s := range seedActs {
                _ = repo.CreateAct(ctx, domain.Act(s))
        }
        for _, s := range seedEvents {
                _ = repo.AppendPostAssentEvent(ctx, domain.PostAssentEvent(s))
        }
        return repo
}

// DebtSeeder is a callback that populates a DebtRepository with country-specific
// data. The legislation service is country-agnostic and cannot import the
// country adapter (would be a circular dependency), so the caller hands in the
// seeder rather than the data. The seeder may return an error if seeding fails.
type DebtSeeder func(repo DebtRepository) error

// WireDebtRepository constructs an in-memory DebtRepository and seeds it via
// the supplied seeder. Returns the repository and any seeder error so callers
// can decide whether to fail or log+continue. The repository is returned
// populated with whatever data was successfully seeded before the error.
func WireDebtRepository(seeder DebtSeeder) (DebtRepository, error) {
        repo := memory.NewDebtRepository()
        if seeder == nil {
                return repo, nil
        }
        return repo, seeder(repo)
}

// ValidateAttribution is re-exported from the domain. It verifies that
// BorrowingAgreement.GovernmentAdministrationID actually corresponds to the
// administration in power on ContractDate.
func ValidateAttribution(agreement BorrowingAgreement, administrations []Administration) error {
        return domain.ValidateAttribution(domain.BorrowingAgreement(agreement), administrations)
}
