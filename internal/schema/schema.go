package schema

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

const Example = `network:
  column: cidr

metadata:
  database_type: custom-ip-intel
  description: Custom IP intelligence dataset
  ip_version: 6
  record_size: 28
  build_epoch: 1
  include_reserved_networks: true

fields:
  asn:
    column: asn
    type: uint32
  is_vpn:
    column: is_vpn
    type: bool
  geo.country.iso_code:
    column: country
    type: string
  tags:
    column: tags
    type: string_array
    split: "|"
`

type Schema struct {
	Network  NetworkConfig          `yaml:"network"`
	Metadata MetadataConfig         `yaml:"metadata"`
	Fields   map[string]FieldConfig `yaml:"fields"`
}

type NetworkConfig struct {
	Column string `yaml:"column"`
}

type MetadataConfig struct {
	DatabaseType            string `yaml:"database_type"`
	Description             string `yaml:"description"`
	IPVersion               int    `yaml:"ip_version"`
	RecordSize              int    `yaml:"record_size"`
	BuildEpoch              int64  `yaml:"build_epoch"`
	IncludeReservedNetworks bool   `yaml:"include_reserved_networks"`
}

type FieldConfig struct {
	Column   string `yaml:"column"`
	Type     string `yaml:"type"`
	Split    string `yaml:"split"`
	Required bool   `yaml:"required"`
}

func Load(path string) (*Schema, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read schema %s: %w", path, err)
	}
	var s Schema
	if err := yaml.Unmarshal(b, &s); err != nil {
		return nil, fmt.Errorf("parse schema %s: %w", path, err)
	}
	if err := s.Validate(); err != nil {
		return nil, err
	}
	return &s, nil
}

func (s *Schema) Validate() error {
	if strings.TrimSpace(s.Network.Column) == "" {
		return fmt.Errorf("schema network.column is required")
	}
	if len(s.Fields) == 0 {
		return fmt.Errorf("schema fields are required")
	}
	if s.Metadata.DatabaseType == "" {
		s.Metadata.DatabaseType = "mmdbbridge"
	}
	if s.Metadata.Description == "" {
		s.Metadata.Description = "MMDBridge custom dataset"
	}
	if s.Metadata.IPVersion == 0 {
		s.Metadata.IPVersion = 6
	}
	if s.Metadata.RecordSize == 0 {
		s.Metadata.RecordSize = 28
	}
	if s.Metadata.BuildEpoch == 0 {
		s.Metadata.BuildEpoch = 1
	}
	for path, field := range s.Fields {
		if strings.TrimSpace(path) == "" {
			return fmt.Errorf("field path cannot be empty")
		}
		if strings.TrimSpace(field.Column) == "" {
			return fmt.Errorf("field %s: column is required", path)
		}
		if !validType(field.Type) {
			return fmt.Errorf("field %s: unsupported type %q", path, field.Type)
		}
		if strings.Contains(path, "..") || strings.HasPrefix(path, ".") || strings.HasSuffix(path, ".") {
			return fmt.Errorf("field %s: invalid dotted path", path)
		}
	}
	return nil
}

func (s *Schema) OrderedFieldPaths() []string {
	paths := make([]string, 0, len(s.Fields))
	for path := range s.Fields {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func (s *Schema) Header() []string {
	header := []string{s.Network.Column}
	for _, path := range s.OrderedFieldPaths() {
		header = append(header, s.Fields[path].Column)
	}
	return header
}

func validType(t string) bool {
	switch t {
	case "string", "bool", "int32", "uint16", "uint32", "uint64", "float32", "float64", "string_array":
		return true
	default:
		return false
	}
}
