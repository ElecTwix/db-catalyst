// Package cli provides the command-line interface logic for db-catalyst.
package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
)

// Options holds the configuration derived from command-line arguments.
type Options struct {
	ConfigPath   string
	Out          string
	DryRun       bool
	ListQueries  bool
	StrictConfig bool
	Verbose      bool
	NoJSONTags   bool
	EmitJSONTags bool
	Args         []string
}

// Parse processes command-line arguments and returns the options.
func Parse(args []string) (Options, error) {
	const defaultConfig = "db-catalyst.toml"

	opts := Options{
		ConfigPath: defaultConfig,
	}

	fs := flag.NewFlagSet("db-catalyst", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	fs.StringVar(&opts.ConfigPath, "config", opts.ConfigPath, "Path to configuration file")
	fs.StringVar(&opts.ConfigPath, "c", opts.ConfigPath, "Path to configuration file")
	fs.StringVar(&opts.Out, "out", "", "Override output directory; relative paths are resolved against the config directory")
	fs.BoolVar(&opts.DryRun, "dry-run", false, "Generate code without writing files")
	fs.BoolVar(&opts.ListQueries, "list-queries", false, "List configured queries without generating code")
	fs.BoolVar(&opts.StrictConfig, "strict-config", false, "Treat configuration warnings as errors")
	fs.BoolVar(&opts.Verbose, "verbose", false, "Enable verbose logging")
	fs.BoolVar(&opts.Verbose, "v", false, "Enable verbose logging")
	fs.BoolVar(&opts.NoJSONTags, "no-json-tags", false, "Disable JSON tags in generated code")

	if len(args) == 0 {
		usage := Usage(fs)
		return Options{}, fmt.Errorf("%w\n\n%s", flag.ErrHelp, usage)
	}

	if err := fs.Parse(args); err != nil {
		usage := Usage(fs)
		if errors.Is(err, flag.ErrHelp) {
			return Options{}, fmt.Errorf("%w\n\n%s", err, usage)
		}
		return Options{}, fmt.Errorf("%w\n\n%s", err, usage)
	}

	opts.Args = fs.Args()
	return opts, nil
}

// Usage returns the usage string for the command-line interface.
func Usage(fs *flag.FlagSet) string {
	if fs == nil {
		return ""
	}
	var buf strings.Builder
	_, _ = fmt.Fprintf(&buf, "Usage of %s:\n", fs.Name())
	out := fs.Output()
	fs.SetOutput(&buf)
	fs.PrintDefaults()
	fs.SetOutput(out)
	return buf.String()
}
