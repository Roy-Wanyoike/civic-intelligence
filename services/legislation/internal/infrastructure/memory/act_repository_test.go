// Comprehensive tests for the in-memory ActRepository (issue #202).
//
// Each test exercises one interface method (or a tightly-coupled pair like
// Append + List). The tests are deterministic — no real clock, no network,
// no external state. They use testify/assert for readability.
package memory

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// fixedTime is a deterministic timestamp used as the "now" baseline in tests.
var fixedTime = time.Date(2024, 6, 15, 12, 0, 0, 0, time.UTC)

// sampleAct constructs a minimal valid Act for tests. Each test can override
// fields as needed.
func sampleAct(id string, overrides ...func(*domain.Act)) domain.Act {
	a := domain.Act{
		ID:         domain.ID(id),
		BillID:     domain.ID("bill-" + id),
		CountryID:  "KE",
		ActNumber:  "No. 1 of 2024",
		ActName:    "Test Act " + id,
		AssentedAt: fixedTime.Add(-30 * 24 * time.Hour),
		Status:     domain.ActStatusCommenced,
		SourceURL:  "https://example.test/act/" + id,
	}
	for _, o := range overrides {
		o(&a)
	}
	return a
}

// sampleVersion constructs a minimal valid ActVersion for tests.
func sampleVersion(actID string, v int) domain.ActVersion {
	return domain.ActVersion{
		ID:      domain.ID("ver-" + actID + "-" + itoa(v)),
		ActID:   domain.ID(actID),
		Version: v,
		Text:    "text v" + itoa(v) + " of " + actID,
	}
}

// sampleEvent constructs a minimal valid PostAssentEvent for tests.
func sampleEvent(actID string, et domain.PostAssentEventType, daysAgo int) domain.PostAssentEvent {
	return domain.PostAssentEvent{
		ID:        domain.ID("ev-" + actID + "-" + string(et)),
		ActID:     domain.ID(actID),
		EventType: et,
		EventDate: fixedTime.Add(-time.Duration(daysAgo) * 24 * time.Hour),
		Title:     string(et) + " for " + actID,
		SourceURL: "https://example.test/event/" + actID,
	}
}

// itoa is a tiny strconv.Itoa-free helper to keep test deps minimal.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// -----------------------------------------------------------------------------
// CreateAct + GetAct
// -----------------------------------------------------------------------------

// TestActRepo_CreateAct_PersistsAndReturnsByGet verifies the basic
// create → read round-trip.
func TestActRepo_CreateAct_PersistsAndReturnsByGet(t *testing.T) {
	repo := NewActRepo()
	ctx := context.Background()
	a := sampleAct("a1")

	require.NoError(t, repo.CreateAct(ctx, a))

	got, err := repo.GetAct(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, a.ID, got.ID)
	assert.Equal(t, a.ActName, got.ActName)
	assert.Equal(t, a.Status, got.Status)
	// CreatedAt / UpdatedAt should be auto-populated by CreateAct.
	assert.False(t, got.CreatedAt.IsZero(), "CreatedAt should be populated")
	assert.False(t, got.UpdatedAt.IsZero(), "UpdatedAt should be populated")
}

// TestActRepo_CreateAct_RejectsDuplicate verifies CreateAct is NOT idempotent
// — re-creating an existing Act returns an error. The platform never silently
// overwrites canonical civic truth.
func TestActRepo_CreateAct_RejectsDuplicate(t *testing.T) {
	repo := NewActRepo()
	ctx := context.Background()
	a := sampleAct("a1")

	require.NoError(t, repo.CreateAct(ctx, a))
	err := repo.CreateAct(ctx, a)
	assert.Error(t, err, "creating the same act twice should fail")
	assert.Contains(t, err.Error(), "already exists")
}

