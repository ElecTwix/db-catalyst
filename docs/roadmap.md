# Roadmap

db-catalyst follows a structured release plan prioritizing simplicity and SQLite focus.

## Version History

| Version | Date | Status |
|---------|------|--------|
| v0.1.0 | Jan 2026 | Released |
| v0.2.0 | TBD | In Progress |
| v0.3.0 | TBD | Planned |
| v0.4.0 | TBD | Future |

## v0.1.0 (Released)

- Config parsing (TOML)
- Schema catalog
- Query analysis
- Go code generation
- CLI with dry-run mode
- sqlc config migration tool
- Prepared queries support

## v0.2.0 - Quality Polish

**Goal:** Improve developer experience and documentation.

### In Progress
- [x] Emit empty slices option
- [x] Column ordering determinism
- [ ] Better parameter name inference
- [ ] Detailed diagnostics
- [ ] Documentation (schema, query, generated-code)

### Known Issues
- No explicit parameter type override in queries
- Limited error context in diagnostics

## v0.3.0 - Optional Features

**Goal:** Additive features without complexity creep.

### Planned Features

1. **Pointer Nullables**
   ```toml
   [generation]
   emit_pointers_for_null = true
   ```
   Generate `*string` instead of `string` for nullable columns.

2. **JSON Tag Control**
   ```toml
   [generation]
   emit_json_tags = false
   ```
   Already wired in generator, needs CLI flag.

3. **Deterministic Caching**
   - Cache parsed schema/query ASTs
   - Target: <200ms for incremental builds

4. **Performance Benchmarks**
   - Internal benchmark suite
   - CI performance regression tests

## v0.4.0 - Extensibility

**Goal:** Prepare for potential multi-database support.

### Exploration

- Define engine interface boundaries
- Experiment with PostgreSQL driver
- No rewrites, just exploration

## Non-Goals

These are explicitly out of scope:

- Multiple database drivers (stick to SQLite)
- Complex migrations (use separate tools)
- ORMs or query builders
- GUI or web interfaces
- Plugin systems

## Contributing

See [AGENTS.md](../AGENTS.md) for contribution guidelines.

## Release Checklist

- [ ] All tests pass
- [ ] Benchmarks run clean
- [ ] Docs updated
- [ ] Changelog entry
- [ ] Version bump
