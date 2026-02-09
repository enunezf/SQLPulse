package sqlserver

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/enunezf/SQLPulse/internal/core/domain"
)

// SchemaExtractor extracts DDL from SQL Server
type SchemaExtractor struct {
	db *sql.DB
}

// NewSchemaExtractor creates a new schema extractor
func NewSchemaExtractor(db *sql.DB) *SchemaExtractor {
	return &SchemaExtractor{db: db}
}

// ExtractSchema extracts the complete database schema
func (e *SchemaExtractor) ExtractSchema(ctx context.Context, opts *domain.DumpOptions) (*domain.DatabaseSchema, error) {
	schema := &domain.DatabaseSchema{}

	// Get database name
	row := e.db.QueryRowContext(ctx, "SELECT DB_NAME()")
	if err := row.Scan(&schema.DatabaseName); err != nil {
		return nil, fmt.Errorf("failed to get database name: %w", err)
	}

	var err error

	// Extract schemas
	schema.Schemas, err = e.ExtractSchemas(ctx)
	if err != nil {
		return nil, err
	}

	// Extract tables with indexes and constraints
	if opts.IncludeTables {
		schema.Tables, err = e.ExtractTables(ctx, opts.SchemaFilter, opts.TableFilter)
		if err != nil {
			return nil, err
		}
	}

	// Extract views
	if opts.IncludeViews {
		schema.Views, err = e.ExtractViews(ctx, opts.SchemaFilter)
		if err != nil {
			return nil, err
		}
	}

	// Extract stored procedures
	if opts.IncludeProcedures {
		schema.StoredProcedures, err = e.ExtractProcedures(ctx, opts.SchemaFilter)
		if err != nil {
			return nil, err
		}
	}

	// Extract functions
	if opts.IncludeFunctions {
		schema.Functions, err = e.ExtractFunctions(ctx, opts.SchemaFilter)
		if err != nil {
			return nil, err
		}
	}

	// Extract triggers
	if opts.IncludeTriggers {
		schema.Triggers, err = e.ExtractTriggers(ctx, opts.SchemaFilter)
		if err != nil {
			return nil, err
		}
	}

	return schema, nil
}

// ExtractSchemas extracts schema definitions
func (e *SchemaExtractor) ExtractSchemas(ctx context.Context) ([]domain.Schema, error) {
	query := `
		SELECT
			s.name AS schema_name,
			p.name AS owner_name
		FROM sys.schemas s
		INNER JOIN sys.database_principals p ON s.principal_id = p.principal_id
		WHERE s.schema_id < 16384
			AND s.name NOT IN ('dbo', 'guest', 'INFORMATION_SCHEMA', 'sys', 'db_owner',
				'db_accessadmin', 'db_securityadmin', 'db_ddladmin', 'db_backupoperator',
				'db_datareader', 'db_datawriter', 'db_denydatareader', 'db_denydatawriter')
		ORDER BY s.name
	`

	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query schemas: %w", err)
	}
	defer rows.Close()

	var schemas []domain.Schema
	for rows.Next() {
		var s domain.Schema
		if err := rows.Scan(&s.Name, &s.Owner); err != nil {
			return nil, fmt.Errorf("failed to scan schema: %w", err)
		}
		schemas = append(schemas, s)
	}

	return schemas, rows.Err()
}

