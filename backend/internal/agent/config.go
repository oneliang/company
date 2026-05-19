package agent

import (
	"os"

	"gopkg.in/yaml.v3"
)

// Config holds configuration for agent runtime.
type Config struct {
	LLM struct {
		Provider string `yaml:"provider"`
		Model    string `yaml:"model"`
		APIKey   string `yaml:"api_key"`
		BaseURL  string `yaml:"base_url"`
	} `yaml:"llm"`
	Agent struct {
		MaxSteps    int     `yaml:"max_steps"`
		Temperature float64 `yaml:"temperature"`
	} `yaml:"agent"`
}

// LoadConfig reads agent configuration from YAML file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	// Expand environment variables for API key
	cfg.LLM.APIKey = os.ExpandEnv(cfg.LLM.APIKey)
	return &cfg, nil
}