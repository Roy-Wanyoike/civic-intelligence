// Package main — small helpers for the public debt endpoints. Kept in a
// separate file so public_debt.go reads as a linear spec walkthrough.
package main

import (
        "time"

        kenya_seed "github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya/kenya_seed"
)

// farPastDate is a sentinel "from" value used when listing snapshots without
// a lower bound. Year 1 is far enough that any real observation will be
// included.
func farPastDate() time.Time {
        return time.Date(1, 1, 1, 0, 0, 0, 0, time.UTC)
}

// farFutureDate is a sentinel "to" value used when listing snapshots without
// an upper bound. Year 9999 is far enough that any real observation will be
// included.
func farFutureDate() time.Time {
        return time.Date(9999, 12, 31, 23, 59, 59, 0, time.UTC)
}

// kenyaDebtDashboardFigures returns the dashboard-level derived figures
// (FY 2023/24 debt service + debt-to-GDP ratio) sourced from the Kenya
// Treasury Annual Public Debt Report 2023/24 and IMF WEO.
//
// These figures are NOT derived from snapshot deltas — Spec section 9
// forbids inferring new borrowing from (end - start). They are explicit,
// authoritative values that the Kenya seed package owns.
func kenyaDebtDashboardFigures() (debtService, debtToGDP float64) {
        return kenya_seed.DebtDashboardFigures()
}
