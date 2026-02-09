// Package sqlserver provides the SQL Server database adapter implementation.
package sqlserver

import (
	"context"
	"database/sql"
	"fmt"

	_ "github.com/microsoft/go-mssqldb" // SQL Server driver

	"github.com/enunezf/SQLPulse/internal/core/domain"
	"github.com/enunezf/SQLPulse/internal/security"
)

// Adapter implements the DatabasePort interface for SQL Server
type Adapter struct {
	config   *domain.ConnectionConfig
	db       *sql.DB
	approver security.Approver
}

// NewAdapter creates a new SQL Server adapter
func NewAdapter(config *domain.ConnectionConfig) *Adapter {
	return &Adapter{
		config:   config,
		approver: security.NewInteractiveApprover(),
	}
}

// Connect establishes a connection to SQL Server
func (a *Adapter) Connect(ctx context.Context) error {
	if err := a.config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	connStr := a.config.ConnectionString()

	db, err := sql.Open("sqlserver", connStr)
	if err != nil {
		return fmt.Errorf("failed to open connection: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)

	// Verify the connection
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return fmt.Errorf("failed to ping database: %w", err)
	}

	a.db = db
	return nil
}

// Ping verifies the connection is still alive
func (a *Adapter) Ping(ctx context.Context) error {
	if a.db == nil {
		return fmt.Errorf("not connected")
	}
	return a.db.PingContext(ctx)
}

// Close closes the database connection
func (a *Adapter) Close() error {
	if a.db != nil {
		return a.db.Close()
	}
	return nil
}

// GetServerInfo retrieves information about the connected SQL Server
func (a *Adapter) GetServerInfo(ctx context.Context) (*domain.ServerInfo, error) {
	if a.db == nil {
		return nil, fmt.Errorf("not connected")
	}

	info := &domain.ServerInfo{}

	// Get server version and edition
	query := `
		SELECT
			@@VERSION as Version,
			SERVERPROPERTY('Edition') as Edition,
			SERVERPROPERTY('ProductVersion') as ProductVersion,
			@@SERVERNAME as ServerName
	`

	row := a.db.QueryRowContext(ctx, query)
	err := row.Scan(&info.Version, &info.Edition, &info.ProductName, &info.ServerName)
	if err != nil {
		return nil, fmt.Errorf("failed to get server info: %w", err)
	}

	return info, nil
}

// ExecuteWithApproval executes SQL after getting user approval
func (a *Adapter) ExecuteWithApproval(ctx context.Context, sqlText string, level security.ApprovalLevel, operation string) error {
	if a.db == nil {
		return fmt.Errorf("not connected")
	}

	// Create approval request
	req := security.ApprovalRequest{
		Operation:     operation,
		SQL:           sqlText,
		Level:         level,
		ImpactSummary: "", // Can be populated by caller
	}

	// Request approval
	approved, err := a.approver.RequestApproval(req)
	if err != nil {
		return fmt.Errorf("approval error: %w", err)
	}

	if !approved {
		return fmt.Errorf("operation cancelled by user")
	}

	// Execute the SQL
	_, err = a.db.ExecContext(ctx, sqlText)
	if err != nil {
		return fmt.Errorf("execution failed: %w", err)
	}

	return nil
}

// SetApprover sets the approver to use for operations
func (a *Adapter) SetApprover(approver security.Approver) {
	a.approver = approver
}

// DB returns the underlying database connection for advanced usage
func (a *Adapter) DB() *sql.DB {
	return a.db
}
