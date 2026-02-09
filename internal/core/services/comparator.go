// Package services contains the business logic services.
package services

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/enunezf/SQLPulse/internal/core/domain"
)

// SchemaComparator compares two database schemas
type SchemaComparator struct {
	options *domain.DiffOptions
}

// NewSchemaComparator creates a new schema comparator
func NewSchemaComparator(options *domain.DiffOptions) *SchemaComparator {
	if options == nil {
		options = domain.DefaultDiffOptions()
	}
	return &SchemaComparator{options: options}
}

// Compare compares source and target schemas and returns the differences
func (c *SchemaComparator) Compare(source, target *domain.DatabaseSchema) *domain.DiffResult {
	result := &domain.DiffResult{
		SourceDatabase: source.DatabaseName,
		TargetDatabase: target.DatabaseName,
		Differences:    []domain.Difference{},
	}

	// Compare tables
	if c.options.IncludeTables {
		c.compareTables(source.Tables, target.Tables, result)
	}

	// Compare views
	if c.options.IncludeViews {
		c.compareViews(source.Views, target.Views, result)
	}

	// Compare stored procedures
	if c.options.IncludeProcedures {
		c.compareProcedures(source.StoredProcedures, target.StoredProcedures, result)
	}

	// Compare functions
	if c.options.IncludeFunctions {
		c.compareFunctions(source.Functions, target.Functions, result)
	}

	// Compare triggers
	if c.options.IncludeTriggers {
		c.compareTriggers(source.Triggers, target.Triggers, result)
	}

	result.CalculateSummary()
	return result
}

// compareTables compares table structures
func (c *SchemaComparator) compareTables(source, target []domain.Table, result *domain.DiffResult) {
	sourceMap := c.tablesToMap(source)
	targetMap := c.tablesToMap(target)

	// Find removed tables (in source but not in target)
	for name := range sourceMap {
		if _, exists := targetMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffRemoved,
				Category:    domain.DiffCategoryTable,
				ObjectName:  name,
				Description: fmt.Sprintf("Table [%s] exists in source but not in target", name),
				MigrationSQL: fmt.Sprintf("CREATE TABLE %s (\n    -- Copy structure from source\n)", name),
			})
		}
	}

	// Find added tables (in target but not in source)
	for name, tgtTable := range targetMap {
		if _, exists := sourceMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffAdded,
				Category:    domain.DiffCategoryTable,
				ObjectName:  name,
				Description: fmt.Sprintf("Table [%s] exists in target but not in source", name),
				MigrationSQL: fmt.Sprintf("DROP TABLE %s;", c.formatTableName(tgtTable)),
			})
		}
	}

	// Compare tables that exist in both
	for name, srcTable := range sourceMap {
		if tgtTable, exists := targetMap[name]; exists {
			c.compareTableStructure(srcTable, tgtTable, result)
		}
	}
}

// compareTableStructure compares two tables in detail
func (c *SchemaComparator) compareTableStructure(source, target domain.Table, result *domain.DiffResult) {
	tableName := c.formatTableName(source)

	// Compare columns
	c.compareColumns(tableName, source.Columns, target.Columns, result)

	// Compare indexes
	if c.options.IncludeIndexes {
		c.compareIndexes(tableName, source.Indexes, target.Indexes, result)
	}

	// Compare foreign keys
	if c.options.IncludeForeignKeys {
		c.compareForeignKeys(tableName, source.ForeignKeys, target.ForeignKeys, result)
	}

	// Compare check constraints
	if c.options.IncludeConstraints {
		c.compareCheckConstraints(tableName, source.CheckConstraints, target.CheckConstraints, result)
	}

	// Compare primary keys
	c.comparePrimaryKeys(tableName, source.PrimaryKey, target.PrimaryKey, result)
}

