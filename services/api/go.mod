module github.com/Roy-Wanyoike/civic-intelligence/services/api

go 1.23

require (
	github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/registry v0.0.0
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
	github.com/Roy-Wanyoike/civic-intelligence/adapters/burkina_faso v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/cameroon v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/dr_congo v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/egypt v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/ethiopia v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/ivory_coast v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/malawi v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/morocco v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/mozambique v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/niger v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/rwanda v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/senegal v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/tanzania v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda v0.0.0 // indirect
	github.com/Roy-Wanyoike/civic-intelligence/adapters/zambia v0.0.0 // indirect
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
	github.com/Roy-Wanyoike/civic-intelligence/adapters/burkina_faso => ../../adapters/burkina_faso
	github.com/Roy-Wanyoike/civic-intelligence/adapters/cameroon => ../../adapters/cameroon
	github.com/Roy-Wanyoike/civic-intelligence/adapters/dr_congo => ../../adapters/dr_congo
	github.com/Roy-Wanyoike/civic-intelligence/adapters/egypt => ../../adapters/egypt
	github.com/Roy-Wanyoike/civic-intelligence/adapters/ethiopia => ../../adapters/ethiopia
	github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana => ../../adapters/ghana
	github.com/Roy-Wanyoike/civic-intelligence/adapters/ivory_coast => ../../adapters/ivory_coast
	github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya => ../../adapters/kenya
	github.com/Roy-Wanyoike/civic-intelligence/adapters/malawi => ../../adapters/malawi
	github.com/Roy-Wanyoike/civic-intelligence/adapters/morocco => ../../adapters/morocco
	github.com/Roy-Wanyoike/civic-intelligence/adapters/mozambique => ../../adapters/mozambique
	github.com/Roy-Wanyoike/civic-intelligence/adapters/niger => ../../adapters/niger
	github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria => ../../adapters/nigeria
	github.com/Roy-Wanyoike/civic-intelligence/adapters/registry => ../../adapters/registry
	github.com/Roy-Wanyoike/civic-intelligence/adapters/rwanda => ../../adapters/rwanda
	github.com/Roy-Wanyoike/civic-intelligence/adapters/senegal => ../../adapters/senegal
	github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa => ../../adapters/south_africa
	github.com/Roy-Wanyoike/civic-intelligence/adapters/tanzania => ../../adapters/tanzania
	github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda => ../../adapters/uganda
	github.com/Roy-Wanyoike/civic-intelligence/adapters/zambia => ../../adapters/zambia
	github.com/Roy-Wanyoike/civic-intelligence/packages/auth => ../../packages/auth
	github.com/Roy-Wanyoike/civic-intelligence/packages/config => ../../packages/config
	github.com/Roy-Wanyoike/civic-intelligence/packages/contracts => ../../packages/contracts
	github.com/Roy-Wanyoike/civic-intelligence/packages/observability => ../../packages/observability
	github.com/Roy-Wanyoike/civic-intelligence/services/legislation => ../../services/legislation
	github.com/Roy-Wanyoike/civic-intelligence/services/simulation => ../../services/simulation
)
