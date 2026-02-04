// Package diagnostics provides rich diagnostic information for db-catalyst.
package diagnostics

import (
	"fmt"
	"io"
	"strings"
)

// Formatter formats diagnostics for display.
type Formatter struct {
	// ShowContext controls whether to display code snippets.
	ShowContext bool
	// ShowSuggestions controls whether to display suggestions.
	ShowSuggestions bool
	// ShowNotes controls whether to display notes.
	ShowNotes bool
	// ShowRelated controls whether to display related information.
	ShowRelated bool
	// ShowSource controls whether to display the source component.
	ShowSource bool
	// ShowCode controls whether to display error codes.
	ShowCode bool
	// ShowCodeDescription controls whether to display error code descriptions.
	ShowCodeDescription bool
	// Colorize controls whether to use ANSI color codes.
	Colorize bool
	// ContextLines is the number of context lines to show around the error.
	ContextLines int
}

// NewFormatter creates a new formatter with default settings.
func NewFormatter() *Formatter {
	return &Formatter{
		ShowContext:         true,
		ShowSuggestions:     true,
		ShowNotes:           true,
		ShowRelated:         true,
		ShowSource:          false,
		ShowCode:            true,
		ShowCodeDescription: false,
		Colorize:            false,
		ContextLines:        2,
	}
}

// NewVerboseFormatter creates a formatter with all options enabled.
func NewVerboseFormatter() *Formatter {
	return &Formatter{
		ShowContext:         true,
		ShowSuggestions:     true,
		ShowNotes:           true,
		ShowRelated:         true,
		ShowSource:          true,
		ShowCode:            true,
		ShowCodeDescription: true,
		Colorize:            true,
		ContextLines:        3,
	}
}

// NewSimpleFormatter creates a formatter with minimal output.
func NewSimpleFormatter() *Formatter {
	return &Formatter{
		ShowContext:         false,
		ShowSuggestions:     false,
		ShowNotes:           false,
		ShowRelated:         false,
		ShowSource:          false,
		ShowCode:            false,
		ShowCodeDescription: false,
		Colorize:            false,
		ContextLines:        0,
	}
}

// Format formats a single diagnostic as a string.
func (f *Formatter) Format(d Diagnostic) string {
	var b strings.Builder
	f.formatDiagnostic(&b, d)
	return b.String()
}

// FormatAll formats all diagnostics in a collection.
func (f *Formatter) FormatAll(c *Collection) string {
	var b strings.Builder
	for i, d := range c.All() {
		if i > 0 {
			b.WriteString("\n")
		}
		f.formatDiagnostic(&b, d)
	}
	return b.String()
}

// Write writes a single diagnostic to the writer.
func (f *Formatter) Write(w io.Writer, d Diagnostic) error {
	_, err := fmt.Fprint(w, f.Format(d))
	return err
}

// WriteAll writes all diagnostics in a collection to the writer.
func (f *Formatter) WriteAll(w io.Writer, c *Collection) error {
	_, err := fmt.Fprint(w, f.FormatAll(c))
	return err
}

// PrintSummary prints a summary of diagnostics.
func (f *Formatter) PrintSummary(w io.Writer, c *Collection) {
	summary := c.Summary()
	if summary.Total == 0 {
		return
	}

	parts := make([]string, 0, 3)
	if summary.Errors > 0 {
		parts = append(parts, f.colorize(fmt.Sprintf("%d error(s)", summary.Errors), colorRed))
	}
	if summary.Warnings > 0 {
		parts = append(parts, f.colorize(fmt.Sprintf("%d warning(s)", summary.Warnings), colorYellow))
	}
	if summary.Infos > 0 {
		parts = append(parts, f.colorize(fmt.Sprintf("%d info(s)", summary.Infos), colorBlue))
	}

	if len(parts) > 0 {
		_, _ = fmt.Fprintf(w, "\n%s\n", strings.Join(parts, ", "))
	}
}

