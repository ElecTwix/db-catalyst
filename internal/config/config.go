// Package config loads and validates the db-catalyst configuration.
package config

import (
	"errors"
	"fmt"
	"go/token"
	"os"
	"path/filepath"
	"slices"
	"strings"

	toml "github.com/pelletier/go-toml/v2"

	"github.com/electwix/db-catalyst/internal/fileset"
	"github.com/electwix/db-catalyst/internal/logging"
)

// Driver identifies the SQLite driver implementation to target.
type Driver string

const (
	// DriverModernC targets modernc.org/sqlite.
	DriverModernC Driver = "modernc"
	// DriverMattN targets github.com/mattn/go-sqlite3.
	DriverMattN Driver = "mattn"
)

var validDrivers = map[Driver]struct{}{
	DriverModernC: {},
	DriverMattN:   {},
}

// Language identifies the target programming language for code generation.
type Language string

const (
	// LanguageGo generates Go code.
	LanguageGo Language = "go"
	// LanguageRust generates Rust code.
	LanguageRust Language = "rust"
	// LanguageTypeScript generates TypeScript code.
	LanguageTypeScript Language = "typescript"
)

var validLanguages = map[Language]struct{}{
	LanguageGo:         {},
	LanguageRust:       {},
	LanguageTypeScript: {},
}

// Database identifies the target database dialect.
type Database string

const (
	// DatabaseSQLite targets SQLite.
	DatabaseSQLite Database = "sqlite"
	// DatabasePostgreSQL targets PostgreSQL.
	DatabasePostgreSQL Database = "postgresql"
	// DatabaseMySQL targets MySQL.
	DatabaseMySQL Database = "mysql"
)

var validDatabases = map[Database]struct{}{
	DatabaseSQLite:     {},
	DatabasePostgreSQL: {},
	DatabaseMySQL:      {},
}

// CustomTypeMapping defines how a custom type maps to SQLite and Go types.
type CustomTypeMapping struct {
	CustomType string `toml:"custom_type"`
	SQLiteType string `toml:"sqlite_type"`
	GoType     string `toml:"go_type"`
	GoImport   string `toml:"go_import"`
	GoPackage  string `toml:"go_package"`
	Pointer    bool   `toml:"pointer"`
}

// CustomTypesConfig captures custom type mappings.
type CustomTypesConfig struct {
	Mappings []CustomTypeMapping `toml:"mapping"`
}

// DatabaseTypeOverride defines db_type to go_type mappings (sqlc compatibility).
type DatabaseTypeOverride struct {
	DatabaseType string `toml:"db_type"`
	GoType       string `toml:"go_type"`
}

// GoTypeDetails captures complex go_type configuration (sqlc compatibility).
// It can unmarshal from either a string or a map.
type GoTypeDetails struct {
	Import  string `toml:"import"`
	Package string `toml:"package"`
	Type    string `toml:"type"`
	Pointer bool   `toml:"pointer"`
}

// UnmarshalTOML implements custom TOML unmarshaling for GoTypeDetails.
// This handles both formats:
//   - Simple: go_type = "string"
//   - Complex: go_type = { import = "...", package = "...", type = "..." }
func (g *GoTypeDetails) UnmarshalTOML(data any) error {
	switch v := data.(type) {
	case string:
		// Simple format: go_type = "string"
		g.Type = v
		return nil
	case map[string]any:
		// Complex format: go_type = { import = "...", ... }
		if typ, ok := v["type"].(string); ok {
			g.Type = typ
		}
		if imp, ok := v["import"].(string); ok {
			g.Import = imp
		}
		if pkg, ok := v["package"].(string); ok {
			g.Package = pkg
		}
		if ptr, ok := v["pointer"].(bool); ok {
			g.Pointer = ptr
		}
		return nil
	default:
		return fmt.Errorf("go_type must be a string or a map, got %T", data)
	}
}

