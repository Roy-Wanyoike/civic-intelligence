// Package memory — tests for the in-memory DebtRepository.
//
// These tests cover every domain.DebtRepository method plus the
// immutability invariants (Spec section 9):
//   - RecordDebtSnapshot rejects re-writes by ID AND by (country, observationDate).
//   - RecordGovernmentDebtSummary rejects re-writes by AdministrationID.
//   - AppendDisbursement / AppendRepayment reject orphans (unknown agreement ID).
//
// They also cover domain.ValidateAttribution (issue #221):
//   - happy path: agreement attributed to the administration in power on ContractDate.
//   - missing ContractDate → ErrAttributionMissingContractDate.
//   - unknown AdministrationID → ErrAttributionUnknownAdministration.
//   - wrong administration → ErrAttributionMismatch with helpful message.
package memory

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/government"
	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/domain"
)

// testAdmins is the fixture used by ValidateAttribution tests. Three
// administrations span the full date range so every test scenario has a
// matching "in power" administration.
func testAdmins() []government.Administration {
	return []government.Administration{
		{
			ID:        "admin-kibaki",
			Name:      "Kibaki Administration",
			StartDate: time.Date(2002, 12, 30, 0, 0, 0, 0, time.UTC),
			EndDate:   ptrTime(time.Date(2013, 4, 9, 0, 0, 0, 0, time.UTC)),
		},
		{
			ID:        "admin-uhuru",
			Name:      "Uhuru Kenyatta Administration",
			StartDate: time.Date(2013, 4, 9, 0, 0, 0, 0, time.UTC),
			EndDate:   ptrTime(time.Date(2022, 9, 13, 0, 0, 0, 0, time.UTC)),
		},
		{
			ID:        "admin-ruto",
			Name:      "William Ruto Administration",
			StartDate: time.Date(2022, 9, 13, 0, 0, 0, 0, time.UTC),
			EndDate:   nil, // current
		},
	}
}

func ptrTime(t time.Time) *time.Time { return &t }
func ptrFloat(f float64) *float64    { return &f }
func ptrID(s string) *domain.ID      { id := domain.ID(s); return &id }

// sampleAgreement constructs a BorrowingAgreement for tests.
func sampleAgreement(id string, adminID string, contractDate time.Time) domain.BorrowingAgreement {
	return domain.BorrowingAgreement{
		ID:                               domain.ID(id),
		CountryCode:                      "KE",
		GovernmentAdministrationID:       domain.ID(adminID),
		CreditorID:                       "cred-1",
		CreditorName:                     "World Bank",
		CreditorCategory:                 domain.CreditorMultilateral,
		InstrumentType:                   "External Loan",
		DomesticOrExternal:               domain.BorrowingExternal,
		OriginalAmount:                   1_000_000_000,
		OriginalCurrency:                 "USD",
		ConvertedAmountReportingCurrency: 150_000_000_000,
		Purpose:                          string(domain.PurposeInfrastructure),
		Sector:                           "Transport",
		ContractDate:                     &contractDate,
		Status:                           "CONTRACTED",
		CreatedAt:                        time.Now().UTC(),
		UpdatedAt:                        time.Now().UTC(),
	}
}

// === CreateBorrowingAgreement =================================================

func TestCreateBorrowingAgreement_Success(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	a := sampleAgreement("loan-1", "admin-uhuru", time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC))
	if err := repo.CreateBorrowingAgreement(ctx, a); err != nil {
		t.Fatalf("create: %v", err)
	}
	// Verify the agreement is retrievable.
	got, err := repo.GetBorrowingAgreement(ctx, "loan-1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != "loan-1" {
		t.Errorf("expected loan-1; got %s", got.ID)
	}
	if got.GovernmentAdministrationID != "admin-uhuru" {
		t.Errorf("expected admin-uhuru; got %s", got.GovernmentAdministrationID)
	}
}

func TestCreateBorrowingAgreement_DuplicateID_ReturnsErrAlreadyExists(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	a := sampleAgreement("loan-1", "admin-uhuru", time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC))
	if err := repo.CreateBorrowingAgreement(ctx, a); err != nil {
		t.Fatalf("create first: %v", err)
	}
	err := repo.CreateBorrowingAgreement(ctx, a)
	if !errors.Is(err, domain.ErrBorrowingAgreementAlreadyExists) {
		t.Errorf("expected ErrBorrowingAgreementAlreadyExists; got %v", err)
	}
}

// === GetBorrowingAgreement ====================================================