// PrintCategorizedSummary prints a categorized summary of diagnostics.
func (f *Formatter) PrintCategorizedSummary(w io.Writer, c *Collection) {
	cat := c.Categorize()
	if !cat.HasDiagnostics() {
		return
	}

	_, _ = fmt.Fprintln(w)
	_, _ = fmt.Fprintln(w, f.colorize("Diagnostic Summary:", colorBold))

	printCategory := func(name string, diags []Diagnostic, color string) {
		if len(diags) == 0 {
			return
		}
		_, _ = fmt.Fprintf(w, "  %s: %s\n", f.colorize(name, color), f.colorize(fmt.Sprintf("%d", len(diags)), color))
	}

	printCategory("Schema Errors", cat.SchemaErrors, colorRed)
	printCategory("Query Errors", cat.QueryErrors, colorRed)
	printCategory("Config Errors", cat.ConfigErrors, colorRed)
	printCategory("Codegen Errors", cat.CodegenErrors, colorRed)
	printCategory("Warnings", cat.Warnings, colorYellow)
	printCategory("Infos", cat.Infos, colorBlue)
	printCategory("Uncategorized", cat.Uncategorized, colorCyan)
}

func (f *Formatter) formatDiagnostic(b *strings.Builder, d Diagnostic) {
	// Header line: path:line:column: severity: message
	if d.HasLocation() {
		location := f.colorize(fmt.Sprintf("%s:%d:%d", d.Location.Path, d.Location.Line, d.Location.Column), colorCyan)
		fmt.Fprintf(b, "%s: ", location)
	}

	severity := f.colorize(d.Severity.String(), f.severityColor(d.Severity))
	fmt.Fprintf(b, "%s: %s", severity, d.Message)

	// Code
	if f.ShowCode && d.Code != "" {
		codeStr := f.colorize(fmt.Sprintf("[%s]", d.Code), colorMagenta)
		fmt.Fprintf(b, " %s", codeStr)
		if f.ShowCodeDescription {
			desc := CodeDescription(d.Code)
			if desc != "Unknown error code" {
				fmt.Fprintf(b, " (%s)", desc)
			}
		}
	}

	if f.ShowSource && d.Source != "" {
		fmt.Fprintf(b, " (%s)", d.Source)
	}

	b.WriteString("\n")

	// Context
	if f.ShowContext && d.Context != "" {
		f.formatContext(b, d)
	}

	// Suggestions
	if f.ShowSuggestions && len(d.Suggestions) > 0 {
		for _, sugg := range d.Suggestions {
			f.formatSuggestion(b, sugg)
		}
	}

	// Notes
	if f.ShowNotes && len(d.Notes) > 0 {
		for _, note := range d.Notes {
			f.formatNote(b, note)
		}
	}

	// Related info
	if f.ShowRelated && len(d.Related) > 0 {
		for _, rel := range d.Related {
			f.formatRelated(b, rel)
		}
	}
}

func (f *Formatter) formatContext(b *strings.Builder, d Diagnostic) {
	lines := strings.Split(d.Context, "\n")
	for _, line := range lines {
		fmt.Fprintf(b, "  %s %s\n", f.colorize("-->", colorBlue), line)
	}
}

func (f *Formatter) formatSuggestion(b *strings.Builder, sugg Suggestion) {
	fmt.Fprintf(b, "  %s %s\n", f.colorize("help:", colorGreen), sugg.Message)
	if sugg.Replacement != "" {
		fmt.Fprintf(b, "    %s %s\n", f.colorize("=>", colorGreen), sugg.Replacement)
	}
}

func (f *Formatter) formatNote(b *strings.Builder, note string) {
	fmt.Fprintf(b, "  %s %s\n", f.colorize("note:", colorBlue), note)
}