// rawColumnOverride is used for TOML unmarshaling before conversion.
type rawColumnOverride struct {
	Column string `toml:"column"`
	GoType any    `toml:"go_type"`
}

// ColumnOverride defines column-specific type overrides (sqlc compatibility).
// The go_type field can be either a simple string or a complex object.
type ColumnOverride struct {
	Column string        `toml:"column"`
	GoType GoTypeDetails `toml:"go_type"`
}

// GenerationOptions captures additional generation options.
type GenerationOptions struct {
	EmitEmptySlices     bool   `toml:"emit_empty_slices"`
	EmitPreparedQueries bool   `toml:"emit_prepared_queries"`
	EmitJSONTags        bool   `toml:"emit_json_tags"`
	EmitPointersForNull bool   `toml:"emit_pointers_for_null"`
	SQLDialect          string `toml:"sql_dialect"`
}

// CacheConfig captures caching configuration for incremental builds.
type CacheConfig struct {
	Enabled bool   `toml:"enabled"`
	Dir     string `toml:"dir"`
}

// Cache is the normalized cache configuration forwarded to the pipeline.
type Cache struct {
	Enabled bool
	Dir     string
}

// JobPlan is the fully-resolved configuration used by downstream stages.
type JobPlan struct {
	Package             string
	Out                 string
	Language            Language
	Database            Database
	SQLiteDriver        Driver
	Schemas             []string
	Queries             []string
	CustomTypes         []CustomTypeMapping
	ColumnOverrides     map[string]ColumnOverride
	EmitJSONTags        bool
	EmitPointersForNull bool
	PreparedQueries     PreparedQueries
	SQLDialect          string
	Cache               Cache
}

// PreparedQueriesConfig captures optional prepared statement generation settings.
type PreparedQueriesConfig struct {
	Enabled         bool `toml:"enabled"`
	Metrics         bool `toml:"metrics"`
	ThreadSafe      bool `toml:"thread_safe"`
	EmitEmptySlices bool `toml:"emit_empty_slices"`
}

// PreparedQueries is the normalized configuration forwarded to the pipeline.
type PreparedQueries struct {
	Enabled         bool
	Metrics         bool
	ThreadSafe      bool
	EmitEmptySlices bool
}

// Config mirrors the expected db-catalyst TOML schema.
type Config struct {
	Package      string            `toml:"package"`
	Out          string            `toml:"out"`
	Language     Language          `toml:"language"`
	Database     Database          `toml:"database"`
	SQLiteDriver Driver            `toml:"sqlite_driver"`
	Schemas      []string          `toml:"schemas"`
	Queries      []string          `toml:"queries"`
	CustomTypes  CustomTypesConfig `toml:"custom_types"`
	// Overrides are parsed separately to handle flexible go_type formats
	Generation      GenerationOptions     `toml:"generation"`
	PreparedQueries PreparedQueriesConfig `toml:"prepared_queries"`
	Cache           CacheConfig           `toml:"cache"`
}

// LoadOptions tunes config loading behavior.
type LoadOptions struct {
	Strict   bool
	Resolver *fileset.Resolver
	// Logger receives warning messages. If nil, warnings are only added to Result.Warnings.
	Logger logging.Logger
}

// Result wraps a loaded job plan alongside any non-fatal warnings.
type Result struct {
	Plan     JobPlan
	Warnings []string
}

