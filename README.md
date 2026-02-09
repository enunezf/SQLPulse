# SQLPulse

A command-line tool for SQL Server administration with built-in safety mechanisms.

## Features

- **Connection Management**: SQL Server and Windows authentication support
- **DDL Extraction**: Export complete database schema (tables, views, procedures, functions, triggers, indexes, constraints)
- **Safety First**: Built-in approval system for destructive operations (dry-run mode)
- **Schema Comparison**: Compare schemas between databases (coming soon)
- **Performance Dashboard**: Query analysis and optimization suggestions (coming soon)

## Installation

### From Source

```bash
git clone https://github.com/enunezf/SQLPulse.git
cd SQLPulse
go build ./cmd/sqlpulse
```

### Requirements

- Go 1.22 or higher
- SQL Server 2016 or higher

## Quick Start

### Test Connection

```bash
# SQL Server authentication
sqlpulse connect --server localhost --database master --user sa --password yourpassword

# Windows authentication
sqlpulse connect --server localhost --database master --trusted

# With custom port
sqlpulse connect --server myserver --database mydb --user sa --password secret --port 1434
```

### Export Database Schema (DDL)

```bash
# Export entire database schema to stdout
sqlpulse dump --server localhost --database mydb --user sa --password secret

# Export to file
sqlpulse dump --server localhost --database mydb --user sa --password secret --output schema.sql

# Export specific schemas
sqlpulse dump --server localhost --database mydb --user sa --password secret --schema dbo,sales

# Export specific tables
sqlpulse dump --server localhost --database mydb --user sa --password secret --table Users,Orders

# Export only tables and indexes (exclude procedures, views, etc.)
sqlpulse dump --server localhost --database mydb --user sa --password secret \
    --no-views --no-procedures --no-functions --no-triggers
```

## Commands

### `connect`

Test connection to SQL Server and display server information.

```bash
sqlpulse connect [flags]
```

### `dump`

Extract DDL (Data Definition Language) from a SQL Server database.

```bash
sqlpulse dump [flags]
```

**Flags:**
| Flag | Description |
|------|-------------|
| `-o, --output` | Output file (default: stdout) |
| `--schema` | Filter by schema names (comma-separated) |
| `--table` | Filter by table names (comma-separated) |
| `--no-tables` | Exclude tables |
| `--no-views` | Exclude views |
| `--no-procedures` | Exclude stored procedures |
| `--no-functions` | Exclude functions |
| `--no-triggers` | Exclude triggers |
| `--no-indexes` | Exclude indexes (non-PK) |
| `--no-foreign-keys` | Exclude foreign keys |
| `--no-constraints` | Exclude check constraints |

## Global Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--server` | `-s` | SQL Server hostname or IP address |
| `--database` | `-d` | Database name |
| `--user` | `-u` | Username for SQL authentication |
| `--password` | `-p` | Password for SQL authentication |
| `--trusted` | `-t` | Use Windows/Integrated authentication |
| `--port` | | SQL Server port (default: 1433) |
| `--trust-cert` | | Trust server certificate (insecure) |
| `--dry-run` | | Show what would be executed without making changes |

## Safety Features

SQLPulse implements a mandatory approval system for operations that modify data:

| Level | Description | Confirmation |
|-------|-------------|--------------|
| **ReadOnly** | SELECT queries, schema extraction | None |
| **Modification** | INSERT, UPDATE, ALTER | Simple y/n prompt |
| **Destructive** | DROP, TRUNCATE, DELETE | Type "CONFIRM" |

Use `--dry-run` to preview operations without executing them.

## Project Structure

```
SQLPulse/
├── cmd/
│   └── sqlpulse/
│       └── main.go              # Entry point
├── internal/
│   ├── cli/
│   │   ├── root.go              # Root Cobra command
│   │   ├── connect.go           # connect command
│   │   └── dump.go              # dump command
│   ├── core/
│   │   ├── domain/
│   │   │   ├── connection.go    # Connection config model
│   │   │   └── schema.go        # Schema models (Table, Column, Index, etc.)
│   │   └── ports/
│   │       ├── database.go      # Database port interface
│   │       └── schema.go        # Schema extraction port
│   ├── adapters/
│   │   └── sqlserver/
│   │       ├── adapter.go       # SQL Server connection adapter
│   │       └── schema.go        # DDL extraction implementation
│   └── security/
│       └── approval.go          # Approval system (dry-run)
├── go.mod
├── go.sum
└── .gitignore
```

## Roadmap

| Phase | Feature | Status |
|-------|---------|--------|
| 1 | Cobra CLI + SQL Server connection | Done |
| 2 | DDL extraction (`dump` command) | Done |
| 3 | Schema comparison (`diff` command) | Planned |
| 4 | Performance dashboard (Top Queries) | Planned |
| 5 | Dry-run execution and improvements | Planned |
| 6 | AI integration (Claude API) | Planned |

## Security Considerations

- Never commit credentials to version control
- Use environment variables or secure vaults for passwords
- The `--trust-cert` flag should only be used in development
- Review all DDL dumps before executing on production systems

## License

MIT License

## Contributing

Contributions are welcome. Please open an issue first to discuss what you would like to change.
