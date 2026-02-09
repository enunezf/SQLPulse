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
	"github.com/enunezf/SQLPulse/internal/core/services"
)

var (
	// Target connection flags
	targetServer   string
	targetDatabase string
	targetUser     string
	targetPassword string
	targetTrusted  bool
	targetPort     int

	// Diff options
	outputFormat     string
	generateMigration bool
	migrationFile    string
	ignoreCollation  bool
)

// diffCmd represents the diff command
var diffCmd = &cobra.Command{
	Use:   "diff",
	Short: "Compare schemas between two SQL Server databases",
	Long: `Compare the schema of two SQL Server databases and show differences.

This command connects to a source and target database, extracts their schemas,
and produces a diff showing what has changed. Output can be in git-diff style
or as a migration script.

The source database is specified using the global flags (--server, --database, etc.)
The target database is specified using --target-* flags.

Examples:
  # Compare two databases on the same server
  sqlpulse diff --server localhost --database source_db --user sa --password secret \
      --target-database target_db

  # Compare databases on different servers
  sqlpulse diff --server server1 --database db1 --user sa --password secret \
      --target-server server2 --target-database db2 --target-user sa --target-password secret2

  # Generate migration script
  sqlpulse diff --server localhost --database dev_db --user sa --password secret \
      --target-server localhost --target-database prod_db --target-user sa --target-password secret \
      --generate-migration --migration-file migration.sql

  # Compare only tables, ignore procedures
  sqlpulse diff --server localhost --database db1 --user sa --password secret \
      --target-database db2 --no-procedures --no-functions --no-views`,
	RunE: runDiff,
}

func init() {
	rootCmd.AddCommand(diffCmd)

	// Target database flags
	diffCmd.Flags().StringVar(&targetServer, "target-server", "", "Target SQL Server (defaults to source server)")
	diffCmd.Flags().StringVar(&targetDatabase, "target-database", "", "Target database name (required)")
	diffCmd.Flags().StringVar(&targetUser, "target-user", "", "Target username (defaults to source user)")
	diffCmd.Flags().StringVar(&targetPassword, "target-password", "", "Target password (defaults to source password)")
	diffCmd.Flags().BoolVar(&targetTrusted, "target-trusted", false, "Use Windows auth for target")
	diffCmd.Flags().IntVar(&targetPort, "target-port", 0, "Target port (defaults to source port)")

	// Output options
	diffCmd.Flags().StringVar(&outputFormat, "format", "git", "Output format: git, summary, or full")
	diffCmd.Flags().BoolVar(&generateMigration, "generate-migration", false, "Generate migration SQL script")
	diffCmd.Flags().StringVar(&migrationFile, "migration-file", "", "Output file for migration script")
	diffCmd.Flags().BoolVar(&ignoreCollation, "ignore-collation", false, "Ignore collation differences")

	// Reuse filter flags from dump (already defined in dump.go)
	diffCmd.Flags().BoolVar(&noTables, "no-tables", false, "Exclude tables from comparison")
	diffCmd.Flags().BoolVar(&noViews, "no-views", false, "Exclude views from comparison")
	diffCmd.Flags().BoolVar(&noProcedures, "no-procedures", false, "Exclude stored procedures")
	diffCmd.Flags().BoolVar(&noFunctions, "no-functions", false, "Exclude functions")
	diffCmd.Flags().BoolVar(&noTriggers, "no-triggers", false, "Exclude triggers")
	diffCmd.Flags().BoolVar(&noIndexes, "no-indexes", false, "Exclude indexes")
	diffCmd.Flags().BoolVar(&noForeignKeys, "no-foreign-keys", false, "Exclude foreign keys")
	diffCmd.Flags().BoolVar(&noConstraints, "no-constraints", false, "Exclude check constraints")

	diffCmd.MarkFlagRequired("target-database")
}