// compareColumns compares column definitions
func (c *SchemaComparator) compareColumns(tableName string, source, target []domain.Column, result *domain.DiffResult) {
	sourceMap := c.columnsToMap(source)
	targetMap := c.columnsToMap(target)

	// Find removed columns
	for name, srcCol := range sourceMap {
		if _, exists := targetMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffRemoved,
				Category:    domain.DiffCategoryColumn,
				ObjectName:  fmt.Sprintf("%s.%s", tableName, name),
				Description: fmt.Sprintf("Column [%s] missing in target", name),
				MigrationSQL: fmt.Sprintf("ALTER TABLE %s ADD %s;", tableName, srcCol.GenerateSQL()),
			})
		}
	}

	// Find added columns
	for name := range targetMap {
		if _, exists := sourceMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffAdded,
				Category:    domain.DiffCategoryColumn,
				ObjectName:  fmt.Sprintf("%s.%s", tableName, name),
				Description: fmt.Sprintf("Column [%s] exists only in target", name),
				MigrationSQL: fmt.Sprintf("ALTER TABLE %s DROP COLUMN [%s];", tableName, name),
			})
		}
	}

	// Compare columns that exist in both
	for name, srcCol := range sourceMap {
		if tgtCol, exists := targetMap[name]; exists {
			c.compareColumnDetails(tableName, srcCol, tgtCol, result)
		}
	}
}

// compareColumnDetails compares individual column properties
func (c *SchemaComparator) compareColumnDetails(tableName string, source, target domain.Column, result *domain.DiffResult) {
	colName := fmt.Sprintf("%s.%s", tableName, source.Name)

	// Compare data type
	if source.DataType != target.DataType {
		result.Differences = append(result.Differences, domain.Difference{
			Type:         domain.DiffModified,
			Category:     domain.DiffCategoryColumn,
			ObjectName:   colName,
			PropertyName: "DataType",
			SourceValue:  source.DataType,
			TargetValue:  target.DataType,
			Description:  fmt.Sprintf("Data type differs: %s vs %s", source.DataType, target.DataType),
			MigrationSQL: fmt.Sprintf("ALTER TABLE %s ALTER COLUMN [%s] %s;", tableName, source.Name, source.DataType),
		})
	}

	// Compare max length (for string types)
	if source.MaxLength != target.MaxLength {
		result.Differences = append(result.Differences, domain.Difference{
			Type:         domain.DiffModified,
			Category:     domain.DiffCategoryColumn,
			ObjectName:   colName,
			PropertyName: "MaxLength",
			SourceValue:  fmt.Sprintf("%d", source.MaxLength),
			TargetValue:  fmt.Sprintf("%d", target.MaxLength),
			Description:  fmt.Sprintf("Max length differs: %d vs %d", source.MaxLength, target.MaxLength),
		})
	}

	// Compare precision/scale (for numeric types)
	if source.Precision != target.Precision || source.Scale != target.Scale {
		result.Differences = append(result.Differences, domain.Difference{
			Type:         domain.DiffModified,
			Category:     domain.DiffCategoryColumn,
			ObjectName:   colName,
			PropertyName: "Precision/Scale",
			SourceValue:  fmt.Sprintf("(%d,%d)", source.Precision, source.Scale),
			TargetValue:  fmt.Sprintf("(%d,%d)", target.Precision, target.Scale),
			Description:  fmt.Sprintf("Precision/Scale differs: (%d,%d) vs (%d,%d)", source.Precision, source.Scale, target.Precision, target.Scale),
		})
	}

	// Compare nullability
	if source.IsNullable != target.IsNullable {
		srcNull := "NULL"
		tgtNull := "NULL"
		if !source.IsNullable {
			srcNull = "NOT NULL"
		}
		if !target.IsNullable {
			tgtNull = "NOT NULL"
		}
		result.Differences = append(result.Differences, domain.Difference{
			Type:         domain.DiffModified,
			Category:     domain.DiffCategoryColumn,
			ObjectName:   colName,
			PropertyName: "Nullability",
			SourceValue:  srcNull,
			TargetValue:  tgtNull,
			Description:  fmt.Sprintf("Nullability differs: %s vs %s", srcNull, tgtNull),
		})
	}

	// Compare identity
	if source.IsIdentity != target.IsIdentity {
		result.Differences = append(result.Differences, domain.Difference{
			Type:         domain.DiffModified,
			Category:     domain.DiffCategoryColumn,
			ObjectName:   colName,
			PropertyName: "Identity",
			SourceValue:  fmt.Sprintf("%v", source.IsIdentity),
			TargetValue:  fmt.Sprintf("%v", target.IsIdentity),
			Description:  fmt.Sprintf("Identity property differs"),
		})
	}

	// Compare collation (if not ignored)
	if !c.options.IgnoreCollation && source.Collation != target.Collation {
		result.Differences = append(result.Differences, domain.Difference{
			Type:         domain.DiffModified,
			Category:     domain.DiffCategoryColumn,
			ObjectName:   colName,
			PropertyName: "Collation",
			SourceValue:  source.Collation,
			TargetValue:  target.Collation,
			Description:  fmt.Sprintf("Collation differs: %s vs %s", source.Collation, target.Collation),
		})
	}
}

