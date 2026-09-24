module github.com/Roy-Wanyoike/civic-intelligence/adapters/registry

go 1.23

require (
	github.com/Roy-Wanyoike/civic-intelligence/adapters/burkina_faso v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/cameroon v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/dr_congo v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/egypt v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/ethiopia v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/ivory_coast v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/malawi v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/morocco v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/mozambique v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/niger v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/rwanda v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/senegal v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/tanzania v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/adapters/zambia v0.0.0
	github.com/Roy-Wanyoike/civic-intelligence/packages/contracts v0.0.0
)

require golang.org/x/net v0.30.0 // indirect

replace (
	github.com/Roy-Wanyoike/civic-intelligence => ../..
	github.com/Roy-Wanyoike/civic-intelligence/adapters/burkina_faso => ../burkina_faso
	github.com/Roy-Wanyoike/civic-intelligence/adapters/cameroon => ../cameroon
	github.com/Roy-Wanyoike/civic-intelligence/adapters/dr_congo => ../dr_congo
	github.com/Roy-Wanyoike/civic-intelligence/adapters/egypt => ../egypt
	github.com/Roy-Wanyoike/civic-intelligence/adapters/ethiopia => ../ethiopia
	github.com/Roy-Wanyoike/civic-intelligence/adapters/ghana => ../ghana
	github.com/Roy-Wanyoike/civic-intelligence/adapters/ivory_coast => ../ivory_coast
	github.com/Roy-Wanyoike/civic-intelligence/adapters/kenya => ../kenya
	github.com/Roy-Wanyoike/civic-intelligence/adapters/malawi => ../malawi
	github.com/Roy-Wanyoike/civic-intelligence/adapters/morocco => ../morocco
	github.com/Roy-Wanyoike/civic-intelligence/adapters/mozambique => ../mozambique
	github.com/Roy-Wanyoike/civic-intelligence/adapters/niger => ../niger
	github.com/Roy-Wanyoike/civic-intelligence/adapters/nigeria => ../nigeria
	github.com/Roy-Wanyoike/civic-intelligence/adapters/rwanda => ../rwanda
	github.com/Roy-Wanyoike/civic-intelligence/adapters/senegal => ../senegal
	github.com/Roy-Wanyoike/civic-intelligence/adapters/south_africa => ../south_africa
	github.com/Roy-Wanyoike/civic-intelligence/adapters/tanzania => ../tanzania
	github.com/Roy-Wanyoike/civic-intelligence/adapters/uganda => ../uganda
	github.com/Roy-Wanyoike/civic-intelligence/adapters/zambia => ../zambia
	github.com/Roy-Wanyoike/civic-intelligence/packages/contracts => ../../packages/contracts
	github.com/Roy-Wanyoike/civic-intelligence/packages/observability => ../../packages/observability
	github.com/Roy-Wanyoike/civic-intelligence/services/legislation => ../../services/legislation
)
