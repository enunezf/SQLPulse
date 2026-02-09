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

### `diff`

Compare schemas between two SQL Server databases and show differences.

```bash
sqlpulse diff [flags]
```

**Examples:**
```bash
# Compare two databases on the same server
sqlpulse diff --server localhost --database source_db --user sa --password secret \
    --target-database target_db

# Compare databases on different servers
sqlpulse diff --server server1 --database db1 --user sa --password secret \
    --target-server server2 --target-database db2 --target-user sa --target-password secret2

# Generate migration script
sqlpulse diff --server localhost --database dev_db --user sa --password secret \
    --target-database prod_db --generate-migration --migration-file migration.sql
```

**Target Flags:**
| Flag | Description |
|------|-------------|
| `--target-server` | Target SQL Server (defaults to source) |
| `--target-database` | Target database name (required) |
| `--target-user` | Target username (defaults to source) |
| `--target-password` | Target password (defaults to source) |
| `--target-trusted` | Use Windows auth for target |
| `--target-port` | Target port (defaults to source) |

**Output Flags:**
| Flag | Description |
|------|-------------|
| `--format` | Output format: git, summary, or full (default: git) |
| `--generate-migration` | Generate migration SQL script |
| `--migration-file` | Output file for migration script |
| `--ignore-collation` | Ignore collation differences |

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
│   ├── services/
│   │   └── comparator.go        # Schema comparison service
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
| 3 | Schema comparison (`diff` command) | Done |
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
