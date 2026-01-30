// Package sqlcconfig parses sqlc configuration files.
package sqlcconfig

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	schemamodel "github.com/electwix/db-catalyst/internal/schema/model"
	schemaparser "github.com/electwix/db-catalyst/internal/schema/parser"
	schematokenizer "github.com/electwix/db-catalyst/internal/schema/tokenizer"
)

// Config represents the subset of sqlc configuration required for override migration.
type Config struct {
	Version   string     `yaml:"version"`
	SQL       []SQLBlock `yaml:"sql"`
	Overrides []Override `yaml:"overrides"`

	baseDir        string
	columnTypes    map[columnKey]string
	schemaWarnings []string
}

// SQLBlock captures the sqlc sql entry containing schema paths.
type SQLBlock struct {
	Engine   string   `yaml:"engine"`
	Schema   []string `yaml:"schema"`
	Queries  []string `yaml:"queries"`
	Database struct {
		Schema string `yaml:"schema"`
	} `yaml:"database"`
}

// Override mirrors sqlc override entries for db_type or column overrides.
type Override struct {
	DBType string       `yaml:"db_type"`
	Column ColumnTarget `yaml:"column"`
	GoType GoType       `yaml:"go_type"`
}

// ColumnTarget identifies a table column referenced by a sqlc override.
type ColumnTarget struct {
	Schema string
	Table  string
	Name   string
	set    bool
}

// ColumnRef is used to look up column metadata.
type ColumnRef struct {
	Schema string
	Table  string
	Column string
}

// GoType captures sqlc go_type override definitions.
type GoType struct {
	asString string

	Import  string `yaml:"import"`
	Package string `yaml:"package"`
	Type    string `yaml:"type"`
	Pointer bool   `yaml:"pointer"`

	mapped bool
}

// GoTypeInfo represents a normalized Go type description.
type GoTypeInfo struct {
	ImportPath  string
	PackageName string
	TypeName    string
	Pointer     bool
}

// Load reads and validates a sqlc configuration file.
func Load(path string) (Config, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return Config{}, fmt.Errorf("read %s: %w", path, err)
	}

	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)

	var cfg Config
	if err := dec.Decode(&cfg); err != nil {
		return Config{}, fmt.Errorf("parse %s: %w", path, err)
	}
	var extra map[string]any
	if err := dec.Decode(&extra); err != nil {
		if !errors.Is(err, io.EOF) {
			return Config{}, fmt.Errorf("parse %s: %w", path, err)
		}
	} else if len(extra) != 0 {
		return Config{}, fmt.Errorf("%s: unexpected additional YAML document", path)
	}
	if cfg.Version == "" {
		return Config{}, fmt.Errorf("%s: version is required", path)
	}
	if cfg.Version != "2" && cfg.Version != "2.0" {
		return Config{}, fmt.Errorf("%s: unsupported version %q (expected 2)", path, cfg.Version)
	}

	cfg.baseDir = filepath.Dir(path)
	if err := cfg.populateColumnTypes(); err != nil {
		return Config{}, err
	}

	return cfg, nil
}

// SchemaWarnings returns a copy of schema parsing warnings encountered during load.
func (c Config) SchemaWarnings() []string {
	out := make([]string, len(c.schemaWarnings))
	copy(out, c.schemaWarnings)
	return out
}

// ColumnType returns the SQLite type for the referenced column if known.
func (c Config) ColumnType(ref ColumnRef) (string, bool) {
	if c.columnTypes == nil {
		return "", false
	}
	key := makeColumnKey(ref.Schema, ref.Table, ref.Column)
	typ, ok := c.columnTypes[key]
	return typ, ok
}

func (c *Config) populateColumnTypes() error {
	paths, err := c.schemaPaths()
	if err != nil {
		return err
	}
	index, warnings, err := loadColumnTypes(paths)
	if err != nil {
		return err
	}
	c.columnTypes = index
	c.schemaWarnings = warnings
	return nil
}