// compareIndexes compares index definitions
func (c *SchemaComparator) compareIndexes(tableName string, source, target []domain.Index, result *domain.DiffResult) {
	sourceMap := c.indexesToMap(source)
	targetMap := c.indexesToMap(target)

	for name, srcIdx := range sourceMap {
		if _, exists := targetMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffRemoved,
				Category:    domain.DiffCategoryIndex,
				ObjectName:  fmt.Sprintf("%s.%s", tableName, name),
				Description: fmt.Sprintf("Index [%s] missing in target", name),
				MigrationSQL: srcIdx.GenerateSQL() + ";",
			})
		}
	}

	for name := range targetMap {
		if _, exists := sourceMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffAdded,
				Category:    domain.DiffCategoryIndex,
				ObjectName:  fmt.Sprintf("%s.%s", tableName, name),
				Description: fmt.Sprintf("Index [%s] exists only in target", name),
				MigrationSQL: fmt.Sprintf("DROP INDEX [%s] ON %s;", name, tableName),
			})
		}
	}

	// Compare index properties for matching indexes
	for name, srcIdx := range sourceMap {
		if tgtIdx, exists := targetMap[name]; exists {
			c.compareIndexDetails(tableName, srcIdx, tgtIdx, result)
		}
	}
}

// compareIndexDetails compares individual index properties
func (c *SchemaComparator) compareIndexDetails(tableName string, source, target domain.Index, result *domain.DiffResult) {
	idxName := fmt.Sprintf("%s.%s", tableName, source.Name)

	if source.IsUnique != target.IsUnique {
		result.Differences = append(result.Differences, domain.Difference{
			Type:         domain.DiffModified,
			Category:     domain.DiffCategoryIndex,
			ObjectName:   idxName,
			PropertyName: "IsUnique",
			SourceValue:  fmt.Sprintf("%v", source.IsUnique),
			TargetValue:  fmt.Sprintf("%v", target.IsUnique),
			Description:  "Unique property differs",
		})
	}

	if source.IsClustered != target.IsClustered {
		result.Differences = append(result.Differences, domain.Difference{
			Type:         domain.DiffModified,
			Category:     domain.DiffCategoryIndex,
			ObjectName:   idxName,
			PropertyName: "IsClustered",
			SourceValue:  fmt.Sprintf("%v", source.IsClustered),
			TargetValue:  fmt.Sprintf("%v", target.IsClustered),
			Description:  "Clustered property differs",
		})
	}

	// Compare columns
	srcCols := c.indexColumnsToString(source.Columns)
	tgtCols := c.indexColumnsToString(target.Columns)
	if srcCols != tgtCols {
		result.Differences = append(result.Differences, domain.Difference{
			Type:         domain.DiffModified,
			Category:     domain.DiffCategoryIndex,
			ObjectName:   idxName,
			PropertyName: "Columns",
			SourceValue:  srcCols,
			TargetValue:  tgtCols,
			Description:  fmt.Sprintf("Index columns differ: [%s] vs [%s]", srcCols, tgtCols),
		})
	}
}