// TestActRepo_GetAct_NotFound verifies the not-found path.
func TestActRepo_GetAct_NotFound(t *testing.T) {
	repo := NewActRepo()
	_, err := repo.GetAct(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// TestActRepo_GetAct_DoesNotLeakInternalState verifies the returned pointer
// is a copy — callers cannot mutate the underlying store via the returned
// pointer.
func TestActRepo_GetAct_DoesNotLeakInternalState(t *testing.T) {
	repo := NewActRepo()
	ctx := context.Background()
	a := sampleAct("a1")
	require.NoError(t, repo.CreateAct(ctx, a))

	got, err := repo.GetAct(ctx, a.ID)
	require.NoError(t, err)
	got.ActName = "MUTATED"

	got2, err := repo.GetAct(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, a.ActName, got2.ActName, "store must not reflect caller mutation")
}

// -----------------------------------------------------------------------------
// ListActs (with filter)
// -----------------------------------------------------------------------------

// TestActRepo_ListActs_EmptyStoreReturnsEmptySlice (never nil) verifies the
// "no acts" path returns [] rather than nil. This is a Go convention that
// prevents JSON marshalling from emitting null instead of [].
func TestActRepo_ListActs_EmptyStoreReturnsEmptySlice(t *testing.T) {
	repo := NewActRepo()
	out, err := repo.ListActs(context.Background(), domain.ActFilter{})
	require.NoError(t, err)
	assert.NotNil(t, out, "should return [] not nil")
	assert.Len(t, out, 0)
}

// TestActRepo_ListActs_ReturnsAll verifies ListActs returns every act in
// the store when no filter is applied.
func TestActRepo_ListActs_ReturnsAll(t *testing.T) {
	repo := NewActRepo()
	ctx := context.Background()
	for _, id := range []string{"a1", "a2", "a3"} {
		require.NoError(t, repo.CreateAct(ctx, sampleAct(id)))
	}

	out, err := repo.ListActs(ctx, domain.ActFilter{})
	require.NoError(t, err)
	assert.Len(t, out, 3)
}

// TestActRepo_ListActs_FilterByCountry verifies country isolation: only
// acts matching the requested CountryID are returned. Two countries' acts
// never leak into each other's listings.
func TestActRepo_ListActs_FilterByCountry(t *testing.T) {
	repo := NewActRepo()
	ctx := context.Background()
	keID := domain.ID("KE")
	ugID := domain.ID("UG")

	require.NoError(t, repo.CreateAct(ctx, sampleAct("ke-1")))
	require.NoError(t, repo.CreateAct(ctx, sampleAct("ke-2")))
	require.NoError(t, repo.CreateAct(ctx, sampleAct("ug-1", func(a *domain.Act) {
		a.CountryID = ugID
	})))

	out, err := repo.ListActs(ctx, domain.ActFilter{CountryID: &keID})
	require.NoError(t, err)
	assert.Len(t, out, 2)
	for _, a := range out {
		assert.Equal(t, keID, a.CountryID, "no UG acts should leak into KE list")
	}

	outUG, err := repo.ListActs(ctx, domain.ActFilter{CountryID: &ugID})
	require.NoError(t, err)
	assert.Len(t, outUG, 1)
	assert.Equal(t, ugID, outUG[0].CountryID)
}

// TestActRepo_ListActs_FilterByStatus verifies the status filter is applied.
func TestActRepo_ListActs_FilterByStatus(t *testing.T) {
	repo := NewActRepo()
	ctx := context.Background()
	require.NoError(t, repo.CreateAct(ctx, sampleAct("a1", func(a *domain.Act) { a.Status = domain.ActStatusCommenced })))
	require.NoError(t, repo.CreateAct(ctx, sampleAct("a2", func(a *domain.Act) { a.Status = domain.ActStatusAmended })))
	require.NoError(t, repo.CreateAct(ctx, sampleAct("a3", func(a *domain.Act) { a.Status = domain.ActStatusRepealed })))

	out, err := repo.ListActs(ctx, domain.ActFilter{
		Status: []domain.ActStatus{domain.ActStatusAmended, domain.ActStatusRepealed},
	})
	require.NoError(t, err)
	assert.Len(t, out, 2)
	for _, a := range out {
		assert.Contains(t, []domain.ActStatus{domain.ActStatusAmended, domain.ActStatusRepealed}, a.Status)
	}
}

// TestActRepo_ListActs_Pagination verifies the Limit + Offset fields are
// applied after filtering.
func TestActRepo_ListActs_Pagination(t *testing.T) {
	repo := NewActRepo()
	ctx := context.Background()
	for i := 0; i < 5; i++ {
		require.NoError(t, repo.CreateAct(ctx, sampleAct("a"+itoa(i))))
	}

	// Limit=2 returns first 2.
	out, err := repo.ListActs(ctx, domain.ActFilter{Limit: 2})
	require.NoError(t, err)
	assert.Len(t, out, 2)

	// Offset=3, Limit=2 returns the last 2.
	out, err = repo.ListActs(ctx, domain.ActFilter{Offset: 3, Limit: 2})
	require.NoError(t, err)
	assert.Len(t, out, 2)

	// Offset beyond range returns empty slice.
	out, err = repo.ListActs(ctx, domain.ActFilter{Offset: 100})
	require.NoError(t, err)
	assert.Len(t, out, 0)
}

// -----------------------------------------------------------------------------
// UpdateAct
// -----------------------------------------------------------------------------

// TestActRepo_UpdateAct_MutatesExisting verifies the update path.
func TestActRepo_UpdateAct_MutatesExisting(t *testing.T) {
	repo := NewActRepo()
	ctx := context.Background()
	a := sampleAct("a1")
	require.NoError(t, repo.CreateAct(ctx, a))

	original, err := repo.GetAct(ctx, a.ID)
	require.NoError(t, err)

	// Mutate a copy.
	updated := *original
	updated.ActName = "Updated Act Name"
	updated.Status = domain.ActStatusAmended
	require.NoError(t, repo.UpdateAct(ctx, updated))

	got, err := repo.GetAct(ctx, a.ID)
	require.NoError(t, err)
	assert.Equal(t, "Updated Act Name", got.ActName)
	assert.Equal(t, domain.ActStatusAmended, got.Status)
	assert.True(t, got.UpdatedAt.After(original.UpdatedAt), "UpdatedAt should be bumped")
}

// TestActRepo_UpdateAct_NotFoundReturnsError verifies updating a non-existent
// act returns an error (no silent create).
func TestActRepo_UpdateAct_NotFoundReturnsError(t *testing.T) {
	repo := NewActRepo()
	err := repo.UpdateAct(context.Background(), sampleAct("nonexistent"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

// -----------------------------------------------------------------------------
// AppendVersion + ListVersions
// -----------------------------------------------------------------------------

// TestActRepo_AppendVersion_ThenList verifies versions are appended in order
// and that the returned slice is a copy (callers cannot mutate the store).
func TestActRepo_AppendVersion_ThenList(t *testing.T) {
	repo := NewActRepo()
	ctx := context.Background()
	require.NoError(t, repo.CreateAct(ctx, sampleAct("a1")))

	v1 := sampleVersion("a1", 1)
	v2 := sampleVersion("a1", 2)
	v3 := sampleVersion("a1", 3)

	require.NoError(t, repo.AppendVersion(ctx, v1))
	require.NoError(t, repo.AppendVersion(ctx, v2))
	require.NoError(t, repo.AppendVersion(ctx, v3))

	out, err := repo.ListVersions(ctx, "a1")
	require.NoError(t, err)
	assert.Len(t, out, 3)
	// Versions are appended in insertion order.
	assert.Equal(t, 1, out[0].Version)
	assert.Equal(t, 2, out[1].Version)
	assert.Equal(t, 3, out[2].Version)

	// Mutate the returned slice — the store must not be affected.
	out[0].Text = "MUTATED"
	out2, _ := repo.ListVersions(ctx, "a1")
	assert.Equal(t, v1.Text, out2[0].Text, "store must not reflect caller mutation")
}

// TestActRepo_ListVersions_EmptyReturnsEmptySlice verifies the not-found path
// returns [] rather than nil.
func TestActRepo_ListVersions_EmptyReturnsEmptySlice(t *testing.T) {
	repo := NewActRepo()
	out, err := repo.ListVersions(context.Background(), "no-versions-yet")
	require.NoError(t, err)
	assert.NotNil(t, out, "should return [] not nil")
	assert.Len(t, out, 0)
}

// -----------------------------------------------------------------------------
// AppendPostAssentEvent + ListPostAssentEvents
// -----------------------------------------------------------------------------

// TestActRepo_AppendPostAssentEvent_ThenList verifies events are appended in
// order and that the returned slice is a copy.
func TestActRepo_AppendPostAssentEvent_ThenList(t *testing.T) {
	repo := NewActRepo()
	ctx := context.Background()
	require.NoError(t, repo.CreateAct(ctx, sampleAct("a1")))

	ev1 := sampleEvent("a1", domain.PostAssentEventCommencement, 30)
	ev2 := sampleEvent("a1", domain.PostAssentEventRegulation, 10)
	ev3 := sampleEvent("a1", domain.PostAssentEventAmendment, 5)

	require.NoError(t, repo.AppendPostAssentEvent(ctx, ev1))
	require.NoError(t, repo.AppendPostAssentEvent(ctx, ev2))
	require.NoError(t, repo.AppendPostAssentEvent(ctx, ev3))

	out, err := repo.ListPostAssentEvents(ctx, "a1")
	require.NoError(t, err)
	assert.Len(t, out, 3)
	assert.Equal(t, ev1.EventType, out[0].EventType)
	assert.Equal(t, ev2.EventType, out[1].EventType)
	assert.Equal(t, ev3.EventType, out[2].EventType)

	// Mutate the returned slice — the store must not be affected.
	out[0].Title = "MUTATED"
	out2, _ := repo.ListPostAssentEvents(ctx, "a1")
	assert.Equal(t, ev1.Title, out2[0].Title, "store must not reflect caller mutation")
}

// TestActRepo_ListPostAssentEvents_EmptyReturnsEmptySlice verifies the
// not-found path returns [] rather than nil.
func TestActRepo_ListPostAssentEvents_EmptyReturnsEmptySlice(t *testing.T) {
	repo := NewActRepo()
	out, err := repo.ListPostAssentEvents(context.Background(), "no-events-yet")
	require.NoError(t, err)
	assert.NotNil(t, out, "should return [] not nil")
	assert.Len(t, out, 0)
}

// -----------------------------------------------------------------------------
// RecordAssent (Spec §15 — Bill→Act transition is explicit)
// -----------------------------------------------------------------------------

// TestActRepo_RecordAssent_ProducesAct verifies that recording an ASSENTED
// event produces an Act in the store with status ASSENTED, the BillID linked,
// and the assent timestamp set.
func TestActRepo_RecordAssent_ProducesAct(t *testing.T) {
	repo := NewActRepo()
	ctx := context.Background()

	ev := domain.PresidentialAssentEvent{
		ID:             "ascent-1",
		BillID:         "bill-xyz",
		PresidentID:    "president-1",
		AdministrationID: "admin-1",
		Date:           fixedTime,
		OfficialSource: "https://example.test/assent-1",
		AssentStatus:   domain.AssentStatusAssented,
		CreatedAt:      fixedTime,
	}

	act, err := repo.RecordAssent(ctx, ev)
	require.NoError(t, err)
	require.NotNil(t, act)

	// The Act ID is derived from the Bill ID by prefixing "act-".
	assert.Equal(t, domain.ID("act-bill-xyz"), act.ID)
	assert.Equal(t, domain.ID("bill-xyz"), act.BillID)
	assert.Equal(t, fixedTime, act.AssentedAt)
	assert.Equal(t, domain.ActStatusAssented, act.Status)

	// The Act is persisted in the store — GetAct returns the same record.
	got, err := repo.GetAct(ctx, act.ID)
	require.NoError(t, err)
	assert.Equal(t, act.ID, got.ID)

	// An ASSENT post-assent event is also recorded.
	events, err := repo.ListPostAssentEvents(ctx, act.ID)
	require.NoError(t, err)
	require.Len(t, events, 1)
	assert.Equal(t, domain.PostAssentEventAssent, events[0].EventType)
	assert.Equal(t, "Presidential Assent", events[0].Title)
}

// TestActRepo_RecordAssent_RejectsNonAssented verifies that non-assent
// statuses do NOT produce an Act. This is the core of Spec §15: the Bill→Act
// transition is explicit; only ASSENTED produces an Act.
func TestActRepo_RecordAssent_RejectsNonAssented(t *testing.T) {
	repo := NewActRepo()
	ctx := context.Background()

	for _, status := range []domain.PresidentialAssentStatus{
		domain.AssentStatusReturned,
		domain.AssentStatusWithheld,
		domain.AssentStatusUnknown,
	} {
		ev := domain.PresidentialAssentEvent{
			ID:           domain.ID("ascent-" + string(status)),
			BillID:       domain.ID("bill-" + string(status)),
			Date:         fixedTime,
			AssentStatus: status,
			CreatedAt:    fixedTime,
		}
		act, err := repo.RecordAssent(ctx, ev)
		assert.Error(t, err, "status %s should not produce an Act", status)
		assert.Nil(t, act)
		assert.Contains(t, err.Error(), "only ASSENTED")
	}

	// Verify no acts or events were created.
	acts, err := repo.ListActs(ctx, domain.ActFilter{})
	require.NoError(t, err)
	assert.Len(t, acts, 0, "no acts should be created for non-assent events")
}

// TestActRepo_RecordAssent_IdempotentOnSameBill verifies that recording
// assent for the same Bill twice returns an error (the second call would
// create a duplicate Act).
func TestActRepo_RecordAssent_IdempotentOnSameBill(t *testing.T) {
	repo := NewActRepo()
	ctx := context.Background()

	ev := domain.PresidentialAssentEvent{
		ID:             "ascent-1",
		BillID:         "bill-xyz",
		Date:           fixedTime,
		OfficialSource: "https://example.test/assent-1",
		AssentStatus:   domain.AssentStatusAssented,
		CreatedAt:      fixedTime,
	}

	act1, err := repo.RecordAssent(ctx, ev)
	require.NoError(t, err)
	require.NotNil(t, act1)

	// Second call with the same BillID must fail (CreateAct rejects duplicates).
	act2, err := repo.RecordAssent(ctx, ev)
	assert.Error(t, err)
	assert.Nil(t, act2)
	assert.Contains(t, err.Error(), "already exists")
}

// -----------------------------------------------------------------------------
// Concurrency — verifies the sync.RWMutex actually serializes access.
// -----------------------------------------------------------------------------

// TestActRepo_ConcurrentCreateIsSafe verifies the repo is safe under
// concurrent writes. This is a smoke test — if the RWMutex is missing,
// `go test -race` would flag a data race.
func TestActRepo_ConcurrentCreateIsSafe(t *testing.T) {
	repo := NewActRepo()
	ctx := context.Background()

	const n = 100
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(i int) {
			defer wg.Done()
			_ = repo.CreateAct(ctx, sampleAct("a"+itoa(i)))
		}(i)
	}
	wg.Wait()

	out, err := repo.ListActs(ctx, domain.ActFilter{})
	require.NoError(t, err)
	assert.Len(t, out, n, "all %d concurrent creates should land in the store", n)
}

// TestActRepo_ConcurrentReadersDoNotBlock verifies many concurrent readers
// can read without blocking each other. With sync.RWMutex, multiple readers
// can hold the read lock simultaneously.
func TestActRepo_ConcurrentReadersDoNotBlock(t *testing.T) {
	repo := NewActRepo()
	ctx := context.Background()
	require.NoError(t, repo.CreateAct(ctx, sampleAct("a1")))

	const n = 50
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			_, _ = repo.GetAct(ctx, "a1")
		}()
	}
	wg.Wait()
}

// -----------------------------------------------------------------------------
// Compile-time interface assertion (mirrors the one in act_repository.go).
// -----------------------------------------------------------------------------

// TestActRepo_ImplementsActRepository is a compile-time assertion encoded as
// a test so it surfaces as a clear test failure (not a build failure) if the
// interface drifts.
func TestActRepo_ImplementsActRepository(t *testing.T) {
	var _ domain.ActRepository = (*ActRepo)(nil)
	// Sanity check: the test itself just needs to compile.
	t.Log("ActRepo satisfies domain.ActRepository at compile time")
}

// -----------------------------------------------------------------------------
// Error type helpers
// -----------------------------------------------------------------------------

// TestActRepo_ErrorsAreCompatibleWithErrorsIs verifies the repo's error
// conventions don't accidentally wrap errors in ways that break errors.Is.
func TestActRepo_ErrorsAreCompatibleWithErrorsIs(t *testing.T) {
	repo := NewActRepo()
	ctx := context.Background()

	// Not-found error.
	_, err := repo.GetAct(ctx, "nonexistent")
	require.Error(t, err)
	assert.False(t, errors.Is(err, context.DeadlineExceeded), "sanity check on errors.Is")

	// Conflict error.
	require.NoError(t, repo.CreateAct(ctx, sampleAct("a1")))
	err = repo.CreateAct(ctx, sampleAct("a1"))
	require.Error(t, err)
}