// Load reads, validates, and resolves a db-catalyst configuration file.
func Load(path string, opts LoadOptions) (Result, error) {
	var res Result

	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return res, fmt.Errorf("read %s: %w", path, err)
	}

	var cfg Config
	if err := toml.Unmarshal(data, &cfg); err != nil {
		return res, fmt.Errorf("%s: %w", path, err)
	}

	unknownKeys, err := collectUnknownKeys(data)
	if err != nil {
		return res, fmt.Errorf("%s: %w", path, err)
	}
	if len(unknownKeys) > 0 {
		slices.Sort(unknownKeys)
		message := fmt.Sprintf("%s: unknown configuration keys: %s", path, strings.Join(unknownKeys, ", "))
		if opts.Strict {
			return res, errors.New(message)
		}
		if opts.Logger != nil {
			opts.Logger.Warn("unknown configuration keys", "path", path, "keys", unknownKeys)
		}
		res.Warnings = append(res.Warnings, message)
	}

	preparedUnknown, err := collectUnknownPreparedKeys(data)
	if err != nil {
		return res, fmt.Errorf("%s: %w", path, err)
	}
	if len(preparedUnknown) > 0 {
		slices.Sort(preparedUnknown)
		message := fmt.Sprintf("%s: unknown prepared_queries keys: %s", path, strings.Join(preparedUnknown, ", "))
		if opts.Strict {
			return res, errors.New(message)
		}
		if opts.Logger != nil {
			opts.Logger.Warn("unknown prepared_queries keys", "path", path, "keys", preparedUnknown)
		}
		res.Warnings = append(res.Warnings, message)
	}

	if err := validatePackage(path, cfg.Package); err != nil {
		return res, err
	}

	out, err := resolveOut(path, cfg.Out)
	if err != nil {
		return res, err
	}

	driver, err := resolveDriver(path, cfg.SQLiteDriver)
	if err != nil {
		return res, err
	}

	lang, err := resolveLanguage(path, cfg.Language)
	if err != nil {
		return res, err
	}

	db, err := resolveDatabase(path, cfg.Database)
	if err != nil {
		return res, err
	}

	baseDir := filepath.Dir(path)

	var resolver fileset.Resolver
	if opts.Resolver != nil {
		resolver = *opts.Resolver
	} else {
		resolver, err = fileset.NewOSResolver(baseDir)
		if err != nil {
			return res, fmt.Errorf("%s: %w", path, err)
		}
	}

	schemas, err := resolvePatterns(resolver, "schemas", cfg.Schemas)
	if err != nil {
		return res, fmt.Errorf("%s: %w", path, err)
	}

	queries, err := resolvePatterns(resolver, "queries", cfg.Queries)
	if err != nil {
		return res, fmt.Errorf("%s: %w", path, err)
	}

	prepared := PreparedQueries{
		Enabled:         cfg.PreparedQueries.Enabled,
		Metrics:         cfg.PreparedQueries.Metrics,
		ThreadSafe:      cfg.PreparedQueries.ThreadSafe,
		EmitEmptySlices: cfg.PreparedQueries.EmitEmptySlices,
	}

	// Process custom types to extract import paths from go_type if needed
	customTypes := normalizeCustomTypes(cfg.CustomTypes.Mappings)

	// Process column overrides into a lookup map
	// First, unmarshal raw overrides to handle both string and object formats
	rawOverrides := unmarshalRawOverrides(data)
	convertedOverrides := convertRawOverrides(rawOverrides)
	columnOverrides := normalizeColumnOverrides(convertedOverrides)

	// Set default cache directory if enabled but not specified
	cacheDir := cfg.Cache.Dir
	if cfg.Cache.Enabled && cacheDir == "" {
		cacheDir = ".db-catalyst-cache"
	}

	res.Plan = JobPlan{
		Package:             cfg.Package,
		Out:                 out,
		Language:            lang,
		Database:            db,
		SQLiteDriver:        driver,
		Schemas:             schemas,
		Queries:             queries,
		CustomTypes:         customTypes,
		ColumnOverrides:     columnOverrides,
		EmitJSONTags:        cfg.Generation.EmitJSONTags,
		EmitPointersForNull: cfg.Generation.EmitPointersForNull,
		PreparedQueries:     prepared,
		SQLDialect:          cfg.Generation.SQLDialect,
		Cache: Cache{
			Enabled: cfg.Cache.Enabled,
			Dir:     cacheDir,
		},
	}

	return res, nil
}

