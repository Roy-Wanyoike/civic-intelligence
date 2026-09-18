package gazette

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// ParseNoticesHTML parses the Kenya Gazette index page into a slice of
// GazetteEntry records.
//
// The index page lists individual notices with links to their full PDFs.
// The parser is tolerant of layout drift; missing fields are left blank.
func ParseNoticesHTML(r io.Reader) ([]GazetteEntry, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("gazette parser: %w", err)
	}
	var out []GazetteEntry
	walkAnchors(doc, func(href, text string) {
		if !looksLikeNotice(href, text) {
			return
		}
		out = append(out, GazetteEntry{
			Volume:      extractVolume(text),
			NoticeNo:    extractNoticeNo(text),
			Title:       strings.TrimSpace(text),
			Issuer:      extractIssuer(text),
			PublishedAt: parseDateInText(text),
			URL:         href,
		})
	})
	return out, nil
}

// ParseNoticeText parses the plain text of a single Kenya Gazette notice
// (whether extracted from a born-digital PDF or produced by OCR) into a
// structured GazetteEntry.
//
// The parser looks for the standard Gazette notice header, which typically
// includes a volume, a notice number, a publication date, and an issuer.
func ParseNoticeText(text string) GazetteEntry {
	entry := GazetteEntry{
		Volume:      extractVolume(text),
		NoticeNo:    extractNoticeNo(text),
		Title:       firstNonEmptyLine(text),
		Issuer:      extractIssuer(text),
		PublishedAt: parseDateInText(text),
	}
	return entry
}

// --- internal helpers ---

var (
	volumeRe   = regexp.MustCompile(`(?i)\bvol\.?\s*([IVXLCDM]+)\s*[-–—]\s*no\.?\s*(\d+)\b`)
	noticeNoRe = regexp.MustCompile(`(?i)\bnotice\s+no\.?\s*(\d+)\b`)
	dateRe     = regexp.MustCompile(`\b(\d{1,2})\s+(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{4})\b`)
	issuerRe   = regexp.MustCompile(`(?i)\b(Ministry of [A-Z][\w\s,&-]+|Attorney General|Solicitor General|Judicial Service Commission|Independent Electoral and Boundaries Commission|Ethics and Anti-Corruption Commission|Public Service Commission|Commission on Revenue Allocation|National Police Service Commission)\b`)
)

// walkAnchors visits every <a> node and invokes fn with its href and text.
func walkAnchors(n *html.Node, fn func(href, text string)) {
	if n == nil {
		return
	}
	if n.Type == html.ElementNode && n.Data == "a" {
		href := attrOf(n, "href")
		text := textOf(n)
		if href != "" {
			fn(href, text)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkAnchors(c, fn)
	}
}

func attrOf(n *html.Node, key string) string {
	for _, a := range n.Attr {
		if a.Key == key {
			return a.Val
		}
	}
	return ""
}

func textOf(n *html.Node) string {
	if n == nil {
		return ""
	}
	if n.Type == html.TextNode {
		return n.Data
	}
	var sb strings.Builder
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		sb.WriteString(textOf(c))
	}
	return sb.String()
}

func looksLikeNotice(href, text string) bool {
	lc := strings.ToLower(href + " " + text)
	if strings.Contains(lc, "notice") || strings.Contains(lc, "gazette") {
		// Filter out navigation noise.
		if strings.Contains(lc, "login") || strings.Contains(lc, "subscribe") {
			return false
		}
		return true
	}
	return false
}

func extractVolume(text string) string {
	m := volumeRe.FindStringSubmatch(text)
	if len(m) >= 3 {
		return "Vol. " + strings.ToUpper(m[1]) + " — No. " + m[2]
	}
	return ""
}

func extractNoticeNo(text string) string {
	m := noticeNoRe.FindStringSubmatch(text)
	if len(m) >= 2 {
		return "Notice No. " + m[1]
	}
	return ""
}

func extractIssuer(text string) string {
	m := issuerRe.FindStringSubmatch(text)
	if len(m) >= 1 {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// parseDateInText extracts the first plausible publication date from a notice
// text or link. Returns the zero time if nothing parseable was found.
func parseDateInText(text string) time.Time {
	m := dateRe.FindStringSubmatch(text)
	if len(m) < 4 {
		return time.Time{}
	}
	day := atoiSafe(m[1])
	month := monthOf(m[2])
	year := atoiSafe(m[3])
	if day == 0 || month < 1 || year == 0 {
		return time.Time{}
	}
	return time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
}

func atoiSafe(s string) int {
	var n int
	_, err := fmt.Sscanf(strings.TrimSpace(s), "%d", &n)
	if err != nil {
		return 0
	}
	return n
}

func monthOf(name string) int {
	switch strings.ToLower(strings.TrimSpace(name)) {
	case "january":
		return 1
	case "february":
		return 2
	case "march":
		return 3
	case "april":
		return 4
	case "may":
		return 5
	case "june":
		return 6
	case "july":
		return 7
	case "august":
		return 8
	case "september":
		return 9
	case "october":
		return 10
	case "november":
		return 11
	case "december":
		return 12
	}
	return 0
}

func firstNonEmptyLine(text string) string {
	for _, line := range strings.Split(text, "\n") {
		if t := strings.TrimSpace(line); t != "" {
			return t
		}
	}
	return ""
}

// tryExtractPDFText attempts a best-effort extraction of plain text from a
// PDF. Real PDF text extraction is delegated to the documents service; this
// stub handles only the trivial case of PDFs that ship their text content
// uncompressed (a small fraction of born-digital PDFs).
//
// Returns (text, true) if any meaningful text was extracted; (text, false)
// otherwise (the caller should fall back to OCR).
func tryExtractPDFText(pdfBytes []byte) (string, bool) {
	// Look for "BT ... ET" text-object blocks containing "Tj" operators that
	// carry string literals. This is a tiny subset of the PDF spec but it
	// detects text-bearing PDFs cheaply without pulling in a heavyweight
	// PDF library.
	content := string(pdfBytes)
	if !strings.Contains(content, "BT") || !strings.Contains(content, "Tj") {
		return "", false
	}
	// Extract strings in (... ) Tj and <...> Tj operators.
	tjRe := regexp.MustCompile(`\(([^()\\]*(?:\\.[^()\\]*)*)\)\s*Tj`)
	matches := tjRe.FindAllStringSubmatch(content, -1)
	if len(matches) == 0 {
		return "", false
	}
	var sb strings.Builder
	for i, m := range matches {
		if i > 0 {
			sb.WriteString("\n")
		}
		// Unescape basic PDF string escapes.
		s := m[1]
		s = strings.ReplaceAll(s, `\(`, "(")
		s = strings.ReplaceAll(s, `\)`, ")")
		s = strings.ReplaceAll(s, `\\`, `\`)
		sb.WriteString(s)
	}
	text := strings.TrimSpace(sb.String())
	if text == "" {
		return "", false
	}
	return text, true
}

// hashHex returns the lowercase hex SHA-256 of the input bytes. Used as a
// deterministic document ID for OCR caching.
func hashHex(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
