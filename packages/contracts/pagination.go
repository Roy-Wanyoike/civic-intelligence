// Package contracts — generic pagination types.
package contracts

// PageRequest is the standard pagination query.
type PageRequest struct {
	Page     int `json:"page" form:"page" query:"page"`
	PageSize int `json:"page_size" form:"page_size" query:"page_size"`
}

// Defaults returns a PageRequest with defaults applied if zero.
func (p PageRequest) Defaults() PageRequest {
	if p.Page < 1 {
		p.Page = 1
	}
	if p.PageSize < 1 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	return p
}

// Offset returns the SQL OFFSET value for this page.
func (p PageRequest) Offset() int { return (p.Defaults().Page - 1) * p.Defaults().PageSize }

// Page is the standard paginated response wrapper.
type Page[T any] struct {
	Items    []T   `json:"items"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
}

// NewPage constructs a Page from a slice + total count.
func NewPage[T any](items []T, total int64, req PageRequest) Page[T] {
	r := req.Defaults()
	return Page[T]{Items: items, Total: total, Page: r.Page, PageSize: r.PageSize}
}
