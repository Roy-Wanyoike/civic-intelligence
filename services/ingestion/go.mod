module github.com/Roy-Wanyoike/civic-intelligence/services/ingestion

go 1.25.0

require (
	github.com/Roy-Wanyoike/civic-intelligence/packages/config v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/packages/contracts v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/packages/observability v0.0.0
	github.com/nats-io/nats.go v1.53.1
	github.com/stretchr/testify v1.11.1
	go.temporal.io/sdk v1.33.0
)

require (
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/facebookgo/clock v0.0.0-20150410010913-600d898af40a // indirect
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
	github.com/Roy-Wanyoike/civic-intelligence/packages/config => ../../packages/config
	github.com/Roy-Wanyoike/civic-intelligence/packages/contracts => ../../packages/contracts
	github.com/Roy-Wanyoike/civic-intelligence/packages/events => ../../packages/events
	github.com/Roy-Wanyoike/civic-intelligence/packages/observability => ../../packages/observability
)