// compareForeignKeys compares foreign key definitions
func (c *SchemaComparator) compareForeignKeys(tableName string, source, target []domain.ForeignKey, result *domain.DiffResult) {
	sourceMap := c.foreignKeysToMap(source)
	targetMap := c.foreignKeysToMap(target)

	for name, srcFK := range sourceMap {
		if _, exists := targetMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffRemoved,
				Category:    domain.DiffCategoryForeignKey,
				ObjectName:  fmt.Sprintf("%s.%s", tableName, name),
				Description: fmt.Sprintf("Foreign key [%s] missing in target", name),
				MigrationSQL: srcFK.GenerateSQL() + ";",
			})
		}
	}

	for name := range targetMap {
		if _, exists := sourceMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffAdded,
				Category:    domain.DiffCategoryForeignKey,
				ObjectName:  fmt.Sprintf("%s.%s", tableName, name),
				Description: fmt.Sprintf("Foreign key [%s] exists only in target", name),
				MigrationSQL: fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT [%s];", tableName, name),
			})
		}
	}
}

// compareCheckConstraints compares check constraint definitions
func (c *SchemaComparator) compareCheckConstraints(tableName string, source, target []domain.CheckConstraint, result *domain.DiffResult) {
	sourceMap := c.checkConstraintsToMap(source)
	targetMap := c.checkConstraintsToMap(target)

	for name, srcCC := range sourceMap {
		if _, exists := targetMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffRemoved,
				Category:    domain.DiffCategoryConstraint,
				ObjectName:  fmt.Sprintf("%s.%s", tableName, name),
				Description: fmt.Sprintf("Check constraint [%s] missing in target", name),
				MigrationSQL: srcCC.GenerateSQL() + ";",
			})
		}
	}

	for name := range targetMap {
		if _, exists := sourceMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffAdded,
				Category:    domain.DiffCategoryConstraint,
				ObjectName:  fmt.Sprintf("%s.%s", tableName, name),
				Description: fmt.Sprintf("Check constraint [%s] exists only in target", name),
				MigrationSQL: fmt.Sprintf("ALTER TABLE %s DROP CONSTRAINT [%s];", tableName, name),
			})
		}
	}
}

// comparePrimaryKeys compares primary key definitions
func (c *SchemaComparator) comparePrimaryKeys(tableName string, source, target *domain.Index, result *domain.DiffResult) {
	if source == nil && target == nil {
		return
	}

	if source == nil && target != nil {
		result.Differences = append(result.Differences, domain.Difference{
			Type:        domain.DiffAdded,
			Category:    domain.DiffCategoryConstraint,
			ObjectName:  fmt.Sprintf("%s.PK", tableName),
			Description: "Primary key exists only in target",
		})
		return
	}

	if source != nil && target == nil {
		result.Differences = append(result.Differences, domain.Difference{
			Type:        domain.DiffRemoved,
			Category:    domain.DiffCategoryConstraint,
			ObjectName:  fmt.Sprintf("%s.PK", tableName),
			Description: "Primary key missing in target",
		})
		return
	}

	// Compare PK columns
	srcCols := c.indexColumnsToString(source.Columns)
	tgtCols := c.indexColumnsToString(target.Columns)
	if srcCols != tgtCols {
		result.Differences = append(result.Differences, domain.Difference{
			Type:         domain.DiffModified,
			Category:     domain.DiffCategoryConstraint,
			ObjectName:   fmt.Sprintf("%s.%s", tableName, source.Name),
			PropertyName: "Columns",
			SourceValue:  srcCols,
			TargetValue:  tgtCols,
			Description:  fmt.Sprintf("Primary key columns differ: [%s] vs [%s]", srcCols, tgtCols),
		})
	}
}

