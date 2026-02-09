// Package cli provides the command-line interface for SQLPulse.
package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/enunezf/SQLPulse/internal/core/domain"
)

var (
	// Global flags
	server      string
	database    string
	user        string
	password    string
	trustedAuth bool
	port        int
	trustCert   bool
	dryRun      bool

	// Version information
	version = "0.1.0"
)

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "sqlpulse",
	Short: "SQLPulse - SQL Server administration CLI",
	Long: `SQLPulse is a command-line tool for SQL Server administration.

It provides safe database operations with a mandatory approval system
that prevents accidental execution of destructive commands.

Features:
  - Connection management with SQL and Windows authentication
  - Safe operation execution with dry-run support
  - Schema comparison and synchronization (coming soon)
  - Data migration tools (coming soon)

Example:
  sqlpulse connect --server localhost --database master --user sa --password secret`,
	Version: version,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	// Global flags available to all commands
	rootCmd.PersistentFlags().StringVarP(&server, "server", "s", "", "SQL Server hostname or IP address")
	rootCmd.PersistentFlags().StringVarP(&database, "database", "d", "", "Database name")
	rootCmd.PersistentFlags().StringVarP(&user, "user", "u", "", "Username for SQL authentication")
	rootCmd.PersistentFlags().StringVarP(&password, "password", "p", "", "Password for SQL authentication")
	rootCmd.PersistentFlags().BoolVarP(&trustedAuth, "trusted", "t", false, "Use Windows/Integrated authentication")
	rootCmd.PersistentFlags().IntVar(&port, "port", 1433, "SQL Server port")
	rootCmd.PersistentFlags().BoolVar(&trustCert, "trust-cert", false, "Trust server certificate (insecure)")
	rootCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "Show what would be executed without making changes")
}

// GetConnectionConfig builds a ConnectionConfig from the global flags
func GetConnectionConfig() *domain.ConnectionConfig {
	config := domain.NewConnectionConfig()
	config.Server = server
	config.Database = database
	config.User = user
	config.Password = password
	config.TrustedAuth = trustedAuth
	config.Port = port
	config.TrustServer = trustCert
	return config
}

// IsDryRun returns true if dry-run mode is enabled
func IsDryRun() bool {
	return dryRun
}