func TestGetBorrowingAgreement_NotFound(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	_, err := repo.GetBorrowingAgreement(ctx, "does-not-exist")
	if !errors.Is(err, domain.ErrBorrowingAgreementNotFound) {
		t.Errorf("expected ErrBorrowingAgreementNotFound; got %v", err)
	}
}

func TestGetBorrowingAgreement_ReturnsCopy(t *testing.T) {
	// The returned pointer must point to a copy, so mutations by the caller
	// do not corrupt the repository's internal state.
	repo := NewDebtRepository()
	ctx := context.Background()
	a := sampleAgreement("loan-1", "admin-uhuru", time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC))
	_ = repo.CreateBorrowingAgreement(ctx, a)

	got, _ := repo.GetBorrowingAgreement(ctx, "loan-1")
	got.OriginalAmount = 999 // mutate

	again, _ := repo.GetBorrowingAgreement(ctx, "loan-1")
	if again.OriginalAmount == 999 {
		t.Error("GetBorrowingAgreement returned a pointer to internal state; mutation leaked")
	}
}

// === ListBorrowingAgreements ==================================================

func TestListBorrowingAgreements_EmptyFilter_ReturnsAll(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	_ = repo.CreateBorrowingAgreement(ctx, sampleAgreement("loan-1", "admin-uhuru", time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)))
	_ = repo.CreateBorrowingAgreement(ctx, sampleAgreement("loan-2", "admin-ruto", time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)))

	got, err := repo.ListBorrowingAgreements(ctx, domain.DebtFilter{})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 agreements; got %d", len(got))
	}
}

func TestListBorrowingAgreements_FilterByAdministration(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	_ = repo.CreateBorrowingAgreement(ctx, sampleAgreement("loan-1", "admin-uhuru", time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)))
	_ = repo.CreateBorrowingAgreement(ctx, sampleAgreement("loan-2", "admin-ruto", time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)))

	admin := domain.ID("admin-uhuru")
	got, err := repo.ListBorrowingAgreements(ctx, domain.DebtFilter{GovernmentAdministrationID: &admin})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 agreement; got %d", len(got))
	}
	if got[0].ID != "loan-1" {
		t.Errorf("expected loan-1; got %s", got[0].ID)
	}
}

func TestListBorrowingAgreements_FilterByDateRange(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	_ = repo.CreateBorrowingAgreement(ctx, sampleAgreement("loan-1", "admin-uhuru", time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)))
	_ = repo.CreateBorrowingAgreement(ctx, sampleAgreement("loan-2", "admin-ruto", time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC)))

	from := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	got, _ := repo.ListBorrowingAgreements(ctx, domain.DebtFilter{From: &from, To: &to})
	if len(got) != 1 {
		t.Fatalf("expected 1 agreement in [2022-01-01, 2024-01-01); got %d", len(got))
	}
	if got[0].ID != "loan-2" {
		t.Errorf("expected loan-2; got %s", got[0].ID)
	}
}

func TestListBorrowingAgreements_FilterByCreditorCategory(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	a1 := sampleAgreement("loan-1", "admin-uhuru", time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC))
	a2 := sampleAgreement("loan-2", "admin-ruto", time.Date(2023, 1, 10, 0, 0, 0, 0, time.UTC))
	a2.CreditorCategory = domain.CreditorCommercial
	_ = repo.CreateBorrowingAgreement(ctx, a1)
	_ = repo.CreateBorrowingAgreement(ctx, a2)

	cat := domain.CreditorMultilateral
	got, _ := repo.ListBorrowingAgreements(ctx, domain.DebtFilter{CreditorCategory: &cat})
	if len(got) != 1 {
		t.Fatalf("expected 1 multilateral agreement; got %d", len(got))
	}
	if got[0].ID != "loan-1" {
		t.Errorf("expected loan-1; got %s", got[0].ID)
	}
}

func TestListBorrowingAgreements_Limit(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		_ = repo.CreateBorrowingAgreement(ctx, sampleAgreement(
			letters[i],
			"admin-uhuru",
			time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC),
		))
	}
	got, _ := repo.ListBorrowingAgreements(ctx, domain.DebtFilter{Limit: 3})
	if len(got) != 3 {
		t.Errorf("expected 3 agreements; got %d", len(got))
	}
}

var letters = []string{"a", "b", "c", "d", "e", "f", "g", "h"}

// === AppendDisbursement ======================================================