func (c Config) schemaPaths() ([]string, error) {
	uniq := make(map[string]struct{})
	for _, block := range c.SQL {
		for _, pattern := range block.Schema {
			if pattern == "" {
				continue
			}
			abs := pattern
			if !filepath.IsAbs(pattern) {
				abs = filepath.Join(c.baseDir, pattern)
			}
			matches, err := filepath.Glob(abs)
			if err != nil {
				return nil, fmt.Errorf("glob %s: %w", pattern, err)
			}
			if len(matches) == 0 {
				return nil, fmt.Errorf("schema pattern %s matched no files", pattern)
			}
			for _, m := range matches {
				info, err := os.Stat(m)
				if err != nil {
					return nil, fmt.Errorf("stat %s: %w", m, err)
				}
				if info.IsDir() {
					return nil, fmt.Errorf("schema path %s is a directory", m)
				}
				norm := filepath.Clean(m)
				uniq[norm] = struct{}{}
			}
		}
	}
	if len(uniq) == 0 {
		return nil, errors.New("no schema files resolved from sqlc config")
	}
	paths := make([]string, 0, len(uniq))
	for p := range uniq {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths, nil
}

func loadColumnTypes(paths []string) (map[columnKey]string, []string, error) {
	index := make(map[columnKey]string)
	warnings := make([]string, 0)
	catalog := schemamodel.NewCatalog()

	for _, path := range paths {
		contents, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return nil, nil, fmt.Errorf("read schema %s: %w", path, err)
		}
		tokens, err := schematokenizer.Scan(path, contents, true)
		if err != nil {
			return nil, nil, fmt.Errorf("scan schema %s: %w", path, err)
		}
		partial, diags, err := schemaparser.Parse(path, tokens)
		for _, diag := range diags {
			warnings = append(warnings, fmt.Sprintf("%s:%d:%d: %s", diag.Path, diag.Line, diag.Column, diag.Message))
		}
		if err != nil {
			return nil, nil, fmt.Errorf("parse schema %s: %w", path, err)
		}
		if err := mergeCatalog(catalog, partial, &warnings); err != nil {
			return nil, nil, err
		}
	}

	for _, table := range catalog.Tables {
		tableName := canonicalIdentifier(table.Name)
		for _, col := range table.Columns {
			key := makeColumnKey("", tableName, col.Name)
			index[key] = sanitizeSQLiteType(col.Type)
		}
	}

	return index, warnings, nil
}

func mergeCatalog(dest, src *schemamodel.Catalog, warnings *[]string) error {
	if dest == nil || src == nil {
		return errors.New("nil catalog provided")
	}
	if dest.Tables == nil {
		dest.Tables = make(map[string]*schemamodel.Table)
	}
	for name, table := range src.Tables {
		canon := canonicalIdentifier(name)
		if _, exists := dest.Tables[canon]; exists {
			*warnings = append(*warnings, fmt.Sprintf("duplicate table %q", name))
			continue
		}
		dest.Tables[canon] = table
	}
	if dest.Views == nil {
		dest.Views = make(map[string]*schemamodel.View)
	}
	for name, view := range src.Views {
		canon := canonicalIdentifier(name)
		if _, exists := dest.Views[canon]; exists {
			*warnings = append(*warnings, fmt.Sprintf("duplicate view %q", name))
			continue
		}
		dest.Views[canon] = view
	}
	return nil
}

func sanitizeSQLiteType(t string) string {
	trimmed := strings.TrimSpace(t)
	if trimmed == "" {
		return ""
	}
	return strings.ToUpper(trimmed)
}

