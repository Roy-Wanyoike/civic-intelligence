// Package parliament — HTML parser for the Parliament of South Africa Bills
// listing page (parliament.gov.za/bills-and-laws).
//
// The Parliament of South Africa publishes Bills using a card-per-bill layout
// that looks (schematically) like:
//
//      <div class="bill-card">
//        <h3 class="bill-title"><a href=".../b12-2026.pdf">The National Rail Bill, 2026</a></h3>
//        <div class="bill-meta">
//          <span class="bill-number">B 12—2026</span>
//          <span class="bill-sponsor">Minister of Transport</span>
//          <span class="bill-portfolio-committee">Portfolio Committee on Transport</span>
//          <span class="bill-stage">NA Vote</span>
//          <span class="bill-date">01 March 2026</span>
//        </div>
//      </div>
//
// The parser is deliberately tolerant: a Bill card missing the meta block (or
// any individual span) is still returned, with the corresponding fields left
// empty. Malformed HTML never panics — it just yields fewer Bills.
//
package parliament

import (
	"fmt"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"golang.org/x/net/html"
)

// ParseBillsListing parses the HTML of a parliament.gov.za Bills listing
// page into a slice of BillCandidate records. The parser tolerates missing
// fields and never panics on malformed HTML.
func ParseBillsListing(pageHTML string) []BillCandidate {
	doc, err := html.Parse(strings.NewReader(pageHTML))
	if err != nil {
		return nil
	}
	var out []BillCandidate
	walkBillCards(doc, func(card *html.Node) {
		c := parseBillCard(card)
		if c.URL != "" || c.Title != "" {
			out = append(out, c)
		}
	})
	// De-duplicate by URL (a Bill may be linked more than once).
	seen := make(map[string]bool, len(out))
	deduped := out[:0]
	for _, c := range out {
		key := c.URL
		if key == "" {
			key = c.Title
		}
		if seen[key] {
			continue
		}
		seen[key] = true
		deduped = append(deduped, c)
	}
	return deduped
}

// ParseBillDetail extracts Bill metadata from a parliament.gov.za Bill detail
// page (or, in the absence of a per-Bill detail page, from the listing's
// bill-card markup). Returns a single ExtractedRecord of kind "bill". If no
// recognisable Bill title is found, an error is returned.
func ParseBillDetail(pageHTML, sourceURL string) ([]contracts.ExtractedRecord, error) {
	bills := ParseBillsListing(pageHTML)
	if len(bills) == 0 {
		title, pdfURL := firstAnchor(pageHTML)
		if title == "" {
			return nil, fmt.Errorf("south_africa.parliament.ParseBillDetail: no Bill title found in %s", sourceURL)
		}
		rec := contracts.ExtractedRecord{
			Kind:          "bill",
			Title:         title,
			SourceURL:     sourceURL,
			RetrievedAt:   time.Now().UTC(),
			Confidence:    0.55,
			ExtractorName: "south_africa.parliament.HTMLParser",
		}
		if pdfURL != "" {
			rec.Extra = map[string]interface{}{"pdf_url": pdfURL}
		}
		return []contracts.ExtractedRecord{rec}, nil
	}
	b := bills[0]
	house := b.House
	if house == "" {
		house = "National Assembly" // Bills originate in the NA by default
	}
	rec := contracts.ExtractedRecord{
		Kind:          "bill",
		Title:         b.Title,
		House:         house,
		Stage:         MapStageText(b.BillNumber),
		Sponsor:       b.Sponsor,
		PublishedAt:   b.DiscoveredAt,
		SourceURL:     sourceURL,
		RetrievedAt:   time.Now().UTC(),
		Confidence:    0.85,
		ExtractorName: "south_africa.parliament.HTMLParser",
	}
	if b.URL != "" {
		rec.Extra = map[string]interface{}{"pdf_url": b.URL}
	}
	if b.BillNumber != "" {
		if rec.Extra == nil {
			rec.Extra = map[string]interface{}{}
		}
		rec.Extra["bill_number"] = b.BillNumber
	}
	if b.PortfolioCommittee != "" {
		if rec.Extra == nil {
			rec.Extra = map[string]interface{}{}
		}
		rec.Extra["portfolio_committee"] = b.PortfolioCommittee
	}
	return []contracts.ExtractedRecord{rec}, nil
}