// compareViews compares view definitions
func (c *SchemaComparator) compareViews(source, target []domain.View, result *domain.DiffResult) {
	sourceMap := c.viewsToMap(source)
	targetMap := c.viewsToMap(target)

	for name := range sourceMap {
		if _, exists := targetMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffRemoved,
				Category:    domain.DiffCategoryView,
				ObjectName:  name,
				Description: fmt.Sprintf("View [%s] missing in target", name),
			})
		}
	}

	for name := range targetMap {
		if _, exists := sourceMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffAdded,
				Category:    domain.DiffCategoryView,
				ObjectName:  name,
				Description: fmt.Sprintf("View [%s] exists only in target", name),
			})
		}
	}

	// Compare definitions
	for name, srcView := range sourceMap {
		if tgtView, exists := targetMap[name]; exists {
			if !c.definitionsEqual(srcView.Definition, tgtView.Definition) {
				result.Differences = append(result.Differences, domain.Difference{
					Type:        domain.DiffModified,
					Category:    domain.DiffCategoryView,
					ObjectName:  name,
					Description: "View definition differs",
				})
			}
		}
	}
}

// compareProcedures compares stored procedure definitions
func (c *SchemaComparator) compareProcedures(source, target []domain.StoredProcedure, result *domain.DiffResult) {
	sourceMap := c.proceduresToMap(source)
	targetMap := c.proceduresToMap(target)

	for name := range sourceMap {
		if _, exists := targetMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffRemoved,
				Category:    domain.DiffCategoryProcedure,
				ObjectName:  name,
				Description: fmt.Sprintf("Procedure [%s] missing in target", name),
			})
		}
	}

	for name := range targetMap {
		if _, exists := sourceMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffAdded,
				Category:    domain.DiffCategoryProcedure,
				ObjectName:  name,
				Description: fmt.Sprintf("Procedure [%s] exists only in target", name),
			})
		}
	}

	for name, srcProc := range sourceMap {
		if tgtProc, exists := targetMap[name]; exists {
			if !c.definitionsEqual(srcProc.Definition, tgtProc.Definition) {
				result.Differences = append(result.Differences, domain.Difference{
					Type:        domain.DiffModified,
					Category:    domain.DiffCategoryProcedure,
					ObjectName:  name,
					Description: "Procedure definition differs",
				})
			}
		}
	}
}

// compareFunctions compares function definitions
func (c *SchemaComparator) compareFunctions(source, target []domain.Function, result *domain.DiffResult) {
	sourceMap := c.functionsToMap(source)
	targetMap := c.functionsToMap(target)

	for name := range sourceMap {
		if _, exists := targetMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffRemoved,
				Category:    domain.DiffCategoryFunction,
				ObjectName:  name,
				Description: fmt.Sprintf("Function [%s] missing in target", name),
			})
		}
	}

	for name := range targetMap {
		if _, exists := sourceMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffAdded,
				Category:    domain.DiffCategoryFunction,
				ObjectName:  name,
				Description: fmt.Sprintf("Function [%s] exists only in target", name),
			})
		}
	}

	for name, srcFunc := range sourceMap {
		if tgtFunc, exists := targetMap[name]; exists {
			if !c.definitionsEqual(srcFunc.Definition, tgtFunc.Definition) {
				result.Differences = append(result.Differences, domain.Difference{
					Type:        domain.DiffModified,
					Category:    domain.DiffCategoryFunction,
					ObjectName:  name,
					Description: "Function definition differs",
				})
			}
		}
	}
}

// compareTriggers compares trigger definitions
func (c *SchemaComparator) compareTriggers(source, target []domain.Trigger, result *domain.DiffResult) {
	sourceMap := c.triggersToMap(source)
	targetMap := c.triggersToMap(target)

	for name := range sourceMap {
		if _, exists := targetMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffRemoved,
				Category:    domain.DiffCategoryTrigger,
				ObjectName:  name,
				Description: fmt.Sprintf("Trigger [%s] missing in target", name),
			})
		}
	}

	for name := range targetMap {
		if _, exists := sourceMap[name]; !exists {
			result.Differences = append(result.Differences, domain.Difference{
				Type:        domain.DiffAdded,
				Category:    domain.DiffCategoryTrigger,
				ObjectName:  name,
				Description: fmt.Sprintf("Trigger [%s] exists only in target", name),
			})
		}
	}

	for name, srcTrig := range sourceMap {
		if tgtTrig, exists := targetMap[name]; exists {
			if !c.definitionsEqual(srcTrig.Definition, tgtTrig.Definition) {
				result.Differences = append(result.Differences, domain.Difference{
					Type:        domain.DiffModified,
					Category:    domain.DiffCategoryTrigger,
					ObjectName:  name,
					Description: "Trigger definition differs",
				})
			}
		}
	}
}

