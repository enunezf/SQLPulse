# SQLPulse - Progress Tracking

Last updated: 2026-02-09

## Completed Phases

### Phase 1: CLI Foundation + SQL Server Connection ✅

**Status:** COMPLETE

**Implemented:**
- [x] Go module initialization (`github.com/enunezf/SQLPulse`)
- [x] Cobra CLI framework setup
- [x] Root command with global flags (`--server`, `--database`, `--user`, `--password`, `--trusted`, `--port`, `--trust-cert`, `--dry-run`)
- [x] `connect` command - tests connection and shows server info
- [x] SQL Server adapter with connection pooling
- [x] Support for SQL and Windows authentication
- [x] Approval system with 3 levels (ReadOnly, Modification, Destructive)
- [x] InteractiveApprover, AutoApprover, DryRunApprover implementations

**Files:**
- `cmd/sqlpulse/main.go`
- `internal/cli/root.go`
- `internal/cli/connect.go`
- `internal/core/domain/connection.go`
- `internal/core/ports/database.go`
- `internal/adapters/sqlserver/adapter.go`
- `internal/security/approval.go`

---

### Phase 2: DDL Extraction (`dump` command) ✅

**Status:** COMPLETE

**Implemented:**
- [x] Schema extraction from SQL Server system tables
- [x] Table extraction with columns, PKs, indexes, FKs, check constraints
- [x] View, stored procedure, function, trigger extraction
- [x] `dump` command with filtering options
- [x] SQL script generation with GO separators
- [x] Output to file or stdout

**Extraction capabilities:**
- Tables: columns with all properties (type, length, precision, nullable, identity, defaults, computed)
- Primary keys (clustered/nonclustered)
- Indexes with included columns and filters
- Foreign keys with cascade actions
- Check constraints
- Views, procedures, functions, triggers (from sys.sql_modules)

**Files:**
- `internal/cli/dump.go`
- `internal/core/domain/schema.go`
- `internal/core/ports/schema.go`
- `internal/adapters/sqlserver/schema.go`

---

### Phase 3: Schema Comparison (`diff` command) ✅

**Status:** COMPLETE

**Implemented:**
- [x] Schema comparison service
- [x] Diff models (Difference, DiffResult, DiffSummary)
- [x] Detection: added/removed/modified objects
- [x] Comparison of: tables, columns, indexes, FKs, constraints, views, procedures, functions, triggers
- [x] Git-style diff output
- [x] Summary output format
- [x] Migration script generation
- [x] Target database flags (inherit from source when not specified)

**Comparison capabilities:**
- Tables: existence, structure
- Columns: data type, length, precision, scale, nullability, identity, collation
- Indexes: columns, unique, clustered
- Foreign keys: existence
- Check constraints: existence
- Views/Procedures/Functions/Triggers: definition comparison (with whitespace normalization)

**Files:**
- `internal/cli/diff.go`
- `internal/core/domain/diff.go`
- `internal/core/services/comparator.go`

---

## Pending Phases

### Phase 4: Performance Dashboard (Top Queries) 🔜

**Status:** NOT STARTED

**Priority:** High

**To implement:**
- [ ] `perf` command
- [ ] Query `sys.dm_exec_query_stats` for top queries by CPU time
- [ ] Query `sys.dm_os_ring_buffers` for CPU/RAM telemetry
- [ ] Display top N queries with execution stats
- [ ] Option to filter by database
- [ ] Output formats: table, json

**Specifications (from SPECIFICATIONS.md):**
- CPU/RAM: Query `sys.dm_os_ring_buffers`
- Top Queries: Based on `sys.dm_exec_query_stats` ordered by `total_worker_time`

**Suggested structure:**
```
internal/
├── core/
│   └── domain/
│       └── performance.go      # NEW: Query stats, resource usage models
├── adapters/
│   └── sqlserver/
│       └── performance.go      # NEW: DMV queries
└── cli/
    └── perf.go                 # NEW: perf command
```

---

### Phase 5: Execution Plans + Dry Run Improvements 📋

**Status:** NOT STARTED

**Priority:** Critical (dry-run already partially implemented)

**To implement:**
- [ ] `analyze` command for query analysis
- [ ] Execute `SET SHOWPLAN_XML ON` to capture execution plans
- [ ] Parse XML and highlight expensive operations (EstimateIO, EstimateCPU)
- [ ] Suggest missing indexes from `sys.dm_db_missing_index_details`
- [ ] Integrate approval system with actual DDL execution

**Specifications:**
- Command: `sqlpulse analyze --query "SELECT..."`
- Parse SHOWPLAN_XML for expensive nodes
- Suggest indexes from `sys.dm_db_missing_index_details`

---

### Phase 6: AI Integration (Claude API) 📋

**Status:** NOT STARTED

**Priority:** Low

**To implement:**
- [ ] Integration with Claude API
- [ ] Send schema + execution plan as context
- [ ] Receive optimization suggestions
- [ ] Configuration for API key

---

## Git Commits Summary

| Commit | Description |
|--------|-------------|
| `5d0ca41` | Initial commit |
| `d5227c7` | Phase 1 & 2: CLI foundation and DDL extraction |
| (pending) | Phase 3: Schema comparison (diff command) |

---

## How to Continue

1. **Review current state:**
   ```bash
   ./sqlpulse --help
   ./sqlpulse connect --help
   ./sqlpulse dump --help
   ./sqlpulse diff --help
   ```

2. **Next phase (Phase 4):**
   - Create `internal/core/domain/performance.go` with models for query stats
   - Create `internal/adapters/sqlserver/performance.go` with DMV queries
   - Create `internal/cli/perf.go` with the perf command
   - Key DMVs: `sys.dm_exec_query_stats`, `sys.dm_exec_sql_text`, `sys.dm_os_ring_buffers`

3. **Build and test:**
   ```bash
   go build ./cmd/sqlpulse
   ./sqlpulse perf --server localhost --database master --user sa --password secret
   ```

---

## Notes

- The approval system is implemented but only used in `ExecuteWithApproval` method
- Phase 5 will fully integrate approval with actual DDL execution
- All SQL queries use parameterized queries for security
- Error handling uses `fmt.Errorf` with `%w` for proper error wrapping
