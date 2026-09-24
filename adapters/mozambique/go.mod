module github.com/Roy-Wanyoike/civic-intelligence/adapters/mozambique

go 1.23

toolchain go1.23.4

require (
	github.com/Roy-Wanyoike/civic-intelligence/packages/contracts v0.0.0
	github.com/stretchr/testify v1.9.0
	golang.org/x/net v0.30.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/Roy-Wanyoike/civic-intelligence => ../..
	github.com/Roy-Wanyoike/civic-intelligence/packages/contracts => ../../packages/contracts
	github.com/Roy-Wanyoike/civic-intelligence/packages/observability => ../../packages/observability
	github.com/Roy-Wanyoike/civic-intelligence/services/legislation => ../../services/legislation
)
