module github.com/Roy-Wanyoike/civic-intelligence/services/ingestion

go 1.25.0

require (
        github.com/Roy-Wanyoike/civic-intelligence/adapters/registry v0.0.0
        github.com/Roy-Wanyoike/civic-intelligence/packages/config v0.0.0
        github.com/Roy-Wanyoike/civic-intelligence/packages/contracts v0.0.0
        github.com/Roy-Wanyoike/civic-intelligence/packages/observability v0.0.0
        github.com/nats-io/nats.go v1.53.1
        github.com/stretchr/testify v1.11.1
        go.temporal.io/sdk v1.33.0
)

require (
        github.com/Roy-Wanyoike/civic-intelligence/adapters/dr_congo v0.0.0 // indirect
        github.com/Roy-Wanyoike/civic-intelligence/adapters/egypt v0.0.0 // indirect
        github.com/Roy-Wanyoike/civic-intelligence/adapters/ethiopia v0.0.0 // indirect
        github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana v0.0.0 // indirect
        github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya v0.0.0 // indirect
        github.com/Roy-Wanyoike/civic-intelligence/adapters/malawi v0.0.0 // indirect
        github.com/Roy-Wanyoike/civic-intelligence/adapters/morocco v0.0.0 // indirect
        github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria v0.0.0 // indirect
        github.com/Roy-Wanyoike/civic-intelligence/adapters/rwanda v0.0.0 // indirect
        github.com/Roy-Wanyoike/civic-intelligence/adapters/senegal v0.0.0 // indirect
        github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa v0.0.0 // indirect
        github.com/Roy-Wanyoike/civic-intelligence/adapters/tanzania v0.0.0 // indirect
        github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda v0.0.0 // indirect
        github.com/Roy-Wanyoike/civic-intelligence/adapters/zambia v0.0.0 // indirect
        github.com/cenkalti/backoff/v4 v4.3.0 // indirect
        github.com/davecgh/go-spew v1.1.1 // indirect
        github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
        github.com/go-logr/logr v1.4.2 // indirect
        github.com/go-logr/stdr v1.2.2 // indirect
        github.com/gogo/protobuf v1.3.2 // indirect
        github.com/golang/mock v1.6.0 // indirect
        github.com/google/uuid v1.6.0 // indirect
        github.com/grpc-ecosystem/go-grpc-middleware v1.4.0 // indirect
        github.com/grpc-ecosystem/grpc-gateway/v2 v2.22.0 // indirect
        github.com/klauspost/compress v1.18.5 // indirect
        github.com/nats-io/nkeys v0.4.15 // indirect
        github.com/nats-io/nuid v1.0.1 // indirect
        github.com/nexus-rpc/sdk-go v0.3.0 // indirect
        github.com/pborman/uuid v1.2.1 // indirect
        github.com/pmezard/go-difflib v1.0.0 // indirect
        github.com/robfig/cron v1.2.0 // indirect
        github.com/stretchr/objx v0.5.2 // indirect
        go.opentelemetry.io/otel v1.28.0 // indirect
        go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.28.0 // indirect
        go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.28.0 // indirect
        go.opentelemetry.io/otel/metric v1.28.0 // indirect
        go.opentelemetry.io/otel/sdk v1.28.0 // indirect
        go.opentelemetry.io/otel/trace v1.28.0 // indirect
        go.opentelemetry.io/proto/otlp v1.3.1 // indirect
        go.temporal.io/api v1.44.1 // indirect
        golang.org/x/crypto v0.49.0 // indirect
        golang.org/x/net v0.51.0 // indirect
        golang.org/x/sync v0.20.0 // indirect
        golang.org/x/sys v0.42.0 // indirect
        golang.org/x/text v0.35.0 // indirect
        golang.org/x/time v0.3.0 // indirect
        google.golang.org/genproto/googleapis/api v0.0.0-20240827150818-7e3bb234dfed // indirect
        google.golang.org/genproto/googleapis/rpc v0.0.0-20240827150818-7e3bb234dfed // indirect
        google.golang.org/grpc v1.66.0 // indirect
        google.golang.org/protobuf v1.34.2 // indirect
        gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
        github.com/Roy-Wanyoike/civic-intelligence/adapters/dr_congo => ../../adapters/dr_congo
        github.com/Roy-Wanyoike/civic-intelligence/adapters/egypt => ../../adapters/egypt
        github.com/Roy-Wanyoike/civic-intelligence/adapters/ethiopia => ../../adapters/ethiopia
        github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana => ../../adapters/ghana
        github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya => ../../adapters/kenya
        github.com/Roy-Wanyoike/civic-intelligence/adapters/malawi => ../../adapters/malawi
        github.com/Roy-Wanyoike/civic-intelligence/adapters/morocco => ../../adapters/morocco
        github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria => ../../adapters/nigeria
        github.com/Roy-Wanyoike/civic-intelligence/adapters/registry => ../../adapters/registry
        github.com/Roy-Wanyoike/civic-intelligence/adapters/rwanda => ../../adapters/rwanda
        github.com/Roy-Wanyoike/civic-intelligence/adapters/senegal => ../../adapters/senegal
        github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa => ../../adapters/south_africa
        github.com/Roy-Wanyoike/civic-intelligence/adapters/tanzania => ../../adapters/tanzania
        github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda => ../../adapters/uganda
        github.com/Roy-Wanyoike/civic-intelligence/adapters/zambia => ../../adapters/zambia
        github.com/Roy-Wanyoike/civic-intelligence/packages/config => ../../packages/config
        github.com/Roy-Wanyoike/civic-intelligence/packages/contracts => ../../packages/contracts
        github.com/Roy-Wanyoike/civic-intelligence/packages/events => ../../packages/events
        github.com/Roy-Wanyoike/civic-intelligence/packages/observability => ../../packages/observability
)
