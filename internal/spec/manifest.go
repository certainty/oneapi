package spec

import (
	"errors"
	"gopkg.in/yaml.v3"
	"os"
)

type ServerConfig struct {
	Port          *int    `yaml:"port"`
	HealthCheck   *string `yaml:"healthCheck"`
	APIDocsPrefix *string `yaml:"apiDocsPrefix"`
	APIDocsUIPath *string `yaml:"apiDocsUIPath"`
}

type AuthConfig struct {
	BearerToken *struct {
		Token string `yaml:"token"`
	} `yaml:"bearer_token,omitempty"`
}

type FieldDef struct {
	Type     string   `yaml:"type"`
	Required bool     `yaml:"required"`
	Variants []string `yaml:"variants,omitempty"`
}

// EntityDef represents an entity definition in the manifest
type EntityDef struct {
	Fields map[string]FieldDef `yaml:"fields"`
}

type Manifest struct {
	Prefix   *string              `yaml:"prefix"`
	Server   *ServerConfig        `yaml:"server"`
	Auth     *AuthConfig          `yaml:"auth,omitempty"`
	Entities map[string]EntityDef `yaml:"entities"`
}

func LoadManifest(path string) (*Manifest, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var manifest Manifest

	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&manifest); err != nil {
		return nil, err
	}

	if manifest.Entities == nil {
		return nil, errors.New("manifest must have at least one entity")
	}

	return &manifest, nil
}