func canonicalIdentifier(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

type columnKey string

func makeColumnKey(schema, table, column string) columnKey {
	parts := []string{
		canonicalIdentifier(schema),
		canonicalIdentifier(table),
		canonicalIdentifier(column),
	}
	return columnKey(strings.Join(parts, "."))
}

// UnmarshalYAML supports both string and mapping forms for column targets.
func (c *ColumnTarget) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || value.Tag == "!!null" {
		return nil
	}
	switch value.Kind {
	case yaml.ScalarNode:
		text := strings.TrimSpace(value.Value)
		if text == "" {
			return errors.New("column override cannot be empty")
		}
		parts := strings.Split(text, ".")
		switch len(parts) {
		case 2:
			c.Table = strings.TrimSpace(parts[0])
			c.Name = strings.TrimSpace(parts[1])
		case 3:
			c.Schema = strings.TrimSpace(parts[0])
			c.Table = strings.TrimSpace(parts[1])
			c.Name = strings.TrimSpace(parts[2])
		default:
			return fmt.Errorf("invalid column reference %q", text)
		}
		c.set = true
		return nil
	case yaml.MappingNode:
		var tmp struct {
			Schema string `yaml:"schema"`
			Table  string `yaml:"table"`
			Column string `yaml:"column"`
		}
		if err := value.Decode(&tmp); err != nil {
			return err
		}
		if tmp.Table == "" || tmp.Column == "" {
			return errors.New("column override requires table and column")
		}
		c.Schema = tmp.Schema
		c.Table = tmp.Table
		c.Name = tmp.Column
		c.set = true
		return nil
	default:
		return fmt.Errorf("unsupported column override YAML kind: %d", value.Kind)
	}
}

// IsZero reports whether the column target is unset.
func (c ColumnTarget) IsZero() bool {
	return !c.set
}

// UnmarshalYAML captures string or mapping go_type definitions.
func (g *GoType) UnmarshalYAML(value *yaml.Node) error {
	if value == nil || value.Tag == "!!null" {
		return nil
	}
	switch value.Kind {
	case yaml.ScalarNode:
		g.asString = strings.TrimSpace(value.Value)
		g.mapped = false
		return nil
	case yaml.MappingNode:
		var tmp struct {
			Import  string `yaml:"import"`
			Package string `yaml:"package"`
			Type    string `yaml:"type"`
			Pointer bool   `yaml:"pointer"`
		}
		if err := value.Decode(&tmp); err != nil {
			return err
		}
		g.Import = tmp.Import
		g.Package = tmp.Package
		g.Type = tmp.Type
		g.Pointer = tmp.Pointer
		g.mapped = true
		g.asString = ""
		return nil
	default:
		return fmt.Errorf("unsupported go_type YAML kind: %d", value.Kind)
	}
}

// Normalize returns the normalized Go type information for this override.
func (g GoType) Normalize() (GoTypeInfo, error) {
	if g.mapped {
		if g.Type == "" {
			return GoTypeInfo{}, errors.New("go_type: type field is required")
		}
		info := GoTypeInfo{
			ImportPath:  strings.TrimSpace(g.Import),
			PackageName: strings.TrimSpace(g.Package),
			TypeName:    strings.TrimSpace(g.Type),
			Pointer:     g.Pointer,
		}
		if info.ImportPath != "" && info.PackageName == "" {
			info.PackageName = path.Base(info.ImportPath)
		}
		return info, nil
	}
	expr := strings.TrimSpace(g.asString)
	if expr == "" {
		return GoTypeInfo{}, errors.New("go_type must be provided")
	}
	pointer := false
	for strings.HasPrefix(expr, "*") {
		pointer = true
		expr = strings.TrimSpace(expr[1:])
	}
	info := GoTypeInfo{Pointer: pointer}
	if strings.Contains(expr, ".") {
		idx := strings.LastIndex(expr, ".")
		info.ImportPath = strings.TrimSpace(expr[:idx])
		info.TypeName = strings.TrimSpace(expr[idx+1:])
		info.PackageName = path.Base(info.ImportPath)
	} else {
		info.TypeName = expr
	}
	if info.TypeName == "" {
		return GoTypeInfo{}, errors.New("invalid go_type expression")
	}
	return info, nil
}