// ExtractTables extracts table definitions with columns, PKs, and indexes
func (e *SchemaExtractor) ExtractTables(ctx context.Context, schemaFilter, tableFilter []string) ([]domain.Table, error) {
	// Build filter conditions
	whereClause := "WHERE t.is_ms_shipped = 0"
	if len(schemaFilter) > 0 {
		whereClause += fmt.Sprintf(" AND s.name IN ('%s')", strings.Join(schemaFilter, "','"))
	}
	if len(tableFilter) > 0 {
		whereClause += fmt.Sprintf(" AND t.name IN ('%s')", strings.Join(tableFilter, "','"))
	}

	// Query tables
	query := fmt.Sprintf(`
		SELECT
			s.name AS schema_name,
			t.name AS table_name
		FROM sys.tables t
		INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
		%s
		ORDER BY s.name, t.name
	`, whereClause)

	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query tables: %w", err)
	}
	defer rows.Close()

	var tables []domain.Table
	for rows.Next() {
		var t domain.Table
		if err := rows.Scan(&t.SchemaName, &t.Name); err != nil {
			return nil, fmt.Errorf("failed to scan table: %w", err)
		}
		tables = append(tables, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Extract columns, PKs, indexes, and FKs for each table
	for i := range tables {
		tables[i].Columns, err = e.extractColumns(ctx, tables[i].SchemaName, tables[i].Name)
		if err != nil {
			return nil, err
		}

		tables[i].PrimaryKey, err = e.extractPrimaryKey(ctx, tables[i].SchemaName, tables[i].Name)
		if err != nil {
			return nil, err
		}

		tables[i].Indexes, err = e.extractIndexes(ctx, tables[i].SchemaName, tables[i].Name)
		if err != nil {
			return nil, err
		}

		tables[i].ForeignKeys, err = e.extractForeignKeys(ctx, tables[i].SchemaName, tables[i].Name)
		if err != nil {
			return nil, err
		}

		tables[i].CheckConstraints, err = e.extractCheckConstraints(ctx, tables[i].SchemaName, tables[i].Name)
		if err != nil {
			return nil, err
		}
	}

	return tables, nil
}

// extractColumns extracts column definitions for a table
func (e *SchemaExtractor) extractColumns(ctx context.Context, schemaName, tableName string) ([]domain.Column, error) {
	query := `
		SELECT
			c.name AS column_name,
			c.column_id AS ordinal_position,
			TYPE_NAME(c.user_type_id) AS data_type,
			c.max_length,
			c.precision,
			c.scale,
			c.is_nullable,
			CASE WHEN dc.definition IS NOT NULL THEN 1 ELSE 0 END AS has_default,
			ISNULL(dc.definition, '') AS default_value,
			c.is_identity,
			ISNULL(CAST(ic.seed_value AS BIGINT), 0) AS identity_seed,
			ISNULL(CAST(ic.increment_value AS BIGINT), 0) AS identity_increment,
			c.is_computed,
			ISNULL(cc.definition, '') AS computed_definition,
			ISNULL(c.collation_name, '') AS collation_name
		FROM sys.columns c
		INNER JOIN sys.tables t ON c.object_id = t.object_id
		INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
		LEFT JOIN sys.default_constraints dc ON c.default_object_id = dc.object_id
		LEFT JOIN sys.identity_columns ic ON c.object_id = ic.object_id AND c.column_id = ic.column_id
		LEFT JOIN sys.computed_columns cc ON c.object_id = cc.object_id AND c.column_id = cc.column_id
		WHERE s.name = @p1 AND t.name = @p2
		ORDER BY c.column_id
	`

	rows, err := e.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query columns for %s.%s: %w", schemaName, tableName, err)
	}
	defer rows.Close()

	var columns []domain.Column
	for rows.Next() {
		var c domain.Column
		if err := rows.Scan(
			&c.Name, &c.OrdinalPosition, &c.DataType, &c.MaxLength,
			&c.Precision, &c.Scale, &c.IsNullable, &c.HasDefault, &c.DefaultValue,
			&c.IsIdentity, &c.IdentitySeed, &c.IdentityIncrement,
			&c.IsComputed, &c.ComputedDefinition, &c.Collation,
		); err != nil {
			return nil, fmt.Errorf("failed to scan column: %w", err)
		}
		columns = append(columns, c)
	}

	return columns, rows.Err()
}

// extractPrimaryKey extracts the primary key for a table
func (e *SchemaExtractor) extractPrimaryKey(ctx context.Context, schemaName, tableName string) (*domain.Index, error) {
	query := `
		SELECT
			i.name AS index_name,
			i.type_desc AS index_type
		FROM sys.indexes i
		INNER JOIN sys.tables t ON i.object_id = t.object_id
		INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
		WHERE s.name = @p1 AND t.name = @p2 AND i.is_primary_key = 1
	`

	var pk domain.Index
	var indexType string
	err := e.db.QueryRowContext(ctx, query, schemaName, tableName).Scan(&pk.Name, &indexType)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query primary key for %s.%s: %w", schemaName, tableName, err)
	}

	pk.SchemaName = schemaName
	pk.TableName = tableName
	pk.IsPrimaryKey = true
	pk.IsUnique = true
	pk.IsClustered = indexType == "CLUSTERED"

	// Get PK columns
	pk.Columns, err = e.extractIndexColumns(ctx, schemaName, tableName, pk.Name)
	if err != nil {
		return nil, err
	}

	return &pk, nil
}

