package parliament

import (
	"context"
	"fmt"
	"strings"
)

// DiscoverBills fetches the Parliament of Kenya Bill Tracker page and parses
// it into a slice of BillRow records.
//
// Returns a non-nil error only when the fetch itself fails or the HTML is so
// malformed that no table could be parsed. Missing fields in individual rows
// are tolerated (the normalizer will flag them downstream).
func (p *ParliamentAdapter) DiscoverBills(ctx context.Context) ([]BillRow, error) {
	body, err := p.fetchSource(ctx, SourceBillTracker, "bills")
	if err != nil {
		return nil, err
	}
	if body == nil {
		// 304 Not Modified: nothing to do this cycle.
		return nil, nil
	}
	rows, err := ParseBillsHTML(strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrParse, err)
	}
	return rows, nil
}

// BillStageCode attempts to map a raw stage string from the Bill Tracker
// (e.g. "Second Reading", "In Committee") to the adapter's StageCode enum.
// Returns "" if no confident mapping can be made; the caller should then set
// Stage=nil, Confidence=0, Reason="could not be determined" on the record.
func BillStageCode(raw string) string {
	lc := strings.ToLower(strings.TrimSpace(raw))
	if lc == "" {
		return ""
	}
	switch {
	case strings.Contains(lc, "first reading"):
		return "FIRST_READING"
	case strings.Contains(lc, "second reading"):
		return "SECOND_READING"
	case strings.Contains(lc, "committee stage") || strings.Contains(lc, "in committee"):
		return "COMMITTEE_STAGE"
	case strings.Contains(lc, "committee of the whole") || strings.Contains(lc, "whole house"):
		return "COMMITTEE_OF_WHOLE_HOUSE"
	case strings.Contains(lc, "report stage") || strings.Contains(lc, "reporting"):
		return "REPORT_STAGE"
	case strings.Contains(lc, "third reading"):
		return "THIRD_READING"
	case strings.Contains(lc, "mediation"):
		return "MEDIATION"
	case strings.Contains(lc, "assent") || strings.Contains(lc, "presidential"):
		return "PRESIDENTIAL_ASSENT"
	case strings.Contains(lc, "commencement") || strings.Contains(lc, "in force") || strings.Contains(lc, "operational"):
		return "COMMENCEMENT"
	case strings.Contains(lc, "rejected") || strings.Contains(lc, "lost"):
		return "REJECTED"
	case strings.Contains(lc, "withdrawn"):
		return "WITHDRAWN"
	case strings.Contains(lc, "lapsed"):
		return "LAPSED"
	}
	return ""
}

// BillExternalID derives a stable external ID for a BillRow. The Bill Tracker
// uses strings like "National Assembly Bill No. 23 of 2023"; we keep that
// verbatim (whitespace-normalised) as the external ID, which the normalizer
// uses to deduplicate across crawl cycles.
func BillExternalID(row BillRow) string {
	id := strings.TrimSpace(row.BillNo)
	if id == "" {
		return ""
	}
	// Collapse internal whitespace.
	fields := strings.Fields(id)
	return strings.Join(fields, " ")
}
