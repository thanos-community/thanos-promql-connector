package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v2"
)

// queryBackendConfig represents the configuration for a backend query target.
type queryBackendConfig struct {
	QueryTargetURL string `yaml:"query_target_url"`
	Auth           struct {
		CredentialsFile string   `yaml:"credentials_file"`
		Scopes          []string `yaml:"scopes"`
	} `yaml:"auth"`
}

// loadQueryConfig parses the given YAML file into a queryTargetConfig.
func loadQueryConfig(fileName string) (*queryBackendConfig, error) {
	if fileName == "" {
		return nil, fmt.Errorf("query.config-file flag must be set")
	}
	content, err := os.ReadFile(fileName)
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %s", err)
	}
	cfg := &queryBackendConfig{}
	err = yaml.UnmarshalStrict([]byte(content), cfg)
	if err != nil {
		return nil, fmt.Errorf("error parsing YAML file: %s", err)
	}

	if cfg.QueryTargetURL == "" {
		return nil, fmt.Errorf("query_target_url needs to be set")
	}
	return cfg, nil
}