func TestAppendDisbursement_Success(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	_ = repo.CreateBorrowingAgreement(ctx, sampleAgreement("loan-1", "admin-uhuru", time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)))

	d := domain.DebtDisbursement{
		ID:                         "disb-1",
		BorrowingAgreementID:       "loan-1",
		Date:                       time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC),
		Amount:                     500_000_000,
		Currency:                   "USD",
		ExchangeRateAtDisbursement: 150,
	}
	if err := repo.AppendDisbursement(ctx, d); err != nil {
		t.Fatalf("append disbursement: %v", err)
	}
	disbs := repo.ListDisbursements("loan-1")
	if len(disbs) != 1 {
		t.Fatalf("expected 1 disbursement; got %d", len(disbs))
	}
	if disbs[0].ID != "disb-1" {
		t.Errorf("expected disb-1; got %s", disbs[0].ID)
	}
}

func TestAppendDisbursement_Orphan_ReturnsErrNotFound(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	d := domain.DebtDisbursement{
		ID:                   "disb-1",
		BorrowingAgreementID: "loan-does-not-exist",
		Date:                 time.Date(2020, 9, 1, 0, 0, 0, 0, time.UTC),
		Amount:               500_000_000,
	}
	err := repo.AppendDisbursement(ctx, d)
	if !errors.Is(err, domain.ErrBorrowingAgreementNotFound) {
		t.Errorf("expected ErrBorrowingAgreementNotFound; got %v", err)
	}
}

// === AppendRepayment =========================================================

func TestAppendRepayment_Success(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	_ = repo.CreateBorrowingAgreement(ctx, sampleAgreement("loan-1", "admin-uhuru", time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)))

	p := domain.DebtRepayment{
		ID:                   "repay-1",
		BorrowingAgreementID: "loan-1",
		Date:                 time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC),
		Amount:               50_000_000,
		PrincipalPortion:     45_000_000,
		InterestPortion:      5_000_000,
	}
	if err := repo.AppendRepayment(ctx, p); err != nil {
		t.Fatalf("append repayment: %v", err)
	}
	repays := repo.ListRepayments("loan-1")
	if len(repays) != 1 {
		t.Fatalf("expected 1 repayment; got %d", len(repays))
	}
}

func TestAppendRepayment_Orphan_ReturnsErrNotFound(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	p := domain.DebtRepayment{
		ID:                   "repay-1",
		BorrowingAgreementID: "loan-does-not-exist",
		Date:                 time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC),
		Amount:               50_000_000,
	}
	err := repo.AppendRepayment(ctx, p)
	if !errors.Is(err, domain.ErrBorrowingAgreementNotFound) {
		t.Errorf("expected ErrBorrowingAgreementNotFound; got %v", err)
	}
}

func TestAppendRepayment_PreservesCONTRACTED_DURINGAttribution(t *testing.T) {
	// Spec section 4: a loan contracted during Uhuru's term but repaid during
	// Ruto's term is still attributed to Uhuru by CONTRACTED_DURING. The
	// repayment event is a separate record that does NOT change the
	// agreement's attribution.
	repo := NewDebtRepository()
	ctx := context.Background()
	contractDate := time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)  // Uhuru's term
	repaymentDate := time.Date(2024, 3, 10, 0, 0, 0, 0, time.UTC) // Ruto's term
	a := sampleAgreement("loan-1", "admin-uhuru", contractDate)
	_ = repo.CreateBorrowingAgreement(ctx, a)
	_ = repo.AppendRepayment(ctx, domain.DebtRepayment{
		ID:                   "repay-1",
		BorrowingAgreementID: "loan-1",
		Date:                 repaymentDate,
		Amount:               50_000_000,
		PrincipalPortion:     45_000_000,
	})

	got, _ := repo.GetBorrowingAgreement(ctx, "loan-1")
	// CONTRACTED_DURING attribution is unchanged by the repayment.
	if got.GovernmentAdministrationID != "admin-uhuru" {
		t.Errorf("expected admin-uhuru; got %s", got.GovernmentAdministrationID)
	}
}

// === AppendDebtService =======================================================

func TestAppendDebtService_Success(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	s := domain.DebtService{
		ID:           "service-fy2023-24",
		CountryCode:  "KE",
		Period:       "FY 2023/24",
		Principal:    900_000_000_000,
		Interest:     460_000_000_000,
		TotalService: 1_360_000_000_000,
		Currency:     "KES",
	}
	if err := repo.AppendDebtService(ctx, s); err != nil {
		t.Fatalf("append debt service: %v", err)
	}
}

