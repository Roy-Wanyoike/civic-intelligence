module github.com/Roy-Wanyoike/civic-intelligence/services/api

go 1.23

require (
	github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya v0.0.0
<<<<<<< HEAD
=======
	github.com/Roy-Wanyoike/civic-intelligence/adapters/registry v0.0.0
>>>>>>> 67fdf85b4e02633df3f08e295597fbde145fcfc1
	github.com/Roy-Wanyoike/civic-intelligence/packages/auth v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/packages/config v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/packages/observability v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/services/legislation v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/services/simulation v0.0.0

	// OIDC signature verification (P0-3 / issue #59) + JWKS fetch dedup.
	// FIXME: verify with go build when Go available — run `go mod tidy` to
	// populate go.sum entries for these two modules.
	github.com/go-jose/go-jose/v3 v3.0.3
	golang.org/x/sync v0.10.0
)

require (
<<<<<<< HEAD
	github.com/Roy-Wanyoike/civic-intelligence/packages/contracts v0.0.0 // indirect
	golang.org/x/crypto v0.19.0 // indirect
)

replace (
	github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya => ../../adapters/kenya
=======
	github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/tanzania v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/packages/contracts v0.0.0 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0 // indirect
	go.opentelemetry.io/otel v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.28.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.28.0 // indirect
	go.opentelemetry.io/otel/metric v1.28.0 // indirect
	go.opentelemetry.io/otel/sdk v1.28.0 // indirect
	go.opentelemetry.io/otel/trace v1.28.0 // indirect
	go.opentelemetry.io/proto/otlp v1.3.1 // indirect
	golang.org/x/crypto v0.28.0 // indirect
	golang.org/x/net v0.30.0 // indirect
	golang.org/x/sys v0.26.0 // indirect
	golang.org/x/text v0.19.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240701130421-f6361c86f094 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240701130421-f6361c86f094 // indirect
	google.golang.org/grpc v1.64.0 // indirect
	google.golang.org/protobuf v1.34.2 // indirect
)

replace (
	github.com/Roy-Wanyoike/civic-intelligence => ../..
	github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana => ../../adapters/ghana
	github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya => ../../adapters/kenya
	github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria => ../../adapters/nigeria
	github.com/Roy-Wanyoike/civic-intelligence/adapters/registry => ../../adapters/registry
	github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa => ../../adapters/south_africa
	github.com/Roy-Wanyoike/civic-intelligence/adapters/tanzania => ../../adapters/tanzania
	github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda => ../../adapters/uganda
>>>>>>> 67fdf85b4e02633df3f08e295597fbde145fcfc1
	github.com/Roy-Wanyoike/civic-intelligence/packages/auth => ../../packages/auth
	github.com/Roy-Wanyoike/civic-intelligence/packages/config => ../../packages/config
	github.com/Roy-Wanyoike/civic-intelligence/packages/contracts => ../../packages/contracts
	github.com/Roy-Wanyoike/civic-intelligence/packages/observability => ../../packages/observability
	github.com/Roy-Wanyoike/civic-intelligence/services/legislation => ../../services/legislation
	github.com/Roy-Wanyoike/civic-intelligence/services/simulation => ../../services/simulation
)
