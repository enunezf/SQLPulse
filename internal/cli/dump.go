package cli

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/enunezf/SQLPulse/internal/adapters/sqlserver"
	"github.com/enunezf/SQLPulse/internal/core/domain"
)

var (
	// Dump command flags
	outputFile   string
	schemaFilter []string
	tableFilter      []string
	noTables         bool
	noViews          bool
	noProcedures     bool
	noFunctions      bool
	noTriggers       bool
	noIndexes        bool
	noForeignKeys    bool
	noConstraints    bool
)

// dumpCmd represents the dump command
var dumpCmd = &cobra.Command{
	Use:   "dump",
	Short: "Extract DDL from SQL Server database",
	Long: `Extract the complete DDL (Data Definition Language) from a SQL Server database.

This command generates SQL scripts that can recreate the database schema,
including tables, views, stored procedures, functions, triggers, indexes,
and constraints.

Examples:
  # Dump entire database schema
  sqlpulse dump --server localhost --database mydb --user sa --password secret

  # Dump only tables and indexes
  sqlpulse dump --server localhost --database mydb --user sa --password secret --no-views --no-procedures --no-functions --no-triggers

  # Dump specific schemas
  sqlpulse dump --server localhost --database mydb --user sa --password secret --schema dbo,sales

  # Dump to file
  sqlpulse dump --server localhost --database mydb --user sa --password secret --output schema.sql

  # Dump specific tables
  sqlpulse dump --server localhost --database mydb --user sa --password secret --table Users,Orders`,
	RunE: runDump,
}

func init() {
	rootCmd.AddCommand(dumpCmd)

	dumpCmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file (default: stdout)")
	dumpCmd.Flags().StringSliceVar(&schemaFilter, "schema", nil, "Filter by schema names (comma-separated)")
	dumpCmd.Flags().StringSliceVar(&tableFilter, "table", nil, "Filter by table names (comma-separated)")
	dumpCmd.Flags().BoolVar(&noTables, "no-tables", false, "Exclude tables")
	dumpCmd.Flags().BoolVar(&noViews, "no-views", false, "Exclude views")
	dumpCmd.Flags().BoolVar(&noProcedures, "no-procedures", false, "Exclude stored procedures")
	dumpCmd.Flags().BoolVar(&noFunctions, "no-functions", false, "Exclude functions")
	dumpCmd.Flags().BoolVar(&noTriggers, "no-triggers", false, "Exclude triggers")
	dumpCmd.Flags().BoolVar(&noIndexes, "no-indexes", false, "Exclude indexes (non-PK)")
	dumpCmd.Flags().BoolVar(&noForeignKeys, "no-foreign-keys", false, "Exclude foreign keys")
	dumpCmd.Flags().BoolVar(&noConstraints, "no-constraints", false, "Exclude check constraints")
}

func runDump(cmd *cobra.Command, args []string) error {
	config := GetConnectionConfig()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Connecting to %s...\n", config.SafeString())

	// Create adapter and connect
	adapter := sqlserver.NewAdapter(config)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	if err := adapter.Connect(ctx); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer adapter.Close()

	fmt.Fprintln(os.Stderr, "\033[32m✓ Connected\033[0m")

	// Build dump options
	opts := &domain.DumpOptions{
		IncludeTables:      !noTables,
		IncludeViews:       !noViews,
		IncludeProcedures:  !noProcedures,
		IncludeFunctions:   !noFunctions,
		IncludeTriggers:    !noTriggers,
		IncludeIndexes:     !noIndexes,
		IncludeForeignKeys: !noForeignKeys,
		IncludeConstraints: !noConstraints,
		SchemaFilter:       schemaFilter,
		TableFilter:        tableFilter,
		OutputFormat:       "sql",
	}

	// Create schema extractor
	extractor := sqlserver.NewSchemaExtractor(adapter.DB())

	fmt.Fprintln(os.Stderr, "Extracting schema...")

	schema, err := extractor.ExtractSchema(ctx, opts)
	if err != nil {
		return fmt.Errorf("extraction failed: %w", err)
	}

	// Generate output
	output := generateDDL(schema, opts)

	// Write output
	if outputFile != "" {
		if err := os.WriteFile(outputFile, []byte(output), 0644); err != nil {
			return fmt.Errorf("failed to write output file: %w", err)
		}
		fmt.Fprintf(os.Stderr, "\033[32m✓ DDL written to %s\033[0m\n", outputFile)
	} else {
		fmt.Println(output)
	}

	// Print summary to stderr
	printSummary(schema)

	return nil
}