// Helper methods for creating maps

func (c *SchemaComparator) tablesToMap(tables []domain.Table) map[string]domain.Table {
	m := make(map[string]domain.Table)
	for _, t := range tables {
		m[c.formatTableName(t)] = t
	}
	return m
}

func (c *SchemaComparator) formatTableName(t domain.Table) string {
	return fmt.Sprintf("[%s].[%s]", t.SchemaName, t.Name)
}

func (c *SchemaComparator) columnsToMap(columns []domain.Column) map[string]domain.Column {
	m := make(map[string]domain.Column)
	for _, col := range columns {
		m[col.Name] = col
	}
	return m
}

func (c *SchemaComparator) indexesToMap(indexes []domain.Index) map[string]domain.Index {
	m := make(map[string]domain.Index)
	for _, idx := range indexes {
		m[idx.Name] = idx
	}
	return m
}

func (c *SchemaComparator) foreignKeysToMap(fks []domain.ForeignKey) map[string]domain.ForeignKey {
	m := make(map[string]domain.ForeignKey)
	for _, fk := range fks {
		m[fk.Name] = fk
	}
	return m
}

func (c *SchemaComparator) checkConstraintsToMap(ccs []domain.CheckConstraint) map[string]domain.CheckConstraint {
	m := make(map[string]domain.CheckConstraint)
	for _, cc := range ccs {
		m[cc.Name] = cc
	}
	return m
}

func (c *SchemaComparator) viewsToMap(views []domain.View) map[string]domain.View {
	m := make(map[string]domain.View)
	for _, v := range views {
		m[fmt.Sprintf("[%s].[%s]", v.SchemaName, v.Name)] = v
	}
	return m
}

func (c *SchemaComparator) proceduresToMap(procs []domain.StoredProcedure) map[string]domain.StoredProcedure {
	m := make(map[string]domain.StoredProcedure)
	for _, p := range procs {
		m[fmt.Sprintf("[%s].[%s]", p.SchemaName, p.Name)] = p
	}
	return m
}

func (c *SchemaComparator) functionsToMap(funcs []domain.Function) map[string]domain.Function {
	m := make(map[string]domain.Function)
	for _, f := range funcs {
		m[fmt.Sprintf("[%s].[%s]", f.SchemaName, f.Name)] = f
	}
	return m
}

func (c *SchemaComparator) triggersToMap(triggers []domain.Trigger) map[string]domain.Trigger {
	m := make(map[string]domain.Trigger)
	for _, t := range triggers {
		m[fmt.Sprintf("[%s].[%s].[%s]", t.SchemaName, t.TableName, t.Name)] = t
	}
	return m
}

func (c *SchemaComparator) indexColumnsToString(cols []domain.IndexColumn) string {
	var parts []string
	for _, col := range cols {
		part := col.Name
		if col.IsDescending {
			part += " DESC"
		}
		if col.IsIncluded {
			part += " (INCLUDE)"
		}
		parts = append(parts, part)
	}
	return strings.Join(parts, ", ")
}

// definitionsEqual compares two SQL definitions
func (c *SchemaComparator) definitionsEqual(source, target string) bool {
	if c.options.IgnoreWhitespace {
		source = c.normalizeWhitespace(source)
		target = c.normalizeWhitespace(target)
	}
	return source == target
}

// normalizeWhitespace removes extra whitespace for comparison
func (c *SchemaComparator) normalizeWhitespace(s string) string {
	// Replace multiple whitespace with single space
	re := regexp.MustCompile(`\s+`)
	s = re.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}
