// Package contract contains cross-cutting contract tests that verify
// every country adapter, every NATS event subject, and every route
// registered in main.go conforms to the documented contracts.
//
// The tests live in their own Go module so they can be run independently
// of each adapter's own go.mod. Replace directives point at the parent
// monorepo packages.
module github.com/Roy-Wanyoike/civic-intelligence/tests/contract

go 1.23

require (
	github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/registry v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/tanzania v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/packages/contracts v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/packages/events v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/services/legislation v0.0.0
)

require golang.org/x/net v0.30.0 // indirect

replace (
	github.com/Roy-Wanyoike/civic-intelligence => ../..
	github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana => ../../adapters/ghana
	github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya => ../../adapters/kenya
	github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria => ../../adapters/nigeria
	github.com/Roy-Wanyoike/civic-intelligence/adapters/registry => ../../adapters/registry
	github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa => ../../adapters/south_africa
	github.com/Roy-Wanyoike/civic-intelligence/adapters/tanzania => ../../adapters/tanzania
	github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda => ../../adapters/uganda
	github.com/Roy-Wanyoike/civic-intelligence/packages/contracts => ../../packages/contracts
	github.com/Roy-Wanyoike/civic-intelligence/packages/events => ../../packages/events
	github.com/Roy-Wanyoike/civic-intelligence/packages/observability => ../../packages/observability
	github.com/Roy-Wanyoike/civic-intelligence/services/legislation => ../../services/legislation
)