func generateDDL(schema *domain.DatabaseSchema, opts *domain.DumpOptions) string {
	var sb strings.Builder

	// Header
	sb.WriteString("-- ============================================\n")
	sb.WriteString(fmt.Sprintf("-- SQLPulse DDL Export\n"))
	sb.WriteString(fmt.Sprintf("-- Database: %s\n", schema.DatabaseName))
	sb.WriteString(fmt.Sprintf("-- Generated: %s\n", time.Now().Format(time.RFC3339)))
	sb.WriteString("-- ============================================\n\n")

	// Schemas
	if len(schema.Schemas) > 0 {
		sb.WriteString("-- ============================================\n")
		sb.WriteString("-- SCHEMAS\n")
		sb.WriteString("-- ============================================\n\n")
		for _, s := range schema.Schemas {
			sb.WriteString(s.GenerateSQL())
			sb.WriteString(";\nGO\n\n")
		}
	}

	// Tables
	if opts.IncludeTables && len(schema.Tables) > 0 {
		sb.WriteString("-- ============================================\n")
		sb.WriteString("-- TABLES\n")
		sb.WriteString("-- ============================================\n\n")
		for _, t := range schema.Tables {
			sb.WriteString(fmt.Sprintf("-- Table: [%s].[%s]\n", t.SchemaName, t.Name))
			sb.WriteString(t.GenerateSQL())
			sb.WriteString(";\nGO\n\n")
		}
	}

	// Indexes (non-PK)
	if opts.IncludeIndexes {
		var hasIndexes bool
		for _, t := range schema.Tables {
			if len(t.Indexes) > 0 {
				hasIndexes = true
				break
			}
		}
		if hasIndexes {
			sb.WriteString("-- ============================================\n")
			sb.WriteString("-- INDEXES\n")
			sb.WriteString("-- ============================================\n\n")
			for _, t := range schema.Tables {
				for _, idx := range t.Indexes {
					sql := idx.GenerateSQL()
					if sql != "" {
						sb.WriteString(fmt.Sprintf("-- Index: [%s] on [%s].[%s]\n", idx.Name, t.SchemaName, t.Name))
						sb.WriteString(sql)
						sb.WriteString(";\nGO\n\n")
					}
				}
			}
		}
	}

	// Foreign Keys
	if opts.IncludeForeignKeys {
		var hasFKs bool
		for _, t := range schema.Tables {
			if len(t.ForeignKeys) > 0 {
				hasFKs = true
				break
			}
		}
		if hasFKs {
			sb.WriteString("-- ============================================\n")
			sb.WriteString("-- FOREIGN KEYS\n")
			sb.WriteString("-- ============================================\n\n")
			for _, t := range schema.Tables {
				for _, fk := range t.ForeignKeys {
					sb.WriteString(fmt.Sprintf("-- FK: [%s]\n", fk.Name))
					sb.WriteString(fk.GenerateSQL())
					sb.WriteString(";\nGO\n\n")
				}
			}
		}
	}

	// Check Constraints
	if opts.IncludeConstraints {
		var hasConstraints bool
		for _, t := range schema.Tables {
			if len(t.CheckConstraints) > 0 {
				hasConstraints = true
				break
			}
		}
		if hasConstraints {
			sb.WriteString("-- ============================================\n")
			sb.WriteString("-- CHECK CONSTRAINTS\n")
			sb.WriteString("-- ============================================\n\n")
			for _, t := range schema.Tables {
				for _, cc := range t.CheckConstraints {
					sb.WriteString(fmt.Sprintf("-- Check: [%s]\n", cc.Name))
					sb.WriteString(cc.GenerateSQL())
					sb.WriteString(";\nGO\n\n")
				}
			}
		}
	}

	// Views
	if opts.IncludeViews && len(schema.Views) > 0 {
		sb.WriteString("-- ============================================\n")
		sb.WriteString("-- VIEWS\n")
		sb.WriteString("-- ============================================\n\n")
		for _, v := range schema.Views {
			sb.WriteString(fmt.Sprintf("-- View: [%s].[%s]\n", v.SchemaName, v.Name))
			if v.Definition != "" {
				sb.WriteString(v.Definition)
				sb.WriteString(";\nGO\n\n")
			} else {
				sb.WriteString("-- (definition not available - possibly encrypted)\n\n")
			}
		}
	}

	// Stored Procedures
	if opts.IncludeProcedures && len(schema.StoredProcedures) > 0 {
		sb.WriteString("-- ============================================\n")
		sb.WriteString("-- STORED PROCEDURES\n")
		sb.WriteString("-- ============================================\n\n")
		for _, p := range schema.StoredProcedures {
			sb.WriteString(fmt.Sprintf("-- Procedure: [%s].[%s]\n", p.SchemaName, p.Name))
			if p.Definition != "" {
				sb.WriteString(p.Definition)
				sb.WriteString(";\nGO\n\n")
			} else {
				sb.WriteString("-- (definition not available - possibly encrypted)\n\n")
			}
		}
	}

	// Functions
	if opts.IncludeFunctions && len(schema.Functions) > 0 {
		sb.WriteString("-- ============================================\n")
		sb.WriteString("-- FUNCTIONS\n")
		sb.WriteString("-- ============================================\n\n")
		for _, f := range schema.Functions {
			sb.WriteString(fmt.Sprintf("-- Function: [%s].[%s] (%s)\n", f.SchemaName, f.Name, f.FuncType))
			if f.Definition != "" {
				sb.WriteString(f.Definition)
				sb.WriteString(";\nGO\n\n")
			} else {
				sb.WriteString("-- (definition not available - possibly encrypted)\n\n")
			}
		}
	}

	// Triggers
	if opts.IncludeTriggers && len(schema.Triggers) > 0 {
		sb.WriteString("-- ============================================\n")
		sb.WriteString("-- TRIGGERS\n")
		sb.WriteString("-- ============================================\n\n")
		for _, tr := range schema.Triggers {
			sb.WriteString(fmt.Sprintf("-- Trigger: [%s] on [%s].[%s]\n", tr.Name, tr.SchemaName, tr.TableName))
			if tr.Definition != "" {
				sb.WriteString(tr.Definition)
				sb.WriteString(";\nGO\n\n")
			} else {
				sb.WriteString("-- (definition not available - possibly encrypted)\n\n")
			}
		}
	}

	sb.WriteString("-- ============================================\n")
	sb.WriteString("-- END OF DDL EXPORT\n")
	sb.WriteString("-- ============================================\n")

	return sb.String()
}