// === RecordDebtSnapshot (immutable) ==========================================

func TestRecordDebtSnapshot_Success(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	snap := domain.PublicDebtSnapshot{
		ID:              "snap-1",
		CountryCode:     "KE",
		ObservationDate: time.Date(2023, 6, 30, 0, 0, 0, 0, time.UTC),
		TotalDebtStock:  9.18e12,
		DomesticDebt:    3.96e12,
		ExternalDebt:    5.22e12,
		Currency:        "KES",
	}
	if err := repo.RecordDebtSnapshot(ctx, snap); err != nil {
		t.Fatalf("record snapshot: %v", err)
	}
}

func TestRecordDebtSnapshot_RejectsSameID(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	snap := domain.PublicDebtSnapshot{
		ID:              "snap-1",
		CountryCode:     "KE",
		ObservationDate: time.Date(2023, 6, 30, 0, 0, 0, 0, time.UTC),
		TotalDebtStock:  9.18e12,
	}
	_ = repo.RecordDebtSnapshot(ctx, snap)

	err := repo.RecordDebtSnapshot(ctx, snap)
	if !errors.Is(err, domain.ErrSnapshotImmutable) {
		t.Errorf("expected ErrSnapshotImmutable; got %v", err)
	}
}

func TestRecordDebtSnapshot_RejectsSameCountryAndDate(t *testing.T) {
	// Spec section 9: snapshots are immutable. Two snapshots for the same
	// country on the same observation date would silently overwrite each
	// other; the implementation rejects this even when the IDs differ.
	repo := NewDebtRepository()
	ctx := context.Background()
	snap1 := domain.PublicDebtSnapshot{
		ID:              "snap-1",
		CountryCode:     "KE",
		ObservationDate: time.Date(2023, 6, 30, 0, 0, 0, 0, time.UTC),
		TotalDebtStock:  9.18e12,
	}
	_ = repo.RecordDebtSnapshot(ctx, snap1)

	snap2 := snap1
	snap2.ID = "snap-2" // different ID, same country + observation date
	err := repo.RecordDebtSnapshot(ctx, snap2)
	if !errors.Is(err, domain.ErrSnapshotImmutable) {
		t.Errorf("expected ErrSnapshotImmutable for same country+date; got %v", err)
	}
}

func TestRecordDebtSnapshot_AllowsSameDateDifferentCountry(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	date := time.Date(2023, 6, 30, 0, 0, 0, 0, time.UTC)
	snap1 := domain.PublicDebtSnapshot{
		ID:              "snap-ke-1",
		CountryCode:     "KE",
		ObservationDate: date,
		TotalDebtStock:  9.18e12,
	}
	snap2 := domain.PublicDebtSnapshot{
		ID:              "snap-ug-1",
		CountryCode:     "UG",
		ObservationDate: date,
		TotalDebtStock:  20e12,
	}
	if err := repo.RecordDebtSnapshot(ctx, snap1); err != nil {
		t.Fatalf("snap1: %v", err)
	}
	if err := repo.RecordDebtSnapshot(ctx, snap2); err != nil {
		t.Fatalf("snap2: %v", err)
	}
}

// === ListDebtSnapshots =======================================================

func TestListDebtSnapshots_FiltersByCountryAndDateRange(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	for _, tc := range []struct {
		id  string
		cc  string
		yr  int
		val float64
	}{
		{"snap-ke-2021", "KE", 2021, 6.92e12},
		{"snap-ke-2022", "KE", 2022, 7.71e12},
		{"snap-ke-2023", "KE", 2023, 9.18e12},
		{"snap-ke-2024", "KE", 2024, 10.59e12},
		{"snap-ug-2023", "UG", 2023, 20e12},
	} {
		_ = repo.RecordDebtSnapshot(ctx, domain.PublicDebtSnapshot{
			ID:              domain.ID(tc.id),
			CountryCode:     tc.cc,
			ObservationDate: time.Date(tc.yr, 6, 30, 0, 0, 0, 0, time.UTC),
			TotalDebtStock:  tc.val,
		})
	}

	got, err := repo.ListDebtSnapshots(ctx, "KE",
		time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 KE snapshots in [2022, 2023]; got %d", len(got))
	}
	if got[0].ObservationDate.After(got[1].ObservationDate) {
		t.Error("snapshots not returned in chronological order")
	}
}