func (f *Formatter) formatRelated(b *strings.Builder, rel RelatedInfo) {
	location := f.colorize(fmt.Sprintf("%s:%d:%d", rel.Location.Path, rel.Location.Line, rel.Location.Column), colorCyan)
	fmt.Fprintf(b, "  %s %s: %s\n", f.colorize("related:", colorMagenta), location, rel.Message)
}

func (f *Formatter) severityColor(s Severity) string {
	switch s {
	case SeverityError:
		return colorRed
	case SeverityWarning:
		return colorYellow
	case SeverityInfo:
		return colorBlue
	default:
		return colorReset
	}
}

func (f *Formatter) colorize(s, color string) string {
	if !f.Colorize {
		return s
	}
	return color + s + colorReset
}

// ANSI color codes.
const (
	colorReset   = "\033[0m"
	colorRed     = "\033[31m"
	colorGreen   = "\033[32m"
	colorYellow  = "\033[33m"
	colorBlue    = "\033[34m"
	colorMagenta = "\033[35m"
	colorCyan    = "\033[36m"
	colorBold    = "\033[1m"
)

// SimpleFormatter provides a simple one-line format for diagnostics.
type SimpleFormatter struct{}

// Format formats a diagnostic in simple format.
func (f *SimpleFormatter) Format(d Diagnostic) string {
	var b strings.Builder
	if d.HasLocation() {
		fmt.Fprintf(&b, "%s:%d:%d: ", d.Location.Path, d.Location.Line, d.Location.Column)
	}
	fmt.Fprintf(&b, "%s: %s", d.Severity, d.Message)
	if d.Code != "" {
		fmt.Fprintf(&b, " [%s]", d.Code)
	}
	return b.String()
}

// JSONFormatter formats diagnostics as JSON.
type JSONFormatter struct {
	Indent bool
}

// Format formats a diagnostic as JSON.
func (f *JSONFormatter) Format(d Diagnostic) string {
	// Simple JSON formatting - for production use, consider encoding/json
	var parts []string

	parts = append(parts, fmt.Sprintf(`"severity":"%s"`, d.Severity))
	parts = append(parts, fmt.Sprintf(`"message":%q`, d.Message))

	if d.Code != "" {
		parts = append(parts, fmt.Sprintf(`"code":%q`, d.Code))
	}

	if d.HasLocation() {
		locParts := []string{
			fmt.Sprintf(`"path":%q`, d.Location.Path),
			fmt.Sprintf(`"line":%d`, d.Location.Line),
			fmt.Sprintf(`"column":%d`, d.Location.Column),
		}
		parts = append(parts, fmt.Sprintf(`"location":{%s}`, strings.Join(locParts, ",")))
	}

	if d.Source != "" {
		parts = append(parts, fmt.Sprintf(`"source":%q`, d.Source))
	}

	if d.Context != "" {
		parts = append(parts, fmt.Sprintf(`"context":%q`, d.Context))
	}

	if len(d.Suggestions) > 0 {
		var suggParts []string
		for _, sugg := range d.Suggestions {
			suggParts = append(suggParts, fmt.Sprintf(`{"message":%q,"replacement":%q}`, sugg.Message, sugg.Replacement))
		}
		parts = append(parts, fmt.Sprintf(`"suggestions":[%s]`, strings.Join(suggParts, ",")))
	}

	if len(d.Notes) > 0 {
		var noteParts []string
		for _, note := range d.Notes {
			noteParts = append(noteParts, fmt.Sprintf(`%q`, note))
		}
		parts = append(parts, fmt.Sprintf(`"notes":[%s]`, strings.Join(noteParts, ",")))
	}

	return "{" + strings.Join(parts, ",") + "}"
}

// FormatCollection formats an entire collection as a JSON array.
func (f *JSONFormatter) FormatCollection(c *Collection) string {
	var parts []string
	for _, d := range c.All() {
		parts = append(parts, f.Format(d))
	}
	if f.Indent {
		return "[\n  " + strings.Join(parts, ",\n  ") + "\n]"
	}
	return "[" + strings.Join(parts, ",") + "]"
}