func printSummary(schema *domain.DatabaseSchema) {
	var indexCount, fkCount, checkCount int
	for _, t := range schema.Tables {
		indexCount += len(t.Indexes)
		fkCount += len(t.ForeignKeys)
		checkCount += len(t.CheckConstraints)
	}

	fmt.Fprintln(os.Stderr)
	fmt.Fprintln(os.Stderr, strings.Repeat("─", 40))
	fmt.Fprintf(os.Stderr, "\033[1mExtraction Summary:\033[0m\n")
	fmt.Fprintf(os.Stderr, "  Schemas:           %d\n", len(schema.Schemas))
	fmt.Fprintf(os.Stderr, "  Tables:            %d\n", len(schema.Tables))
	fmt.Fprintf(os.Stderr, "  Indexes:           %d\n", indexCount)
	fmt.Fprintf(os.Stderr, "  Foreign Keys:      %d\n", fkCount)
	fmt.Fprintf(os.Stderr, "  Check Constraints: %d\n", checkCount)
	fmt.Fprintf(os.Stderr, "  Views:             %d\n", len(schema.Views))
	fmt.Fprintf(os.Stderr, "  Procedures:        %d\n", len(schema.StoredProcedures))
	fmt.Fprintf(os.Stderr, "  Functions:         %d\n", len(schema.Functions))
	fmt.Fprintf(os.Stderr, "  Triggers:          %d\n", len(schema.Triggers))
	fmt.Fprintln(os.Stderr, strings.Repeat("─", 40))
}
