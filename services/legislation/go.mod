module github.com/Roy-Wanyoike/civic-intelligence/services/legislation

go 1.25.0

require github.com/stretchr/testify v1.9.0

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

require github.com/Roy-Wanyoike/civic-intelligence/packages/contracts v0.0.0

replace github.com/Roy-Wanyoike/civic-intelligence => ../../

replace github.com/Roy-Wanyoike/civic-intelligence/packages/contracts => ../../packages/contracts

replace github.com/Roy-Wanyoike/civic-intelligence/packages/observability => ../../packages/observability

replace github.com/Roy-Wanyoike/civic-intelligence/packages/events => ../../packages/events
