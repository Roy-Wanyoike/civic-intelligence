module github.com/Roy-Wanyoike/civic-intelligence/services/api

go 1.22

require (
	github.com/Roy-Wanyoike/civic-intelligence/packages/auth v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/packages/config v0.0.0
)

replace (
	github.com/Roy-Wanyoike/civic-intelligence/packages/auth => ../../packages/auth
	github.com/Roy-Wanyoike/civic-intelligence/packages/config => ../../packages/config
)
