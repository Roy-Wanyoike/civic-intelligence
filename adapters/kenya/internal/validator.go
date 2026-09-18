package internal

import (
        "errors"
        "fmt"
        "strings"
        "time"

        "github.com/Roy-Wanyoike/civic-intelligence/packages/contracts"
)

// ValidationResult is the output of the validator. The Errors slice is empty
// when the record is fully valid; otherwise each entry describes one defect.
// Warnings describe problems that should be flagged for human review but do
// not block ingestion.
type ValidationResult struct {
        Errors   []string
        Warnings []string
}

// Valid reports whether the validation result contains no errors.
func (v ValidationResult) Valid() bool { return len(v.Errors) == 0 }

// Validator validates extracted records against Kenya-specific invariants:
//   - required fields (source URL, retrieved_at, content_hash) are present;
//   - dates are sane (no future publication beyond +7 days tolerance);
//   - stage transitions are valid per KenyaBillStages;
//   - cross-source consistency (e.g. Parliament tracker vs Kenya Law) —
//     where the same Bill is observed from multiple sources, the most recent
//     stage is taken and any earlier stage from another source is treated
//     as a warning rather than an error.
type Validator struct {
        now                func() time.Time
        futureToleranceDays int
}

// NewValidator returns a Validator with sensible defaults: now() = UTC, and
// a 7-day tolerance for future-dated items (covers clocks skew + Gazette
// notices dated ahead of publication).
func NewValidator() *Validator {
        return &Validator{
                now:                func() time.Time { return time.Now().UTC() },
                futureToleranceDays: 7,
        }
}

// WithClock overrides the now() clock (for tests).
func (v *Validator) WithClock(now func() time.Time) *Validator {
        cp := *v
        cp.now = now
        return &cp
}

// WithFutureTolerance overrides the future-date tolerance (days).
func (v *Validator) WithFutureTolerance(days int) *Validator {
        cp := *v
        cp.futureToleranceDays = days
        return &cp
}

// Validate checks a single SourceItem plus its extracted record.
//
// SourceItem carries the provenance (source URL, retrieved_at, content_hash);
// ExtractedRecord carries the structured content (page count, confidence,
// metadata). The validator ensures both shapes are well-formed for the
// downstream legislation service to consume.
func (v *Validator) Validate(item contracts.SourceItem, rec contracts.ExtractedRecord) ValidationResult {
        res := ValidationResult{}

        // Required provenance fields.
        if item.URL == "" {
                res.Errors = append(res.Errors, "source_item: url is required (no provenance)")
        }
        if item.ExternalID == "" {
                res.Errors = append(res.Errors, "source_item: external_id is required")
        }
        if item.CountryCode != "KE" {
                res.Errors = append(res.Errors, fmt.Sprintf("source_item: country_code must be KE, got %q", item.CountryCode))
        }
        if item.ContentHash == "" {
                res.Errors = append(res.Errors, "source_item: content_hash is required (SHA-256)")
        }

        // DiscoveredAt should be set (the normalizer always sets it, but be safe).
        if item.DiscoveredAt.IsZero() {
                res.Errors = append(res.Errors, "source_item: discovered_at is required")
        }

        // Future-tolerance check on publication date (some Gazette notices are
        // pre-dated for forward commencement).
        if !item.PublishedAt.IsZero() {
                horizon := v.now().AddDate(0, 0, v.futureToleranceDays)
                if item.PublishedAt.After(horizon) {
                        res.Errors = append(res.Errors,
                                fmt.Sprintf("source_item: published_at %s is more than %d days in the future",
                                        item.PublishedAt.Format(time.RFC3339), v.futureToleranceDays))
                }
                if item.PublishedAt.Before(time.Unix(0, 0)) {
                        res.Errors = append(res.Errors, "source_item: published_at is before 1970 (implausible)")
                }
        }

        // Stage transition: encoded in RawMetadata["stage_code"]; if present it must
        // be a known Kenya stage, and (where prior_stage_code is also present)
        // the transition must be permitted by KenyaBillStages.
        if stageCode, ok := item.RawMetadata["stage_code"]; ok && stageCode != "" {
                if FindStage(StageCode(stageCode)) == nil {
                        res.Errors = append(res.Errors,
                                fmt.Sprintf("source_item: stage_code %q is not a known Kenya stage", stageCode))
                }
                if prev, ok := item.RawMetadata["prior_stage_code"]; ok && prev != "" {
                        if !CanTransition(StageCode(prev), StageCode(stageCode)) {
                                res.Warnings = append(res.Warnings,
                                        fmt.Sprintf("source_item: stage transition %s -> %s is not permitted by KenyaBillStages",
                                                prev, stageCode))
                        }
                }
        }

        // ExtractedRecord sanity.
        if rec.Title == "" {
                res.Warnings = append(res.Warnings, "extracted_record: title is empty")
        }
        if rec.Confidence < 0 || rec.Confidence > 1 {
                res.Errors = append(res.Errors,
                        fmt.Sprintf("extracted_record: confidence %f out of [0,1]", rec.Confidence))
        }
        if rec.ExtractedAt.IsZero() {
                res.Errors = append(res.Errors, "extracted_record: extracted_at is required")
        }
        if rec.ExtractorName == "" {
                res.Errors = append(res.Errors, "extracted_record: extractor_name is required")
        }
        if len(rec.Pages) == 0 {
                res.Warnings = append(res.Warnings, "extracted_record: no pages parsed")
        }

        return res
}

