// Package temporal defines the Bill ingestion workflow.
package temporal

// KenyaBillSyncWorkflow orchestrates the full Bill ingestion pipeline.
type KenyaBillSyncWorkflow struct{}

type KenyaBillSyncWorkflowInput struct {
	CountryCode string
	BillID      string
	SourceURL   string
}
