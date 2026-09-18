// Package kenya_law — HTML parser for bill detail pages.
package kenya_law

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// billTitleRe extracts the title from the <title> tag.
var billTitleRe = regexp.MustCompile(`<title>\s*(.+?)\s*-\s*Kenya Law\s*</title>`)

// chamberRe extracts the chamber (house) from the definition list.
var chamberRe = regexp.MustCompile(`<dt[^>]*>\s*Chamber\s*</dt>\s*<dd[^>]*>\s*([^<]+)\s*</dd>`)

// dateRe extracts the date from the definition list.
var dateRe = regexp.MustCompile(`<dt[^>]*>\s*Date\s*</dt>\s*<dd[^>]*>\s*([^<]+)\s*</dd>`)

// pdfLinkRe extracts the PDF download link.
var pdfLinkRe = regexp.MustCompile(`href="([^"]+\.pdf[^"]*)"`)

// ParseBillDetail extracts Bill metadata from a Kenya Law bill detail page.
func ParseBillDetail(html, sourceURL string) ([]contracts.ExtractedRecord, error) {
	title := extractTitle(html)
	if title == "" {
		return nil, fmt.Errorf("could not extract bill title from %s", sourceURL)
	}

	house := extractChamber(html)
	pubDate := extractDate(html)
	pdfURL := extractPDFLink(html)

	record := contracts.ExtractedRecord{
		Kind:        "bill",
		Title:       title,
		House:       house,
		PublishedAt: pubDate,
		SourceURL:   sourceURL,
		RetrievedAt: time.Now().UTC(),
		Confidence:  0.9, // high confidence — this is the official source
	}

	// If we found a PDF link, add it as extra metadata.
	if pdfURL != "" {
		record.Extra = map[string]interface{}{
			"pdf_url": pdfURL,
		}
	}

	return []contracts.ExtractedRecord{record}, nil
}

func extractTitle(html string) string {
	m := billTitleRe.FindStringSubmatch(html)
	if len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func extractChamber(html string) string {
	m := chamberRe.FindStringSubmatch(html)
	if len(m) >= 2 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

func extractDate(html string) time.Time {
	m := dateRe.FindStringSubmatch(html)
	if len(m) >= 2 {
		dateStr := strings.TrimSpace(m[1])
		// Try parsing "7 September 2026" format
		t, err := time.Parse("2 January 2006", dateStr)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}

func extractPDFLink(html string) string {
	m := pdfLinkRe.FindStringSubmatch(html)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}