// extractIndexes extracts non-PK indexes for a table
func (e *SchemaExtractor) extractIndexes(ctx context.Context, schemaName, tableName string) ([]domain.Index, error) {
	query := `
		SELECT
			i.name AS index_name,
			i.is_unique,
			CASE WHEN i.type = 1 THEN 1 ELSE 0 END AS is_clustered,
			i.is_disabled,
			ISNULL(i.filter_definition, '') AS filter_definition
		FROM sys.indexes i
		INNER JOIN sys.tables t ON i.object_id = t.object_id
		INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
		WHERE s.name = @p1 AND t.name = @p2
			AND i.is_primary_key = 0
			AND i.type > 0
			AND i.name IS NOT NULL
		ORDER BY i.name
	`

	rows, err := e.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query indexes for %s.%s: %w", schemaName, tableName, err)
	}
	defer rows.Close()

	var indexes []domain.Index
	for rows.Next() {
		var idx domain.Index
		idx.SchemaName = schemaName
		idx.TableName = tableName
		if err := rows.Scan(&idx.Name, &idx.IsUnique, &idx.IsClustered, &idx.IsDisabled, &idx.FilterDefinition); err != nil {
			return nil, fmt.Errorf("failed to scan index: %w", err)
		}

		// Get index columns
		idx.Columns, err = e.extractIndexColumns(ctx, schemaName, tableName, idx.Name)
		if err != nil {
			return nil, err
		}

		indexes = append(indexes, idx)
	}

	return indexes, rows.Err()
}

// extractIndexColumns extracts columns for an index
func (e *SchemaExtractor) extractIndexColumns(ctx context.Context, schemaName, tableName, indexName string) ([]domain.IndexColumn, error) {
	query := `
		SELECT
			c.name AS column_name,
			ic.key_ordinal AS position,
			ic.is_descending_key,
			ic.is_included_column
		FROM sys.index_columns ic
		INNER JOIN sys.indexes i ON ic.object_id = i.object_id AND ic.index_id = i.index_id
		INNER JOIN sys.columns c ON ic.object_id = c.object_id AND ic.column_id = c.column_id
		INNER JOIN sys.tables t ON i.object_id = t.object_id
		INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
		WHERE s.name = @p1 AND t.name = @p2 AND i.name = @p3
		ORDER BY ic.is_included_column, ic.key_ordinal
	`

	rows, err := e.db.QueryContext(ctx, query, schemaName, tableName, indexName)
	if err != nil {
		return nil, fmt.Errorf("failed to query index columns: %w", err)
	}
	defer rows.Close()

	var columns []domain.IndexColumn
	for rows.Next() {
		var c domain.IndexColumn
		if err := rows.Scan(&c.Name, &c.Position, &c.IsDescending, &c.IsIncluded); err != nil {
			return nil, fmt.Errorf("failed to scan index column: %w", err)
		}
		columns = append(columns, c)
	}

	return columns, rows.Err()
}

// extractForeignKeys extracts foreign key constraints for a table
func (e *SchemaExtractor) extractForeignKeys(ctx context.Context, schemaName, tableName string) ([]domain.ForeignKey, error) {
	query := `
		SELECT
			fk.name AS fk_name,
			SCHEMA_NAME(fk.schema_id) AS schema_name,
			OBJECT_NAME(fk.parent_object_id) AS table_name,
			SCHEMA_NAME(rt.schema_id) AS referenced_schema,
			rt.name AS referenced_table,
			fk.delete_referential_action_desc,
			fk.update_referential_action_desc
		FROM sys.foreign_keys fk
		INNER JOIN sys.tables t ON fk.parent_object_id = t.object_id
		INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
		INNER JOIN sys.tables rt ON fk.referenced_object_id = rt.object_id
		WHERE s.name = @p1 AND t.name = @p2
		ORDER BY fk.name
	`

	rows, err := e.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query foreign keys for %s.%s: %w", schemaName, tableName, err)
	}
	defer rows.Close()

	var fks []domain.ForeignKey
	for rows.Next() {
		var fk domain.ForeignKey
		if err := rows.Scan(&fk.Name, &fk.SchemaName, &fk.TableName,
			&fk.ReferencedSchemaName, &fk.ReferencedTableName,
			&fk.DeleteAction, &fk.UpdateAction); err != nil {
			return nil, fmt.Errorf("failed to scan foreign key: %w", err)
		}

		// Get FK columns
		fk.Columns, err = e.extractForeignKeyColumns(ctx, fk.Name)
		if err != nil {
			return nil, err
		}

		fks = append(fks, fk)
	}

	return fks, rows.Err()
}

