package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

var (
	version = "dev"
	date    = time.Now().Format("2006-01-02")
)

func main() {
	ctx := context.Background()
	if err := run(ctx, os.Args[1:]); err != nil {
		fmt.Fprintf(os.Stderr, "sqlc2dbcat: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	opts, posArgs := parseFlags(args)
	if opts.Help {
		printHelp()
		return nil
	}

	path := "."
	if len(posArgs) > 0 {
		path = posArgs[0]
	}

	return migrate(ctx, path, opts)
}

type options struct {
	Help        bool
	DryRun      bool
	OutputDir   string
	Overwrite   bool
	Validate    bool
	Interactive bool
	Path        string
}

func parseFlags(args []string) (options, []string) {
	fs := flag.NewFlagSet("sqlc2dbcat", flag.ContinueOnError)
	fs.Usage = func() {}

	var opts options
	fs.BoolVar(&opts.Help, "h", false, "show help")
	fs.BoolVar(&opts.Help, "help", false, "show help")
	fs.BoolVar(&opts.DryRun, "n", false, "dry run, don't write files")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "dry run, don't write files")
	fs.StringVar(&opts.OutputDir, "o", "", "output directory (default: same as input)")
	fs.BoolVar(&opts.Overwrite, "f", false, "overwrite existing db-catalyst.toml")
	fs.BoolVar(&opts.Validate, "validate", false, "only validate, don't migrate")
	fs.BoolVar(&opts.Interactive, "i", false, "interactive mode, ask before changes")

	if err := fs.Parse(args); err != nil {
		return options{}, nil
	}

	return opts, fs.Args()
}

func printHelp() {
	fmt.Fprintf(os.Stderr, `sqlc2dbcat %s (%s)
Migrate sqlc projects to db-catalyst

USAGE:
    sqlc2dbcat [OPTIONS] [PATH]

PATH:
    Directory containing sqlc.yaml (default: current directory)

OPTIONS:
    -h, --help              show this help
    -n, --dry-run           show what would be done without writing files
    -o, --output DIR        output directory (default: same as input)
    -f, --overwrite         overwrite existing db-catalyst.toml
    --validate              only validate compatibility, don't migrate
    -i, --interactive       ask before each change

EXAMPLES:
    # Migrate in current directory
    sqlc2dbcat

    # Dry run to see what would change
    sqlc2dbcat -n

    # Migrate specific project
    sqlc2dbcat ./mypackage

    # Only validate compatibility
    sqlc2dbcat --validate
`, version, date)
}

type sqlcConfig struct {
	Version   string     `yaml:"version"`
	SQL       []sqlcJob  `yaml:"sql"`
	Overrides []Override `yaml:"overrides"`
}

type sqlcJob struct {
	Engine     string  `yaml:"engine"`
	Schema     string  `yaml:"schema"`
	Queries    string  `yaml:"queries"`
	Gen        sqlcGen `yaml:"gen"`
	SQLPackage string  `yaml:"sql_package,omitempty"`
}

type sqlcGen struct {
	Go goGenConfig `yaml:"go"`
}

type goGenConfig struct {
	Package      string `yaml:"package"`
	Out          string `yaml:"out"`
	EmitJSONTags bool   `yaml:"emit_json_tags,omitempty"`
	EmitEnumZero bool   `yaml:"emit_enum_zero_method,omitempty"`
	EmitAll      bool   `yaml:"emit_all,omitempty"`
}

type Override struct {
	GoType any    `yaml:"go_type"`
	DBType string `yaml:"db_type,omitempty"`
	Column string `yaml:"column,omitempty"`
	Table  string `yaml:"table,omitempty"`
}

func migrate(ctx context.Context, path string, opts options) error {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	sqlcPath := filepath.Join(absPath, "sqlc.yaml")
	if _, err := os.Stat(sqlcPath); os.IsNotExist(err) {
		return fmt.Errorf("sqlc.yaml not found in %s", absPath)
	}

	data, err := os.ReadFile(sqlcPath)
	if err != nil {
		return fmt.Errorf("read sqlc.yaml: %w", err)
	}

	var sqlcCfg sqlcConfig
	if err := yaml.Unmarshal(data, &sqlcCfg); err != nil {
		return fmt.Errorf("parse sqlc.yaml: %w", err)
	}

	if len(sqlcCfg.SQL) == 0 {
		return fmt.Errorf("no sql jobs found in sqlc.yaml")
	}

	issues := analyzeMigration(sqlcCfg)
	for _, issue := range issues {
		switch issue.Severity {
		case IssueError:
			fmt.Printf("ERROR: %s\n", issue.Message)
		case IssueWarning:
			fmt.Printf("WARNING: %s\n", issue.Message)
		case IssueInfo:
			fmt.Printf("INFO: %s\n", issue.Message)
		}
	}

	hasErrors := false
	for _, issue := range issues {
		if issue.Severity == IssueError {
			hasErrors = true
		}
	}

	if hasErrors {
		fmt.Println("\nMigration has errors. Fix these before proceeding.")
		if opts.Validate {
			return nil
		}
	}

	if opts.Validate {
		fmt.Println("\nValidation complete. See above for issues.")
		return nil
	}

	dbcatCfg := convertConfig(sqlcCfg)

	outputDir := absPath
	if opts.OutputDir != "" {
		outputDir, err = filepath.Abs(opts.OutputDir)
		if err != nil {
			return fmt.Errorf("invalid output dir: %w", err)
		}
	}

	dbcatPath := filepath.Join(outputDir, "db-catalyst.toml")
	exists := false
	if _, err := os.Stat(dbcatPath); err == nil {
		exists = true
	}

	if exists && !opts.Overwrite && !opts.DryRun {
		return fmt.Errorf("db-catalyst.toml already exists in %s (use -f to overwrite)", outputDir)
	}

	var buf bytes.Buffer
	encodeDBConfig(&buf, dbcatCfg)

	fmt.Println("\nGenerated db-catalyst.toml:")
	fmt.Println("---")
	fmt.Println(buf.String())
	fmt.Println("---")

	if opts.DryRun {
		fmt.Println("\n(Dry run - no files written)")
		return nil
	}

	if opts.Interactive {
		fmt.Printf("\nWrite db-catalyst.toml to %s? [y/N] ", outputDir)
		var response string
		fmt.Scanln(&response)
		if strings.ToLower(response) != "y" {
			fmt.Println("Aborted.")
			return nil
		}
	}

	if err := os.WriteFile(dbcatPath, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("write db-catalyst.toml: %w", err)
	}

	fmt.Printf("\nSuccessfully created %s\n", dbcatPath)
	fmt.Println("\nNext steps:")
	fmt.Println("  1. Review db-catalyst.toml configuration")
	fmt.Println("  2. Run: db-catalyst generate")
	fmt.Println("  3. Update import paths in your code")
	fmt.Println("  4. Test compilation")

	return nil
}

type IssueSeverity int

const (
	IssueInfo IssueSeverity = iota
	IssueWarning
	IssueError
)

type Issue struct {
	Severity IssueSeverity
	Message  string
}

func analyzeMigration(sqlcCfg sqlcConfig) []Issue {
	var issues []Issue

	for _, job := range sqlcCfg.SQL {
		if strings.ToLower(job.Engine) != "sqlite" {
			issues = append(issues, Issue{
				Severity: IssueError,
				Message:  fmt.Sprintf("engine=%q is not SQLite. db-catalyst only supports SQLite.", job.Engine),
			})
		}

		if job.SQLPackage != "" && job.SQLPackage != "database/sql" {
			issues = append(issues, Issue{
				Severity: IssueWarning,
				Message:  fmt.Sprintf("sql_package=%q will be ignored. db-catalyst generates code compatible with database/sql.", job.SQLPackage),
			})
		}
	}

	if len(sqlcCfg.Overrides) > 0 {
		for _, ov := range sqlcCfg.Overrides {
			if ov.Column != "" {
				issues = append(issues, Issue{
					Severity: IssueWarning,
					Message:  fmt.Sprintf("column-specific override %q will be converted to type-wide mapping", ov.Column),
				})
			}
		}
	}

	return issues
}

type dbcatConfigOutput struct {
	Package      string            `toml:"package"`
	Out          string            `toml:"out"`
	SQLiteDriver string            `toml:"sqlite_driver,omitempty"`
	Schemas      []string          `toml:"schemas"`
	Queries      []string          `toml:"queries"`
	CustomTypes  customTypesOutput `toml:"custom_types,omitempty"`
}

type customTypesOutput struct {
	Mappings []customTypeMappingOutput `toml:"mapping,omitempty"`
}

type customTypeMappingOutput struct {
	CustomType string `toml:"custom_type"`
	SQLiteType string `toml:"sqlite_type"`
	GoType     string `toml:"go_type"`
	Pointer    bool   `toml:"pointer,omitempty"`
}

func convertConfig(sqlcCfg sqlcConfig) dbcatConfigOutput {
	job := sqlcCfg.SQL[0]

	schemaPattern := "*.sql"
	if job.Schema != "" {
		schemaPattern = filepath.Join(job.Schema, "*.sql")
	}
	queryPattern := "*.sql"
	if job.Queries != "" {
		queryPattern = filepath.Join(job.Queries, "*.sql")
	}

	cfg := dbcatConfigOutput{
		Package: job.Gen.Go.Package,
		Out:     job.Gen.Go.Out,
		Schemas: []string{schemaPattern},
		Queries: []string{queryPattern},
	}

	if len(sqlcCfg.Overrides) > 0 {
		cfg.CustomTypes = convertOverrides(sqlcCfg.Overrides)
	}

	return cfg
}

func convertOverrides(overrides []Override) customTypesOutput {
	var mappings []customTypeMappingOutput
	seenTypes := make(map[string]bool)

	for _, ov := range overrides {
		goType, pointer := parseGoType(ov.GoType)

		if ov.DBType != "" && !seenTypes[ov.DBType] {
			seenTypes[ov.DBType] = true
			mappings = append(mappings, customTypeMappingOutput{
				CustomType: goTypeToCustomName(goType),
				SQLiteType: ov.DBType,
				GoType:     goType,
				Pointer:    pointer,
			})
		}
	}

	return customTypesOutput{Mappings: mappings}
}

func parseGoType(goType any) (string, bool) {
	switch t := goType.(type) {
	case string:
		return t, false
	case map[string]any:
		if ptr, ok := t["pointer"].(bool); ok {
			typ, _ := t["type"].(string)
			return typ, ptr
		}
		typ, _ := t["type"].(string)
		return typ, false
	}
	return "", false
}

func goTypeToCustomName(goType string) string {
	parts := strings.Split(goType, ".")
	if len(parts) > 1 {
		return parts[len(parts)-1]
	}
	return goType
}

func encodeDBConfig(buf *bytes.Buffer, cfg dbcatConfigOutput) {
	buf.WriteString("# Generated by sqlc2dbcat\n")
	buf.WriteString("# Review and adjust as needed\n\n")
	buf.WriteString("package = \"" + cfg.Package + "\"\n")
	buf.WriteString("out = \"" + cfg.Out + "\"\n")
	buf.WriteString("sqlite_driver = \"modernc\"\n")
	buf.WriteString("\n")

	buf.WriteString("schemas = [\n")
	for _, s := range cfg.Schemas {
		buf.WriteString("    \"" + s + "\",\n")
	}
	buf.WriteString("]\n\n")

	buf.WriteString("queries = [\n")
	for _, q := range cfg.Queries {
		buf.WriteString("    \"" + q + "\",\n")
	}
	buf.WriteString("]\n")

	if len(cfg.CustomTypes.Mappings) > 0 {
		buf.WriteString("\n[custom_types]\n")
		for _, m := range cfg.CustomTypes.Mappings {
			buf.WriteString("[[custom_types.mapping]]\n")
			buf.WriteString("custom_type = \"" + m.CustomType + "\"\n")
			buf.WriteString("sqlite_type = \"" + m.SQLiteType + "\"\n")
			buf.WriteString("go_type = \"" + m.GoType + "\"\n")
			if m.Pointer {
				buf.WriteString("pointer = true\n")
			}
			buf.WriteString("\n")
		}
	}
}
