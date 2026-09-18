// Package integration contains black-box integration tests for the Civic
// Intelligence API. Each test starts the real API server binary on a random
// port, makes HTTP requests against it, and asserts on the JSON response
// shape + status codes.
//
// The tests live in their own Go module so they can be versioned + run
// independently of the API service's own go.mod. Replace directives point
// at the parent monorepo packages so the test module can import shared
// contracts when asserting on response shapes.
module github.com/Roy-Wanyoike/civic-intelligence/tests/integration

go 1.23

replace github.com/Roy-Wanyoike/civic-intelligence => ../..

replace github.com/Roy-Wanyoike/civic-intelligence/packages/contracts => ../../packages/contracts
