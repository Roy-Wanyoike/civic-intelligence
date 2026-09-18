package domain

import (
	"context"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// BillRepository is the persistence interface for canonical Bills.
// The implementation lives in infrastructure/ and uses Postgres. The domain
// layer depends only on this interface — never on database/sql.
type BillRepository interface {
	Get(ctx context.Context, id contracts.ID) (*Bill, error)
	FindByIdentifier(ctx context.Context, country contracts.Country, identifier string, year int) (*Bill, error)
	List(ctx context.Context, country contracts.Country, page contracts.PageRequest, filter BillFilter) ([]*Bill, int64, error)
	Save(ctx context.Context, bill *Bill) error
	UpdateStage(ctx context.Context, id contracts.ID, newStage string) error
}

type BillFilter struct {
	Status string
	House  string
	Topic  string
	Query  string
}

// BillVersionRepository — IMMUTABLE. Save inserts; never updates.
type BillVersionRepository interface {
	GetCurrent(ctx context.Context, billID contracts.ID) (*BillVersion, error)
	List(ctx context.Context, billID contracts.ID) ([]*BillVersion, error)
	Save(ctx context.Context, version *BillVersion) error
	PromoteCurrent(ctx context.Context, billID contracts.ID, newVersionID contracts.ID) error
}

// BillEventRepository — append-only.
type BillEventRepository interface {
	List(ctx context.Context, billID contracts.ID) ([]*BillEvent, error)
	Save(ctx context.Context, event *BillEvent) error
}