// stageClaim is a per-source claim about a Bill's stage, used internally by
// CrossSourceConsistency.
type stageClaim struct {
        stageCode string
        stage     *KenyaStage
        srcURL    string
}

// CrossSourceConsistency inspects a set of SourceItems that all reference the
// same external Bill (by ExternalID) and checks for inconsistent stage claims.
//
// The rules are intentionally permissive: the *latest* stage (by Kenya's
// stage ordering) wins; older claims become warnings. Conflicting claims of
// equal order become errors, since they indicate a genuine inconsistency
// that humans should arbitrate.
func (v *Validator) CrossSourceConsistency(items []contracts.SourceItem) ValidationResult {
        res := ValidationResult{}
        if len(items) < 2 {
                return res
        }

        var claims []stageClaim
        for _, it := range items {
                sc, ok := it.RawMetadata["stage_code"]
                if !ok || sc == "" {
                        continue
                }
                s := FindStage(StageCode(sc))
                if s == nil {
                        res.Errors = append(res.Errors,
                                fmt.Sprintf("cross-source: stage_code %q on %s not recognised", sc, it.URL))
                        continue
                }
                claims = append(claims, stageClaim{stageCode: sc, stage: s, srcURL: it.URL})
        }
        if len(claims) < 2 {
                return res
        }

        // Find max-order stage (the "latest"). Anything older becomes a warning.
        maxOrder := -1
        for _, c := range claims {
                if c.stage.Order > maxOrder {
                        maxOrder = c.stage.Order
                }
        }
        latestSet := map[string]bool{}
        for _, c := range claims {
                if c.stage.Order == maxOrder {
                        latestSet[c.stageCode] = true
                }
        }
        if len(latestSet) > 1 {
                // Multiple stages claim equal latest order — a genuine inconsistency.
                urls := make([]string, 0, len(claims))
                for _, c := range claims {
                        if c.stage.Order == maxOrder {
                                urls = append(urls, fmt.Sprintf("%s=%s", c.srcURL, c.stageCode))
                        }
                }
                res.Errors = append(res.Errors,
                        fmt.Sprintf("cross-source: conflicting latest stage: %s", strings.Join(urls, ", ")))
                return res
        }
        // Single latest winner; warn about older ones.
        latestLabel := ""
        for _, c := range claims {
                if c.stage.Order == maxOrder {
                        latestLabel = c.stageCode
                }
        }
        for _, c := range claims {
                if c.stage.Order < maxOrder {
                        res.Warnings = append(res.Warnings,
                                fmt.Sprintf("cross-source: %s reports older stage %s; latest is %s",
                                        c.srcURL, c.stageCode, latestLabel))
                }
        }
        return res
}

// ErrInvalid is returned by strict callers when Validate() reports any error.
var ErrInvalid = errors.New("kenya validator: record is invalid")
