package domain

import (
        "context"
        "testing"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
        "github.com/stretchr/testify/assert"
)

// TestHashDeduplicator_PartitionsNewChangedUnchanged is the headline test
// for ingestion's dedup logic. It proves that:
//   - Items whose ExternalID is not in known are flagged as new.
//   - Items whose ContentHash matches are flagged as unchanged.
//   - Items whose ContentHash differs are flagged as changed.
//   - Items without a hash are conservatively flagged as new.
func TestHashDeduplicator_PartitionsNewChangedUnchanged(t *testing.T) {
        d := HashDeduplicator{}
        items := []contracts.SourceItem{
                {ExternalID: "A", ContentHash: "h1"},            // new
                {ExternalID: "B", ContentHash: "h2"},            // unchanged (matches known)
                {ExternalID: "C", ContentHash: "h3-different"}, // changed
                {ExternalID: "D", ContentHash: ""},             // new (no hash)
        }
        known := map[string]string{
                "B": "h2",
                "C": "h3-original",
        }
        newItems, changed, unchanged := d.Partition(items, known)
        assert.Len(t, newItems, 2, "A and D should be new")
        assert.Len(t, changed, 1, "C should be changed")
        assert.Len(t, unchanged, 1, "B should be unchanged")
        assert.Equal(t, "C", changed[0].ExternalID)
        assert.Equal(t, "B", unchanged[0].ExternalID)
}

// TestSHA256Hasher_Stable verifies that the hash is deterministic and
// matches a known SHA-256 value, so dedup keys are stable across services.
func TestSHA256Hasher_Stable(t *testing.T) {
        h := SHA256Hasher{}
        got := h.Hash([]byte("hello world"))
        // Canonical SHA-256 of "hello world".
        assert.Equal(t, "b94d27b9934d3e08a52e52d7da7dabfac484efe37a5380ee9088f7ace2efcde9", got)
        assert.Equal(t, "", h.Hash(nil), "empty input should hash to empty string")
}

// TestAdapterRegistry_ResolveByCountryAndURL verifies the adapter registry
// can resolve by country code or by URL, and returns an error for unknown
// inputs (no silent fallback).
func TestAdapterRegistry_ResolveByCountryAndURL(t *testing.T) {
        // Use a tiny fake adapter so the test does not depend on the Kenya
        // adapter's network-URL allowlist.
        fake := &fakeAdapter{country: "ZZ", supportsURL: "https://example.org/x"}
        r := NewAdapterRegistry(fake)

        got, err := r.AdapterFor("ZZ")
        assert.NoError(t, err)
        assert.Equal(t, "ZZ", got.CountryCode())

        _, err = r.AdapterFor("KE")
        assert.Error(t, err, "unknown country must error")

        got2, err := r.AdapterForURL("https://example.org/x")
        assert.NoError(t, err)
        assert.Same(t, fake, got2)

        _, err = r.AdapterForURL("https://example.com/unknown")
        assert.Error(t, err)
}

// fakeAdapter is a minimal LegislativeSourceAdapter for tests.
type fakeAdapter struct {
        country     string
        supportsURL string
}

func (f *fakeAdapter) CountryCode() string { return f.country }
func (f *fakeAdapter) Supports(url string) bool { return url == f.supportsURL }
func (f *fakeAdapter) Discover(ctx context.Context) ([]contracts.SourceItem, error) {
        return nil, nil
}
func (f *fakeAdapter) Fetch(ctx context.Context, item contracts.SourceItem) (*contracts.RawDocument, error) {
        return nil, nil
}
func (f *fakeAdapter) Parse(ctx context.Context, doc contracts.RawDocument) ([]contracts.ExtractedRecord, error) {
        return nil, nil
}
func (f *fakeAdapter) NormalizeSourceItem(raw map[string]any) (contracts.SourceItem, error) {
        return contracts.SourceItem{CountryCode: f.country}, nil
}
func (f *fakeAdapter) GetLegislativeStructure(ctx context.Context) (*contracts.LegislativeStructure, error) {
        s := contracts.LegislativeStructure{Country: contracts.Country(f.country), CountryCode: f.country}
        return &s, nil
}
func (f *fakeAdapter) GetStages(ctx context.Context) ([]contracts.StageDefinition, error) {
        return nil, nil
}
func (f *fakeAdapter) GetTerminology(ctx context.Context) ([]contracts.TermDefinition, error) {
        return nil, nil
}

// Keep imports honest.
var _ = context.Background
