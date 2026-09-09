package application

import (
	"context"
	"strings"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"github.com/Roy-Wanyoike/civic-intelligence/services/legislation/internal/domain"
)

// GetBillHandler loads a single bill by ID.
type GetBillHandler struct {
	repo domain.BillRepository
}

// NewGetBillHandler constructs the handler.
func NewGetBillHandler(repo domain.BillRepository) *GetBillHandler {
	return &GetBillHandler{repo: repo}
}

// Handle executes the query.
func (h *GetBillHandler) Handle(ctx context.Context, id domain.ID) (*domain.Bill, error) {
	return h.repo.Get(ctx, id)
}

// ListBillsHandler returns a paginated list of bills matching a filter.
type ListBillsHandler struct {
	repo domain.BillRepository
}

// NewListBillsHandler constructs the handler.
func NewListBillsHandler(repo domain.BillRepository) *ListBillsHandler {
	return &ListBillsHandler{repo: repo}
}

// Handle executes the query.
func (h *ListBillsHandler) Handle(ctx context.Context, filter domain.BillFilter, page contracts.PageRequest) (contracts.Page[*domain.Bill], error) {
	return h.repo.List(ctx, filter, page)
}

// GetBillTimelineHandler returns the chronological list of BillEvents for a
// bill. The timeline is what the API exposes as
// GET /api/v1/bills/:id/timeline.
type GetBillTimelineHandler struct {
	events domain.BillEventRepository
}

// NewGetBillTimelineHandler constructs the handler.
func NewGetBillTimelineHandler(events domain.BillEventRepository) *GetBillTimelineHandler {
	return &GetBillTimelineHandler{events: events}
}

// Handle executes the query.
func (h *GetBillTimelineHandler) Handle(ctx context.Context, billID domain.ID) ([]domain.BillEvent, error) {
	return h.events.ListByBill(ctx, billID)
}

// GetCurrentVersionHandler returns the latest BillVersion for a bill.
type GetCurrentVersionHandler struct {
	versions domain.BillVersionRepository
}

// NewGetCurrentVersionHandler constructs the handler.
func NewGetCurrentVersionHandler(versions domain.BillVersionRepository) *GetCurrentVersionHandler {
	return &GetCurrentVersionHandler{versions: versions}
}

// Handle executes the query.
func (h *GetCurrentVersionHandler) Handle(ctx context.Context, billID domain.ID) (*domain.BillVersion, error) {
	return h.versions.CurrentForBill(ctx, billID)
}

// ListVersionsHandler returns all versions of a bill.
type ListVersionsHandler struct {
	versions domain.BillVersionRepository
}

// NewListVersionsHandler constructs the handler.
func NewListVersionsHandler(versions domain.BillVersionRepository) *ListVersionsHandler {
	return &ListVersionsHandler{versions: versions}
}

// Handle executes the query.
func (h *ListVersionsHandler) Handle(ctx context.Context, billID domain.ID) ([]domain.BillVersion, error) {
	return h.versions.ListByBill(ctx, billID)
}

// CompareVersionsResult is the output of comparing two bill versions. The
// Diff is a line-oriented unified diff as text (the heavy lifting happens in
// the documents service; legislation only stores hashes).
type CompareVersionsResult struct {
	FromVersion  domain.BillVersion
	ToVersion    domain.BillVersion
	Identical    bool
	Diff         string
}

// CompareVersionsHandler compares two bill versions by hash and produces a
// minimal unified diff placeholder. The actual text diff is delegated to
// the documents service when the caller has access to document content.
type CompareVersionsHandler struct {
	versions domain.BillVersionRepository
}

// NewCompareVersionsHandler constructs the handler.
func NewCompareVersionsHandler(versions domain.BillVersionRepository) *CompareVersionsHandler {
	return &CompareVersionsHandler{versions: versions}
}

// Handle executes the query.
func (h *CompareVersionsHandler) Handle(ctx context.Context, fromID, toID domain.ID) (*CompareVersionsResult, error) {
	from, err := h.versions.Get(ctx, fromID)
	if err != nil {
		return nil, err
	}
	to, err := h.versions.Get(ctx, toID)
	if err != nil {
		return nil, err
	}
	res := &CompareVersionsResult{FromVersion: *from, ToVersion: *to}
	if from.TextHash == to.TextHash {
		res.Identical = true
		res.Diff = ""
		return res, nil
	}
	// Provide a lightweight textual diff hint without loading full text.
	var b strings.Builder
	b.WriteString("--- version " + itoa(from.VersionNumber) + " (hash " + shortHash(from.TextHash) + ")\n")
	b.WriteString("+++ version " + itoa(to.VersionNumber) + " (hash " + shortHash(to.TextHash) + ")\n")
	if from.ChangesSummary != "" || to.ChangesSummary != "" {
		b.WriteString("@@ changes @@\n")
		if from.ChangesSummary != "" {
			b.WriteString("- " + from.ChangesSummary + "\n")
		}
		if to.ChangesSummary != "" {
			b.WriteString("+ " + to.ChangesSummary + "\n")
		}
	}
	res.Diff = b.String()
	return res, nil
}

// itoa is a tiny dependency-free int->string helper for the diff builder.
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

// shortHash returns the first 8 characters of a hash for compact display.
func shortHash(h string) string {
	if len(h) <= 8 {
		return h
	}
	return h[:8]
}