// extractForeignKeyColumns extracts column mappings for a foreign key
func (e *SchemaExtractor) extractForeignKeyColumns(ctx context.Context, fkName string) ([]domain.ForeignKeyColumn, error) {
	query := `
		SELECT
			COL_NAME(fkc.parent_object_id, fkc.parent_column_id) AS column_name,
			COL_NAME(fkc.referenced_object_id, fkc.referenced_column_id) AS referenced_column
		FROM sys.foreign_key_columns fkc
		INNER JOIN sys.foreign_keys fk ON fkc.constraint_object_id = fk.object_id
		WHERE fk.name = @p1
		ORDER BY fkc.constraint_column_id
	`

	rows, err := e.db.QueryContext(ctx, query, fkName)
	if err != nil {
		return nil, fmt.Errorf("failed to query FK columns: %w", err)
	}
	defer rows.Close()

	var columns []domain.ForeignKeyColumn
	for rows.Next() {
		var c domain.ForeignKeyColumn
		if err := rows.Scan(&c.ColumnName, &c.ReferencedColumnName); err != nil {
			return nil, fmt.Errorf("failed to scan FK column: %w", err)
		}
		columns = append(columns, c)
	}

	return columns, rows.Err()
}

// extractCheckConstraints extracts check constraints for a table
func (e *SchemaExtractor) extractCheckConstraints(ctx context.Context, schemaName, tableName string) ([]domain.CheckConstraint, error) {
	query := `
		SELECT
			cc.name AS constraint_name,
			SCHEMA_NAME(t.schema_id) AS schema_name,
			t.name AS table_name,
			cc.definition,
			cc.is_disabled
		FROM sys.check_constraints cc
		INNER JOIN sys.tables t ON cc.parent_object_id = t.object_id
		INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
		WHERE s.name = @p1 AND t.name = @p2
		ORDER BY cc.name
	`

	rows, err := e.db.QueryContext(ctx, query, schemaName, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to query check constraints: %w", err)
	}
	defer rows.Close()

	var constraints []domain.CheckConstraint
	for rows.Next() {
		var c domain.CheckConstraint
		if err := rows.Scan(&c.Name, &c.SchemaName, &c.TableName, &c.Definition, &c.IsDisabled); err != nil {
			return nil, fmt.Errorf("failed to scan check constraint: %w", err)
		}
		constraints = append(constraints, c)
	}

	return constraints, rows.Err()
}

// ExtractViews extracts view definitions
func (e *SchemaExtractor) ExtractViews(ctx context.Context, schemaFilter []string) ([]domain.View, error) {
	whereClause := "WHERE v.is_ms_shipped = 0"
	if len(schemaFilter) > 0 {
		whereClause += fmt.Sprintf(" AND s.name IN ('%s')", strings.Join(schemaFilter, "','"))
	}

	query := fmt.Sprintf(`
		SELECT
			s.name AS schema_name,
			v.name AS view_name,
			ISNULL(m.definition, '') AS definition
		FROM sys.views v
		INNER JOIN sys.schemas s ON v.schema_id = s.schema_id
		LEFT JOIN sys.sql_modules m ON v.object_id = m.object_id
		%s
		ORDER BY s.name, v.name
	`, whereClause)

	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query views: %w", err)
	}
	defer rows.Close()

	var views []domain.View
	for rows.Next() {
		var v domain.View
		if err := rows.Scan(&v.SchemaName, &v.Name, &v.Definition); err != nil {
			return nil, fmt.Errorf("failed to scan view: %w", err)
		}
		views = append(views, v)
	}

	return views, rows.Err()
}