func TestListDebtSnapshots_EmptyResultForUnknownCountry(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	_ = repo.RecordDebtSnapshot(ctx, domain.PublicDebtSnapshot{
		ID:              "snap-ke-1",
		CountryCode:     "KE",
		ObservationDate: time.Date(2023, 6, 30, 0, 0, 0, 0, time.UTC),
	})
	got, _ := repo.ListDebtSnapshots(ctx, "ZZ",
		time.Date(2000, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2100, 1, 1, 0, 0, 0, 0, time.UTC))
	if len(got) != 0 {
		t.Errorf("expected 0 snapshots for unknown country; got %d", len(got))
	}
}

// === RecordGovernmentDebtSummary / GetGovernmentDebtSummary ==================

func TestRecordGovernmentDebtSummary_Success(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	s := domain.GovernmentDebtSummary{
		AdministrationID: "admin-uhuru",
		Period:           "2013-2022",
		DebtAtStart:      ptrFloat(1.96e12),
		DebtAtEnd:        ptrFloat(7.71e12),
		NewBorrowing:     ptrFloat(5.75e12),
		Currency:         "KES",
		Disclaimer:       "the Republic of Kenya is the borrower; this is not a personal score",
	}
	if err := repo.RecordGovernmentDebtSummary(ctx, s); err != nil {
		t.Fatalf("record summary: %v", err)
	}
}

func TestRecordGovernmentDebtSummary_RejectsDuplicate(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	s := domain.GovernmentDebtSummary{
		AdministrationID: "admin-uhuru",
		Period:           "2013-2022",
	}
	_ = repo.RecordGovernmentDebtSummary(ctx, s)
	err := repo.RecordGovernmentDebtSummary(ctx, s)
	if !errors.Is(err, domain.ErrGovernmentDebtSummaryAlreadyExists) {
		t.Errorf("expected ErrGovernmentDebtSummaryAlreadyExists; got %v", err)
	}
}

func TestGetGovernmentDebtSummary_NotFound(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	_, err := repo.GetGovernmentDebtSummary(ctx, "admin-does-not-exist")
	if !errors.Is(err, domain.ErrGovernmentDebtSummaryNotFound) {
		t.Errorf("expected ErrGovernmentDebtSummaryNotFound; got %v", err)
	}
}

// === AppendReconciliationConflict ===========================================

func TestAppendReconciliationConflict_Success(t *testing.T) {
	repo := NewDebtRepository()
	ctx := context.Background()
	c := domain.FiscalReconciliationConflict{
		ID:           "conflict-1",
		CountryCode:  "KE",
		Subject:      "Public debt stock on 2023-06-30",
		ConflictType: "AMOUNT_MISMATCH",
		SourceA: domain.DebtEvidenceRef{
			Kind:      "CBK_REPORT",
			ID:        "cbk-mei-2023-06",
			SourceURL: "https://www.centralbank.go.ke/uploads/monthly_economic_indicators/",
		},
		SourceB: domain.DebtEvidenceRef{
			Kind:      "TREASURY_REPORT",
			ID:        "apdr-2022-23",
			SourceURL: "https://www.treasury.go.ke/wp-content/uploads/2023/05/Annual-Public-Debt-Report-2022-23.pdf",
		},
		ValueA: "9.18T",
		ValueB: "9.21T",
		Status: "OPEN",
	}
	if err := repo.AppendReconciliationConflict(ctx, c); err != nil {
		t.Fatalf("append conflict: %v", err)
	}
	got := repo.ListReconciliationConflicts()
	if len(got) != 1 {
		t.Fatalf("expected 1 conflict; got %d", len(got))
	}
	if got[0].ID != "conflict-1" {
		t.Errorf("expected conflict-1; got %s", got[0].ID)
	}
}

// === ValidateAttribution (issue #221) ========================================

func TestValidateAttribution_HappyPath(t *testing.T) {
	admins := testAdmins()
	// Contract date during Uhuru's term; agreement correctly attributed to Uhuru.
	agreement := sampleAgreement("loan-1", "admin-uhuru", time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC))
	if err := domain.ValidateAttribution(agreement, admins); err != nil {
		t.Errorf("expected nil; got %v", err)
	}
}

func TestValidateAttribution_HappyPath_CurrentAdministration(t *testing.T) {
	// Ruto's administration has EndDate == nil (current). The validation
	// must accept any contract date >= 2022-09-13.
	admins := testAdmins()
	agreement := sampleAgreement("loan-1", "admin-ruto", time.Date(2024, 3, 1, 0, 0, 0, 0, time.UTC))
	if err := domain.ValidateAttribution(agreement, admins); err != nil {
		t.Errorf("expected nil; got %v", err)
	}
}