// parseBillCard extracts a single BillCandidate from a
// <div class="bill-card"> node.
func parseBillCard(card *html.Node) BillCandidate {
	var c BillCandidate
	walkAnchors(card, func(_ *html.Node, href, text string) {
		if c.URL == "" && href != "" {
			c.URL = href
			if c.Title == "" {
				c.Title = strings.TrimSpace(text)
			}
		}
	})
	walkSpans(card, func(cls, text string) {
		switch cls {
		case "bill-number":
			c.BillNumber = strings.TrimSpace(text)
		case "bill-sponsor":
			c.Sponsor = strings.TrimSpace(text)
		case "bill-portfolio-committee":
			c.PortfolioCommittee = strings.TrimSpace(text)
		case "bill-stage":
			c.BillNumber = strings.TrimSpace(text)
		case "bill-date":
			c.DiscoveredAt = parseSADate(strings.TrimSpace(text))
		case "bill-house":
			c.House = strings.TrimSpace(text)
		}
	})
	return c
}

// walkBillCards visits every <div class="bill-card"> node and invokes fn.
func walkBillCards(n *html.Node, fn func(*html.Node)) {
	if n == nil {
		return
	}
	if n.Type == html.ElementNode && n.Data == "div" && hasClass(n, "bill-card") {
		fn(n)
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkBillCards(c, fn)
	}
}

// walkAnchors visits every <a> node and invokes fn with its href and text.
func walkAnchors(n *html.Node, fn func(a *html.Node, href, text string)) {
	if n == nil {
		return
	}
	if n.Type == html.ElementNode && n.Data == "a" {
		fn(n, attrOf(n, "href"), textOf(n))
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkAnchors(c, fn)
	}
}

// walkSpans visits every <span> node with a class attribute, invoking fn with
// the first class token and the span's trimmed text.
func walkSpans(n *html.Node, fn func(cls, text string)) {
	if n == nil {
		return
	}
	if n.Type == html.ElementNode && n.Data == "span" {
		for _, a := range n.Attr {
			if a.Key == "class" {
				fields := strings.Fields(a.Val)
				if len(fields) > 0 {
					fn(fields[0], strings.TrimSpace(textOf(n)))
				}
				break
			}
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkSpans(c, fn)
	}
}

// hasClass reports whether n has the given class in its class attribute.
func hasClass(n *html.Node, want string) bool {
	for _, a := range n.Attr {
		if a.Key != "class" {
			continue
		}
		for _, c := range strings.Fields(a.Val) {
			if c == want {
				return true
			}
		}
	}
	return false
}

// attrOf returns the value of n's attribute named key, or "" if absent.
func attrOf(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

// textOf returns the trimmed concatenation of all descendant text nodes.
func textOf(n *html.Node) string {
	if n == nil {
		return ""
	}
	var sb strings.Builder
	var visit func(*html.Node)
	visit = func(node *html.Node) {
		if node == nil {
			return
		}
		if node.Type == html.TextNode {
			sb.WriteString(node.Data)
		}
		for c := node.FirstChild; c != nil; c = c.NextSibling {
			visit(c)
		}
	}
	visit(n)
	return strings.TrimSpace(sb.String())
}

// firstAnchor returns the trimmed text + href of the first <a> in the HTML.
func firstAnchor(pageHTML string) (text, href string) {
	doc, err := html.Parse(strings.NewReader(pageHTML))
	if err != nil {
		return "", ""
	}
	var foundText, foundHref string
	walkAnchors(doc, func(_ *html.Node, h, t string) {
		if foundText == "" && strings.TrimSpace(t) != "" {
			foundText = strings.TrimSpace(t)
			foundHref = h
		}
	})
	return foundText, foundHref
}

// parseSADate tries the common date layouts used by parliament.gov.za.
func parseSADate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	layouts := []string{
		"2006-01-02",
		"02/01/2006",
		"02 January 2006",
		"02 Jan 2006",
		"January 02, 2006",
		"Jan 02, 2006",
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}
