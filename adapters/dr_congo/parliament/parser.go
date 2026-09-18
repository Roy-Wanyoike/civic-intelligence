// Package parliament — HTML parser for the Parliament of DR Congo Bills
// listing page (assemblee-nationale.cd/projets-lois).
//
// The Parliament of DR Congo publishes Bills using a card-per-bill layout that
// looks (schematically) like:
//
//	<div class="bill-card">
//	  <h3 class="bill-title"><a href=".../loi-donnees-personnelles.pdf">Loi portant mesures de protection des données personnelles</a></h3>
//	  <div class="bill-meta">
//	    <span class="bill-number">Projet de Loi N° 24/012</span>
//	    <span class="bill-sponsor">Ministre des Télécommunications, Numérique et Nouvelles Technologies</span>
//	    <span class="bill-stage">Première Lecture</span>
//	    <span class="bill-date">12 Mars 2024</span>
//	    <span class="bill-house">National Assembly</span>
//	  </div>
//	</div>
//
// The parser is deliberately tolerant: a Bill card missing the meta block
// (or any individual span) is still returned, with the corresponding fields
// left empty. Malformed HTML never panics — it just yields fewer Bills.
package parliament

import (
	"fmt"
	"strings"
	"time"

	"github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
	"golang.org/x/net/html"
)

// stageMapping maps the lowercased substrings the Parliament of DR Congo uses
// on its Bills pages to the canonical stage codes from internal/dr_congo_data.go.
// Order matters: more specific patterns MUST come first.
var stageMapping = []struct {
	needle string
	code   string
}{
	{"proposition", "PROPOSITION"},
	{"commission mixte", "JOINT_COMMISSION"},
	{"commission", "COMMISSION"},
	{"première lecture", "FIRST_READING"},
	{"premiere lecture", "FIRST_READING"},
	{"first reading", "FIRST_READING"},
	{"1ère lecture", "FIRST_READING"},
	{"lecture au sénat", "SENATE_READING"},
	{"lecture au senat", "SENATE_READING"},
	{"senate reading", "SENATE_READING"},
	{"sénat", "SENATE_READING"},
	{"deuxième lecture", "SECOND_READING"},
	{"deuxieme lecture", "SECOND_READING"},
	{"second reading", "SECOND_READING"},
	{"2ème lecture", "SECOND_READING"},
	{"adoption", "ADOPTION"},
	{"promulgation", "PROMULGATION"},
	{"promulgué", "PROMULGATION"},
	{"rejet", "REJECTED"},
	{"retiré", "WITHDRAWN"},
}

// MapStageText maps a raw stage string published by assemblee-nationale.cd to
// the canonical DR Congo stage code. Returns "" when no mapping is known.
func MapStageText(input string) string {
	if input == "" {
		return ""
	}
	lc := strings.ToLower(strings.TrimSpace(input))
	for _, m := range stageMapping {
		if strings.Contains(lc, m.needle) {
			return m.code
		}
	}
	return ""
}

// ParseBillsListing parses the HTML of an assemblee-nationale.cd Bills listing
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

// ParseBillDetail extracts Bill metadata from an assemblee-nationale.cd Bill
// detail page (or, in the absence of a per-Bill detail page, from the
// listing's bill-card markup). Returns a single ExtractedRecord of kind
// "bill". If no recognisable Bill title is found, an error is returned.
func ParseBillDetail(pageHTML, sourceURL string) ([]contracts.ExtractedRecord, error) {
	bills := ParseBillsListing(pageHTML)
	if len(bills) == 0 {
		// Fall back to the first <a> as the title.
		title, pdfURL := firstAnchor(pageHTML)
		if title == "" {
			return nil, fmt.Errorf("dr_congo.parliament.ParseBillDetail: no Bill title found in %s", sourceURL)
		}
		rec := contracts.ExtractedRecord{
			Kind:          "bill",
			Title:         title,
			SourceURL:     sourceURL,
			RetrievedAt:   time.Now().UTC(),
			Confidence:    0.55,
			ExtractorName: "dr_congo.parliament.HTMLParser",
		}
		if pdfURL != "" {
			rec.Extra = map[string]interface{}{"pdf_url": pdfURL}
		}
		return []contracts.ExtractedRecord{rec}, nil
	}
	b := bills[0]
	house := b.House
	if house == "" {
		house = "National Assembly"
	}
	rec := contracts.ExtractedRecord{
		Kind:          "bill",
		Title:         b.Title,
		House:         house,
		Stage:         MapStageText(b.Stage),
		Sponsor:       b.Sponsor,
		PublishedAt:   parseDRCongoDate(b.Date),
		SourceURL:     sourceURL,
		RetrievedAt:   time.Now().UTC(),
		Confidence:    0.85,
		ExtractorName: "dr_congo.parliament.HTMLParser",
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
		case "bill-stage":
			c.Stage = strings.TrimSpace(text)
		case "bill-date":
			c.Date = strings.TrimSpace(text)
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

// parseDRCongoDate tries the common date layouts used by
// assemblee-nationale.cd. Returns the zero time if nothing parseable was
// provided. French month names are normalised to English so time.Parse
// handles them.
func parseDRCongoDate(s string) time.Time {
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
	// Normalise French month names to English so time.Parse handles them.
	monthFix := map[string]string{
		"janvier": "January", "février": "February", "fevrier": "February",
		"mars": "March", "avril": "April", "mai": "May", "juin": "June",
		"juillet": "July", "août": "August", "aout": "August",
		"septembre": "September", "octobre": "October",
		"novembre": "November", "décembre": "December", "decembre": "December",
	}
	normalised := s
	lc := strings.ToLower(s)
	for fr, en := range monthFix {
		if strings.Contains(lc, fr) {
			idx := strings.Index(strings.ToLower(normalised), fr)
			for idx >= 0 {
				normalised = normalised[:idx] + en + normalised[idx+len(fr):]
				idx = strings.Index(strings.ToLower(normalised), fr)
			}
			break
		}
	}
	for _, l := range layouts {
		if t, err := time.Parse(l, normalised); err == nil {
			return t.UTC()
		}
		if t, err := time.Parse(l, s); err == nil {
			return t.UTC()
		}
	}
	return time.Time{}
}
