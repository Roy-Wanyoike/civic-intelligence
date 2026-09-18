module github.com/Roy-Wanyoike/civic-intelligence/services/api

go 1.23

require (
        github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya v0.0.0
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

require github.com/Roy-Wanyoike/civic-intelligence/packages/contracts v0.0.0 // indirect

replace (
        github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya => ../../adapters/kenya
        github.com/Roy-Wanyoike/civic-intelligence/packages/auth => ../../packages/auth
        github.com/Roy-Wanyoike/civic-intelligence/packages/config => ../../packages/config
        github.com/Roy-Wanyoike/civic-intelligence/packages/contracts => ../../packages/contracts
        github.com/Roy-Wanyoike/civic-intelligence/packages/observability => ../../packages/observability
        github.com/Roy-Wanyoike/civic-intelligence/services/legislation => ../../services/legislation
        github.com/Roy-Wanyoike/civic-intelligence/services/simulation => ../../services/simulation
)