func collectUnknownKeys(data []byte) ([]string, error) {
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	known := map[string]struct{}{
		"package":          {},
		"out":              {},
		"language":         {},
		"database":         {},
		"sqlite_driver":    {},
		"schemas":          {},
		"queries":          {},
		"custom_types":     {},
		"overrides":        {},
		"generation":       {},
		"prepared_queries": {},
		"cache":            {},
	}

	unknown := make([]string, 0)
	for key := range raw {
		if _, ok := known[key]; !ok {
			unknown = append(unknown, key)
		}
	}

	return unknown, nil
}

func collectUnknownPreparedKeys(data []byte) ([]string, error) {
	var raw map[string]any
	if err := toml.Unmarshal(data, &raw); err != nil {
		return nil, err
	}
	value, ok := raw["prepared_queries"]
	if !ok {
		return nil, nil
	}
	record, ok := value.(map[string]any)
	if !ok {
		return nil, nil
	}
	known := map[string]struct{}{
		"enabled":           {},
		"metrics":           {},
		"thread_safe":       {},
		"emit_empty_slices": {},
	}
	unknown := make([]string, 0)
	for key := range record {
		if _, ok := known[key]; !ok {
			unknown = append(unknown, key)
		}
	}
	return unknown, nil
}

func validatePackage(path, pkg string) error {
	if pkg == "" {
		return fmt.Errorf("%s: package is required", path)
	}
	if !token.IsIdentifier(pkg) || token.Lookup(pkg) != token.IDENT {
		return fmt.Errorf("%s: invalid package name %q", path, pkg)
	}
	return nil
}

func resolveOut(path, out string) (string, error) {
	if out == "" {
		return "", fmt.Errorf("%s: out is required", path)
	}
	if filepath.IsAbs(out) {
		return "", fmt.Errorf("%s: out must be a relative path", path)
	}

	cleaned := filepath.Clean(out)
	if cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("%s: out must not traverse upwards", path)
	}

	baseDir := filepath.Dir(path)
	return filepath.Join(baseDir, cleaned), nil
}

func resolveDriver(path string, driver Driver) (Driver, error) {
	if driver == "" {
		return DriverModernC, nil
	}
	if _, ok := validDrivers[driver]; !ok {
		return "", fmt.Errorf("%s: unsupported sqlite_driver %q", path, driver)
	}
	return driver, nil
}

func resolveLanguage(path string, lang Language) (Language, error) {
	if lang == "" {
		return LanguageGo, nil
	}
	if _, ok := validLanguages[lang]; !ok {
		return "", fmt.Errorf("%s: unsupported language %q", path, lang)
	}
	return lang, nil
}

func resolveDatabase(path string, db Database) (Database, error) {
	if db == "" {
		return DatabaseSQLite, nil
	}
	if _, ok := validDatabases[db]; !ok {
		return "", fmt.Errorf("%s: unsupported database %q", path, db)
	}
	return db, nil
}

func resolvePatterns(resolver fileset.Resolver, field string, patterns []string) ([]string, error) {
	paths, err := resolver.Resolve(patterns)
	if err != nil {
		switch {
		case errors.Is(err, fileset.ErrNoPatterns):
			return nil, fmt.Errorf("%s must include at least one pattern", field)
		default:
			var noMatchErr fileset.NoMatchError
			if errors.As(err, &noMatchErr) {
				return nil, fmt.Errorf("%s patterns matched no files: %s", field, strings.Join(noMatchErr.Patterns, ", "))
			}

			var patternErr fileset.PatternError
			if errors.As(err, &patternErr) {
				return nil, fmt.Errorf("%s: invalid glob pattern %q: %w", field, patternErr.Pattern, patternErr.Err)
			}

			return nil, fmt.Errorf("%s: %w", field, err)
		}
	}

	return paths, nil
}

