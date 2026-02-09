package ports

import (
	"context"

	"github.com/enunezf/SQLPulse/internal/core/domain"
)

// SchemaPort defines the interface for schema extraction operations
type SchemaPort interface {
	// ExtractSchema extracts the complete database schema
	ExtractSchema(ctx context.Context, opts *domain.DumpOptions) (*domain.DatabaseSchema, error)

	// ExtractTables extracts table definitions
	ExtractTables(ctx context.Context, schemaFilter, tableFilter []string) ([]domain.Table, error)

	// ExtractViews extracts view definitions
	ExtractViews(ctx context.Context, schemaFilter []string) ([]domain.View, error)

	// ExtractProcedures extracts stored procedure definitions
	ExtractProcedures(ctx context.Context, schemaFilter []string) ([]domain.StoredProcedure, error)

	// ExtractFunctions extracts function definitions
	ExtractFunctions(ctx context.Context, schemaFilter []string) ([]domain.Function, error)

	// ExtractTriggers extracts trigger definitions
	ExtractTriggers(ctx context.Context, schemaFilter []string) ([]domain.Trigger, error)

	// ExtractSchemas extracts schema definitions
	ExtractSchemas(ctx context.Context) ([]domain.Schema, error)
}
