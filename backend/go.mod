module github.com/oneliang/company

go 1.26.1

require (
	github.com/google/uuid v1.6.0
	github.com/gorilla/mux v1.8.1
	github.com/gorilla/websocket v1.5.3
	github.com/oneliang/aura/core v0.0.0
	gopkg.in/yaml.v3 v3.0.1
)

require (
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.4.0 // indirect
	github.com/gofrs/flock v0.13.0 // indirect
	github.com/google/jsonschema-go v0.4.2 // indirect
	github.com/mark3labs/mcp-go v0.47.1 // indirect
	github.com/mattn/go-colorable v0.1.13 // indirect
	github.com/mattn/go-isatty v0.0.20 // indirect
	github.com/oneliang/aura/agent v0.0.0 // indirect
	github.com/oneliang/aura/commands v0.0.0 // indirect
	github.com/oneliang/aura/habit v0.0.0-00010101000000-000000000000 // indirect
	github.com/oneliang/aura/knowledge v0.0.0 // indirect
	github.com/oneliang/aura/mcp v0.0.0 // indirect
	github.com/oneliang/aura/personality v0.0.0 // indirect
	github.com/oneliang/aura/session v0.0.0 // indirect
	github.com/oneliang/aura/shared v0.0.0 // indirect
	github.com/oneliang/aura/skill v0.0.0-00010101000000-000000000000 // indirect
	github.com/oneliang/aura/storage v0.0.0 // indirect
	github.com/oneliang/aura/tools v0.0.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.4 // indirect
	github.com/philippgille/chromem-go v0.7.0 // indirect
	github.com/rs/zerolog v1.34.0 // indirect
	github.com/sagikazarmark/locafero v0.11.0 // indirect
	github.com/sourcegraph/conc v0.3.1-0.20240121214520-5f936abd7ae8 // indirect
	github.com/spf13/afero v1.15.0 // indirect
	github.com/spf13/cast v1.10.0 // indirect
	github.com/spf13/pflag v1.0.10 // indirect
	github.com/spf13/viper v1.21.0 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	github.com/yosida95/uritemplate/v3 v3.0.2 // indirect
	go.yaml.in/yaml/v3 v3.0.4 // indirect
	golang.org/x/crypto v0.48.0 // indirect
	golang.org/x/sync v0.20.0 // indirect
	golang.org/x/sys v0.43.0 // indirect
	golang.org/x/text v0.34.0 // indirect
)

// Local development - point to aura project
replace (
	github.com/oneliang/aura/adapters => /Users/oneliang/AgentProjects/aura/modules/adapters
	github.com/oneliang/aura/agent => /Users/oneliang/AgentProjects/aura/modules/agent
	github.com/oneliang/aura/api => /Users/oneliang/AgentProjects/aura/modules/api
	github.com/oneliang/aura/cli => /Users/oneliang/AgentProjects/aura/modules/cli
	github.com/oneliang/aura/commands => /Users/oneliang/AgentProjects/aura/modules/commands
	github.com/oneliang/aura/core => /Users/oneliang/AgentProjects/aura/modules/core
	github.com/oneliang/aura/habit => /Users/oneliang/AgentProjects/aura/modules/habit
	github.com/oneliang/aura/knowledge => /Users/oneliang/AgentProjects/aura/modules/knowledge
	github.com/oneliang/aura/mcp => /Users/oneliang/AgentProjects/aura/modules/mcp
	github.com/oneliang/aura/personality => /Users/oneliang/AgentProjects/aura/modules/personality
	github.com/oneliang/aura/session => /Users/oneliang/AgentProjects/aura/modules/session
	github.com/oneliang/aura/shared => /Users/oneliang/AgentProjects/aura/modules/shared
	github.com/oneliang/aura/skill => /Users/oneliang/AgentProjects/aura/modules/skill
	github.com/oneliang/aura/storage => /Users/oneliang/AgentProjects/aura/modules/storage
	github.com/oneliang/aura/tools => /Users/oneliang/AgentProjects/aura/modules/tools
)
