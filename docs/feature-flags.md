# Feature Flags

This document captures configuration toggles that change the db-catalyst code generator or runtime behaviour. All flags live inside `db-catalyst.toml` unless otherwise noted.

## Prepared Queries

The generator can emit a prepared-statement aware wrapper that reuses compiled SQL statements and optionally instruments each invocation.

```toml
[prepared_queries]
enabled = true
metrics = true
thread_safe = true
```

- `enabled` *(bool, default `false`)*: when `true`, db-catalyst emits a `prepared.go` companion and the `Prepare` helper. Legacy output remains untouched when this is disabled.
- `metrics` *(bool, default `false`)*: wraps each prepared method with a duration/error callback. Provide a `PreparedMetricsRecorder` implementation when using this toggle; otherwise the hooks remain dormant.
- `thread_safe` *(bool, default `false`)*: guards statement preparation and closure with per-query mutexes so concurrent goroutines can lazily initialize statements safely. When `false`, statements are prepared eagerly in `Prepare` and cached without additional locking.

> **Lifecycle tip:** `Prepare` returns a `PreparedQueries` wrapper that exposes `Close()`. Always call `Close()` when you are done to release held statements; when `thread_safe` is enabled the method is idempotent under the hood.

## Adding New Flags

Keep feature flags additive: default to the legacy behaviour, gate optional output behind explicit opt-in, and document new keys both here and in `db-catalyst-spec.md`. Update goldens and specs whenever a flag changes generated output.