// ExtractProcedures extracts stored procedure definitions
func (e *SchemaExtractor) ExtractProcedures(ctx context.Context, schemaFilter []string) ([]domain.StoredProcedure, error) {
	whereClause := "WHERE p.is_ms_shipped = 0"
	if len(schemaFilter) > 0 {
		whereClause += fmt.Sprintf(" AND s.name IN ('%s')", strings.Join(schemaFilter, "','"))
	}

	query := fmt.Sprintf(`
		SELECT
			s.name AS schema_name,
			p.name AS proc_name,
			ISNULL(m.definition, '') AS definition
		FROM sys.procedures p
		INNER JOIN sys.schemas s ON p.schema_id = s.schema_id
		LEFT JOIN sys.sql_modules m ON p.object_id = m.object_id
		%s
		ORDER BY s.name, p.name
	`, whereClause)

	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query procedures: %w", err)
	}
	defer rows.Close()

	var procs []domain.StoredProcedure
	for rows.Next() {
		var p domain.StoredProcedure
		if err := rows.Scan(&p.SchemaName, &p.Name, &p.Definition); err != nil {
			return nil, fmt.Errorf("failed to scan procedure: %w", err)
		}
		procs = append(procs, p)
	}

	return procs, rows.Err()
}

// ExtractFunctions extracts function definitions
func (e *SchemaExtractor) ExtractFunctions(ctx context.Context, schemaFilter []string) ([]domain.Function, error) {
	whereClause := "WHERE o.is_ms_shipped = 0"
	if len(schemaFilter) > 0 {
		whereClause += fmt.Sprintf(" AND s.name IN ('%s')", strings.Join(schemaFilter, "','"))
	}

	query := fmt.Sprintf(`
		SELECT
			s.name AS schema_name,
			o.name AS func_name,
			ISNULL(m.definition, '') AS definition,
			CASE o.type
				WHEN 'FN' THEN 'SCALAR'
				WHEN 'IF' THEN 'INLINE'
				WHEN 'TF' THEN 'TABLE'
				ELSE 'UNKNOWN'
			END AS func_type
		FROM sys.objects o
		INNER JOIN sys.schemas s ON o.schema_id = s.schema_id
		LEFT JOIN sys.sql_modules m ON o.object_id = m.object_id
		%s AND o.type IN ('FN', 'IF', 'TF')
		ORDER BY s.name, o.name
	`, whereClause)

	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query functions: %w", err)
	}
	defer rows.Close()

	var funcs []domain.Function
	for rows.Next() {
		var f domain.Function
		if err := rows.Scan(&f.SchemaName, &f.Name, &f.Definition, &f.FuncType); err != nil {
			return nil, fmt.Errorf("failed to scan function: %w", err)
		}
		funcs = append(funcs, f)
	}

	return funcs, rows.Err()
}

// ExtractTriggers extracts trigger definitions
func (e *SchemaExtractor) ExtractTriggers(ctx context.Context, schemaFilter []string) ([]domain.Trigger, error) {
	whereClause := "WHERE tr.is_ms_shipped = 0"
	if len(schemaFilter) > 0 {
		whereClause += fmt.Sprintf(" AND s.name IN ('%s')", strings.Join(schemaFilter, "','"))
	}

	query := fmt.Sprintf(`
		SELECT
			s.name AS schema_name,
			t.name AS table_name,
			tr.name AS trigger_name,
			ISNULL(m.definition, '') AS definition,
			tr.is_disabled
		FROM sys.triggers tr
		INNER JOIN sys.tables t ON tr.parent_id = t.object_id
		INNER JOIN sys.schemas s ON t.schema_id = s.schema_id
		LEFT JOIN sys.sql_modules m ON tr.object_id = m.object_id
		%s
		ORDER BY s.name, t.name, tr.name
	`, whereClause)

	rows, err := e.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query triggers: %w", err)
	}
	defer rows.Close()

	var triggers []domain.Trigger
	for rows.Next() {
		var tr domain.Trigger
		if err := rows.Scan(&tr.SchemaName, &tr.TableName, &tr.Name, &tr.Definition, &tr.IsDisabled); err != nil {
			return nil, fmt.Errorf("failed to scan trigger: %w", err)
		}
		triggers = append(triggers, tr)
	}

	return triggers, rows.Err()
}
