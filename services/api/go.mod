module github.com/Roy-Wanyoike/civic-intelligence/services/api

go 1.22

require (
	github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/packages/auth v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/packages/config v0.0.0
)

require github.com/Roy-Wanyoike/civic-intelligence/packages/contracts v0.0.0 // indirect

replace (
	github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya => ../../adapters/kenya
	github.com/Roy-Wanyoike/civic-intelligence/packages/auth => ../../packages/auth
	github.com/Roy-Wanyoike/civic-intelligence/packages/config => ../../packages/config
	github.com/Roy-Wanyoike/civic-intelligence/packages/contracts => ../../packages/contracts
)
