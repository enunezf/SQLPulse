// Package ports defines the interfaces (ports) for the hexagonal architecture.
package ports

import (
	"context"

	"github.com/enunezf/SQLPulse/internal/core/domain"
	"github.com/enunezf/SQLPulse/internal/security"
)

// DatabasePort defines the interface for database operations
type DatabasePort interface {
	// Connect establishes a connection to the database
	Connect(ctx context.Context) error

	// Ping verifies the connection is still alive
	Ping(ctx context.Context) error

	// Close closes the database connection
	Close() error

	// GetServerInfo retrieves information about the connected server
	GetServerInfo(ctx context.Context) (*domain.ServerInfo, error)

	// ExecuteWithApproval executes SQL after getting user approval
	ExecuteWithApproval(ctx context.Context, sql string, level security.ApprovalLevel, operation string) error

	// SetApprover sets the approver to use for operations
	SetApprover(approver security.Approver)
}
