# CLAUDE.md - SQLPulse Project Context

## Project Overview

SQLPulse is a CLI tool for SQL Server administration with built-in safety mechanisms. It follows hexagonal architecture and is written in Go.

## Tech Stack

- **Language:** Go 1.22+
- **CLI Framework:** Cobra
- **Database Driver:** github.com/microsoft/go-mssqldb
- **Architecture:** Hexagonal (Ports & Adapters)

## Project Structure

```
SQLPulse/
├── cmd/sqlpulse/main.go           # Entry point
├── internal/
│   ├── cli/                       # Cobra commands
│   │   ├── root.go                # Root command + global flags
│   │   ├── connect.go             # Test connection
│   │   ├── dump.go                # Extract DDL
│   │   └── diff.go                # Compare schemas
│   ├── core/
│   │   ├── domain/                # Domain models
│   │   │   ├── connection.go      # Connection config
│   │   │   ├── schema.go          # Schema objects (Table, Column, Index, etc.)
│   │   │   └── diff.go            # Diff models (Difference, DiffResult)
│   │   ├── ports/                 # Interfaces
│   │   │   ├── database.go        # Database port
│   │   │   └── schema.go          # Schema extraction port
│   │   └── services/              # Business logic
│   │       └── comparator.go      # Schema comparison service
│   ├── adapters/
│   │   └── sqlserver/
│   │       ├── adapter.go         # SQL Server connection
│   │       └── schema.go          # DDL extraction from sys tables
│   └── security/
│       └── approval.go            # Approval system (dry-run)
└── doc/                           # Documentation
    ├── ROADMAP.md
    ├── SPECIFICATIONS.md
    └── ARCHITECTURE.md
```

## Key Design Decisions

1. **Hexagonal Architecture**: Separates business logic from infrastructure
2. **Approval System**: All destructive operations require user confirmation
3. **Three approval levels**: ReadOnly (no confirm), Modification (y/n), Destructive (type CONFIRM)
4. **Git-style diff output**: Familiar format for developers

## Build & Run

```bash
# Build
go build ./cmd/sqlpulse

# Run tests (when available)
go test ./...

# Common commands
./sqlpulse connect -s localhost -d master -u sa -p secret
./sqlpulse dump -s localhost -d mydb -u sa -p secret -o schema.sql
./sqlpulse diff -s localhost -d source -u sa -p secret --target-database target
```

## Current Implementation Status

See `doc/PROGRESS.md` for detailed progress tracking.

## Code Conventions

- Use `fmt.Errorf` with `%w` for error wrapping
- Domain models in `internal/core/domain/`
- All SQL Server queries use parameterized queries (`@p1`, `@p2`, etc.)
- CLI flags follow Cobra conventions (short flags for common options)
- Colors in terminal output: green for success, yellow for warnings, red for errors

## SQL Server System Tables Used

- `sys.tables`, `sys.columns` - Table/column metadata
- `sys.indexes`, `sys.index_columns` - Index information
- `sys.foreign_keys`, `sys.foreign_key_columns` - FK constraints
- `sys.check_constraints` - Check constraints
- `sys.sql_modules` - View/procedure/function definitions
- `sys.schemas` - Schema information

## Next Steps (Phase 4)

Performance dashboard implementation:
- Query `sys.dm_exec_query_stats` for top queries
- Query `sys.dm_os_ring_buffers` for CPU/RAM telemetry
- New command: `sqlpulse perf`
