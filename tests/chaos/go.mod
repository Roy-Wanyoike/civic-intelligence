// Package chaos is the standalone Go module for Production Gate #11
// (chaos engineering) contract tests.
//
// The tests in this package are STANDALONE: they do not import any
// production package. They define minimal handler shapes that mirror the
// production contracts the chaos runbooks (tests/chaos/*.md) verify
// manually. See chaos_test.go for the full rationale.
module github.com/Roy-Wanyoike/civic-intelligence/tests/chaos

go 1.23