func runDiff(cmd *cobra.Command, args []string) error {
	// Build source config
	sourceConfig := GetConnectionConfig()
	if err := sourceConfig.Validate(); err != nil {
		return fmt.Errorf("source configuration error: %w", err)
	}

	// Build target config (inherit from source where not specified)
	targetConfig := domain.NewConnectionConfig()
	targetConfig.Server = targetServer
	if targetConfig.Server == "" {
		targetConfig.Server = sourceConfig.Server
	}
	targetConfig.Database = targetDatabase
	targetConfig.User = targetUser
	if targetConfig.User == "" {
		targetConfig.User = sourceConfig.User
	}
	targetConfig.Password = targetPassword
	if targetConfig.Password == "" {
		targetConfig.Password = sourceConfig.Password
	}
	targetConfig.TrustedAuth = targetTrusted
	if !targetTrusted && !sourceConfig.TrustedAuth && targetUser == "" {
		targetConfig.TrustedAuth = sourceConfig.TrustedAuth
	}
	targetConfig.Port = targetPort
	if targetConfig.Port == 0 {
		targetConfig.Port = sourceConfig.Port
	}
	targetConfig.TrustServer = sourceConfig.TrustServer

	if err := targetConfig.Validate(); err != nil {
		return fmt.Errorf("target configuration error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Connect to source
	fmt.Fprintf(os.Stderr, "Connecting to source: %s...\n", sourceConfig.SafeString())
	sourceAdapter := sqlserver.NewAdapter(sourceConfig)
	if err := sourceAdapter.Connect(ctx); err != nil {
		return fmt.Errorf("source connection failed: %w", err)
	}
	defer sourceAdapter.Close()
	fmt.Fprintln(os.Stderr, "\033[32m✓ Source connected\033[0m")

	// Connect to target
	fmt.Fprintf(os.Stderr, "Connecting to target: %s...\n", targetConfig.SafeString())
	targetAdapter := sqlserver.NewAdapter(targetConfig)
	if err := targetAdapter.Connect(ctx); err != nil {
		return fmt.Errorf("target connection failed: %w", err)
	}
	defer targetAdapter.Close()
	fmt.Fprintln(os.Stderr, "\033[32m✓ Target connected\033[0m")

	// Build extraction options
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
	}

	// Extract source schema
	fmt.Fprintln(os.Stderr, "Extracting source schema...")
	sourceExtractor := sqlserver.NewSchemaExtractor(sourceAdapter.DB())
	sourceSchema, err := sourceExtractor.ExtractSchema(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to extract source schema: %w", err)
	}

	// Extract target schema
	fmt.Fprintln(os.Stderr, "Extracting target schema...")
	targetExtractor := sqlserver.NewSchemaExtractor(targetAdapter.DB())
	targetSchema, err := targetExtractor.ExtractSchema(ctx, opts)
	if err != nil {
		return fmt.Errorf("failed to extract target schema: %w", err)
	}

	// Build diff options
	diffOpts := &domain.DiffOptions{
		IncludeTables:      !noTables,
		IncludeViews:       !noViews,
		IncludeProcedures:  !noProcedures,
		IncludeFunctions:   !noFunctions,
		IncludeTriggers:    !noTriggers,
		IncludeIndexes:     !noIndexes,
		IncludeForeignKeys: !noForeignKeys,
		IncludeConstraints: !noConstraints,
		IgnoreCollation:    ignoreCollation,
		IgnoreWhitespace:   true,
	}

	// Compare schemas
	fmt.Fprintln(os.Stderr, "Comparing schemas...")
	comparator := services.NewSchemaComparator(diffOpts)
	result := comparator.Compare(sourceSchema, targetSchema)

	// Output results
	fmt.Fprintln(os.Stderr)

	if !result.HasDifferences() {
		fmt.Println("\033[32m✓ Schemas are identical\033[0m")
		return nil
	}

	// Print based on format
	switch outputFormat {
	case "git":
		fmt.Println(result.PrintGitStyle())
	case "summary":
		printDiffSummary(result)
	case "full":
		fmt.Println(result.PrintGitStyle())
		fmt.Println()
		printDiffSummary(result)
	default:
		fmt.Println(result.PrintGitStyle())
	}

	// Generate migration script if requested
	if generateMigration {
		migration := result.GenerateMigrationScript()
		if migrationFile != "" {
			if err := os.WriteFile(migrationFile, []byte(migration), 0644); err != nil {
				return fmt.Errorf("failed to write migration file: %w", err)
			}
			fmt.Fprintf(os.Stderr, "\n\033[32m✓ Migration script written to %s\033[0m\n", migrationFile)
		} else {
			fmt.Println("\n" + migration)
		}
	}

	return nil
}

func printDiffSummary(result *domain.DiffResult) {
	fmt.Println(strings.Repeat("─", 50))
	fmt.Printf("\033[1mDiff Summary: %s → %s\033[0m\n", result.SourceDatabase, result.TargetDatabase)
	fmt.Println(strings.Repeat("─", 50))

	fmt.Printf("  Total differences: %d\n", result.Summary.TotalDifferences)
	fmt.Printf("  \033[32m+ Added:   %d\033[0m (in target only)\n", result.Summary.Added)
	fmt.Printf("  \033[31m- Removed: %d\033[0m (in source only)\n", result.Summary.Removed)
	fmt.Printf("  \033[33m~ Modified: %d\033[0m\n", result.Summary.Modified)

	if len(result.Summary.ByCategory) > 0 {
		fmt.Println()
		fmt.Println("  By category:")
		for cat, count := range result.Summary.ByCategory {
			fmt.Printf("    %-15s %d\n", cat+":", count)
		}
	}
	fmt.Println(strings.Repeat("─", 50))
}
