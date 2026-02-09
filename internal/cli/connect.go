package cli

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/enunezf/SQLPulse/internal/adapters/sqlserver"
)

// connectCmd represents the connect command
var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Test connection to SQL Server",
	Long: `Test the connection to a SQL Server instance and display server information.

This command verifies that the provided credentials and connection settings
are valid by establishing a connection and querying basic server information.

Examples:
  # Connect using SQL authentication
  sqlpulse connect --server localhost --database master --user sa --password secret

  # Connect using Windows authentication
  sqlpulse connect --server myserver --database mydb --trusted

  # Connect with custom port
  sqlpulse connect --server myserver:1434 --database mydb --user sa --password secret --port 1434`,
	RunE: runConnect,
}

func init() {
	rootCmd.AddCommand(connectCmd)
}

func runConnect(cmd *cobra.Command, args []string) error {
	config := GetConnectionConfig()

	// Validate configuration
	if err := config.Validate(); err != nil {
		return fmt.Errorf("configuration error: %w", err)
	}

	fmt.Printf("Connecting to %s...\n", config.SafeString())

	// Create adapter and connect
	adapter := sqlserver.NewAdapter(config)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := adapter.Connect(ctx); err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer adapter.Close()

	fmt.Println("\033[32m✓ Connection successful!\033[0m")

	// Get and display server information
	info, err := adapter.GetServerInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to get server info: %w", err)
	}

	fmt.Println()
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("\033[1mServer Name:\033[0m    %s\n", info.ServerName)
	fmt.Printf("\033[1mEdition:\033[0m        %s\n", info.Edition)
	fmt.Printf("\033[1mProduct Version:\033[0m %s\n", info.ProductName)
	fmt.Println(strings.Repeat("─", 60))
	fmt.Println()
	fmt.Printf("\033[1mVersion Details:\033[0m\n%s\n", formatVersion(info.Version))

	return nil
}

// formatVersion formats the version string for better readability
func formatVersion(version string) string {
	// The version string from SQL Server is quite long,
	// we'll indent it for better readability
	lines := strings.Split(version, "\n")
	var formatted []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			formatted = append(formatted, "  "+line)
		}
	}
	return strings.Join(formatted, "\n")
}