func TestValidateAttribution_MissingContractDate(t *testing.T) {
	admins := testAdmins()
	agreement := sampleAgreement("loan-1", "admin-uhuru", time.Time{})
	agreement.ContractDate = nil
	err := domain.ValidateAttribution(agreement, admins)
	if !errors.Is(err, domain.ErrAttributionMissingContractDate) {
		t.Errorf("expected ErrAttributionMissingContractDate; got %v", err)
	}
}

func TestValidateAttribution_UnknownAdministration(t *testing.T) {
	admins := testAdmins()
	agreement := sampleAgreement("loan-1", "admin-unknown", time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC))
	err := domain.ValidateAttribution(agreement, admins)
	if !errors.Is(err, domain.ErrAttributionUnknownAdministration) {
		t.Errorf("expected ErrAttributionUnknownAdministration; got %v", err)
	}
}

func TestValidateAttribution_Mismatch_AttributedToWrongAdmin(t *testing.T) {
	// Contract date is during Uhuru's term, but the agreement is incorrectly
	// attributed to Ruto. ValidateAttribution must reject this with a
	// helpful error naming the correct administration.
	admins := testAdmins()
	agreement := sampleAgreement("loan-1", "admin-ruto", time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC))
	err := domain.ValidateAttribution(agreement, admins)
	if !errors.Is(err, domain.ErrAttributionMismatch) {
		t.Errorf("expected ErrAttributionMismatch; got %v", err)
	}
	// Error message should name the correct administration.
	if err == nil {
		t.Fatal("expected non-nil error")
	}
	if !contains(err.Error(), "admin-uhuru") {
		t.Errorf("expected error to mention admin-uhuru (the correct admin); got: %v", err)
	}
}

func TestValidateAttribution_BoundaryExactStartDate(t *testing.T) {
	// Uhuru's StartDate is 2013-04-09. A contract date exactly on the start
	// date must be attributed to Uhuru (the window is [StartDate, EndDate)).
	admins := testAdmins()
	agreement := sampleAgreement("loan-1", "admin-uhuru", time.Date(2013, 4, 9, 0, 0, 0, 0, time.UTC))
	if err := domain.ValidateAttribution(agreement, admins); err != nil {
		t.Errorf("expected nil for contract date on admin start boundary; got %v", err)
	}
}

func TestValidateAttribution_BoundaryDayBeforeStart(t *testing.T) {
	// The day before Uhuru's start date must NOT be attributed to Uhuru.
	admins := testAdmins()
	agreement := sampleAgreement("loan-1", "admin-uhuru", time.Date(2013, 4, 8, 0, 0, 0, 0, time.UTC))
	err := domain.ValidateAttribution(agreement, admins)
	if !errors.Is(err, domain.ErrAttributionMismatch) {
		t.Errorf("expected ErrAttributionMismatch; got %v", err)
	}
}

func TestValidateAttribution_NoAdministrationInPowerOnContractDate(t *testing.T) {
	// Contract date before any known administration. Should return mismatch.
	admins := testAdmins()
	agreement := sampleAgreement("loan-1", "admin-uhuru", time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC))
	err := domain.ValidateAttribution(agreement, admins)
	if !errors.Is(err, domain.ErrAttributionMismatch) {
		t.Errorf("expected ErrAttributionMismatch; got %v", err)
	}
}

func TestValidateAttribution_LoanContractedBeforeRepaymentPeriod(t *testing.T) {
	// Spec section 4: a loan contracted in 2020 (Uhuru's term) but repaid in
	// 2024 (Ruto's term) is still CONTRACTED_DURING Uhuru's term.
	// ValidateAttribution must accept this attribution.
	admins := testAdmins()
	contractDate := time.Date(2020, 6, 15, 0, 0, 0, 0, time.UTC)
	agreement := sampleAgreement("loan-1", "admin-uhuru", contractDate)
	if err := domain.ValidateAttribution(agreement, admins); err != nil {
		t.Errorf("attribution by CONTRACTED_DURING should be valid; got %v", err)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle ||
		(len(haystack) > 0 && indexOf(haystack, needle) >= 0))
}

func indexOf(haystack, needle string) int {
	for i := 0; i <= len(haystack)-len(needle); i++ {
		if haystack[i:i+len(needle)] == needle {
			return i
		}
	}
	return -1
}