// normalizeCustomTypes processes custom type mappings to extract import paths
// from go_type when go_import is not explicitly provided.
// For example, go_type="github.com/example/types.UserID" becomes:
//   - go_import="github.com/example/types"
//   - go_package="types"
//   - go_type="UserID"
func normalizeCustomTypes(mappings []CustomTypeMapping) []CustomTypeMapping {
	result := make([]CustomTypeMapping, len(mappings))
	for i, m := range mappings {
		result[i] = m

		// If go_import is already set, just ensure go_package is set
		if m.GoImport != "" {
			if m.GoPackage == "" {
				result[i].GoPackage = extractPackageName(m.GoImport)
			}
			continue
		}

		// Try to extract import from go_type
		if m.GoType != "" && strings.Contains(m.GoType, ".") {
			importPath, typeName := extractImportAndType(m.GoType)
			if importPath != "" {
				result[i].GoImport = importPath
				result[i].GoPackage = extractPackageName(importPath)
				result[i].GoType = typeName
			}
		}
	}
	return result
}

// extractImportAndType splits a fully qualified Go type into import path and type name.
// Example: "github.com/example/types.UserID" -> ("github.com/example/types", "UserID")
func extractImportAndType(qualifiedType string) (importPath, typeName string) {
	// Handle pointer types
	isPointer := strings.HasPrefix(qualifiedType, "*")
	if isPointer {
		qualifiedType = strings.TrimPrefix(qualifiedType, "*")
	}

	// Find the last dot to separate import path from type name
	lastDot := strings.LastIndex(qualifiedType, ".")
	if lastDot == -1 {
		return "", qualifiedType
	}

	importPath = qualifiedType[:lastDot]
	typeName = qualifiedType[lastDot+1:]

	// Re-add pointer if it was there
	if isPointer {
		typeName = "*" + typeName
	}

	return importPath, typeName
}

// extractPackageName extracts the package name from an import path.
// Example: "github.com/example/types" -> "types"
func extractPackageName(importPath string) string {
	parts := strings.Split(importPath, "/")
	return parts[len(parts)-1]
}

// normalizeColumnOverrides processes column overrides into a normalized lookup map.
// The key format is "table.column" (lowercase for case-insensitive lookup).
func normalizeColumnOverrides(overrides []ColumnOverride) map[string]ColumnOverride {
	result := make(map[string]ColumnOverride, len(overrides))
	for _, o := range overrides {
		if o.Column == "" {
			continue
		}
		// Normalize to lowercase for case-insensitive lookup
		key := strings.ToLower(o.Column)
		result[key] = o
	}
	return result
}

// unmarshalRawOverrides extracts raw column overrides from TOML data.
func unmarshalRawOverrides(data []byte) []rawColumnOverride {
	var raw struct {
		Overrides []rawColumnOverride `toml:"overrides"`
	}
	// We ignore errors here because the main unmarshal already succeeded
	// and we're just extracting the overrides field
	_ = toml.Unmarshal(data, &raw)
	return raw.Overrides
}

// convertRawOverrides converts raw TOML-parsed overrides to ColumnOverride structs.
func convertRawOverrides(raw []rawColumnOverride) []ColumnOverride {
	result := make([]ColumnOverride, 0, len(raw))
	for _, r := range raw {
		co := ColumnOverride{
			Column: r.Column,
		}

		switch v := r.GoType.(type) {
		case string:
			// Simple format: go_type = "string"
			co.GoType = GoTypeDetails{Type: v}
		case map[string]any:
			// Complex format: go_type = { import = "...", ... }
			if typ, ok := v["type"].(string); ok {
				co.GoType.Type = typ
			}
			if imp, ok := v["import"].(string); ok {
				co.GoType.Import = imp
			}
			if pkg, ok := v["package"].(string); ok {
				co.GoType.Package = pkg
			}
			if ptr, ok := v["pointer"].(bool); ok {
				co.GoType.Pointer = ptr
			}
		}

		result = append(result, co)
	}
	return result
}
