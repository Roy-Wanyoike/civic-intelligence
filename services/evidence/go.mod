module github.com/Roy-Wanyoike/civic-intelligence/services/evidence

go 1.22

require (
	github.com/Roy-Wanyoike/civic-intelligence/packages/contracts v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/packages/events v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/packages/observability v0.0.0
	github.com/jackc/pgx/v5 v5.5.5
	github.com/nats-io/nats.go v1.34.0
	github.com/rs/zerolog v1.32.0
	github.com/stretchr/testify v1.9.0
)

replace (
	github.com/Roy-Wanyoike/civic-intelligence/packages/contracts => ../../packages/contracts
	github.com/Roy-Wanyoike/civic-intelligence/packages/events => ../../packages/events
	github.com/Roy-Wanyoike/civic-intelligence/packages/observability => ../../packages/observability
)
