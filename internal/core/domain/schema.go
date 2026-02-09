package domain

import (
	"fmt"
	"strings"
)

// ObjectType represents the type of database object
type ObjectType string

const (
	ObjectTypeTable           ObjectType = "TABLE"
	ObjectTypeView            ObjectType = "VIEW"
	ObjectTypeProcedure       ObjectType = "PROCEDURE"
	ObjectTypeFunction        ObjectType = "FUNCTION"
	ObjectTypeTrigger         ObjectType = "TRIGGER"
	ObjectTypeIndex           ObjectType = "INDEX"
	ObjectTypeConstraint      ObjectType = "CONSTRAINT"
	ObjectTypeSchema          ObjectType = "SCHEMA"
	ObjectTypeType            ObjectType = "TYPE"
	ObjectTypeSequence        ObjectType = "SEQUENCE"
	ObjectTypeSynonym         ObjectType = "SYNONYM"
)

// Column represents a table column
type Column struct {
	Name             string
	OrdinalPosition  int
	DataType         string
	MaxLength        int
	Precision        int
	Scale            int
	IsNullable       bool
	HasDefault       bool
	DefaultValue     string
	IsIdentity       bool
	IdentitySeed     int64
	IdentityIncrement int64
	IsComputed       bool
	ComputedDefinition string
	Collation        string
}

// GenerateSQL generates the column definition SQL
func (c *Column) GenerateSQL() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("[%s] ", c.Name))

	// Handle computed columns
	if c.IsComputed {
		sb.WriteString(fmt.Sprintf("AS %s", c.ComputedDefinition))
		return sb.String()
	}

	sb.WriteString(c.DataType)

	// Add length/precision/scale based on data type
	switch strings.ToUpper(c.DataType) {
	case "VARCHAR", "NVARCHAR", "CHAR", "NCHAR", "VARBINARY", "BINARY":
		if c.MaxLength == -1 {
			sb.WriteString("(MAX)")
		} else if strings.HasPrefix(strings.ToUpper(c.DataType), "N") {
			sb.WriteString(fmt.Sprintf("(%d)", c.MaxLength/2))
		} else {
			sb.WriteString(fmt.Sprintf("(%d)", c.MaxLength))
		}
	case "DECIMAL", "NUMERIC":
		sb.WriteString(fmt.Sprintf("(%d,%d)", c.Precision, c.Scale))
	case "DATETIME2", "DATETIMEOFFSET", "TIME":
		if c.Scale > 0 {
			sb.WriteString(fmt.Sprintf("(%d)", c.Scale))
		}
	}

	// Identity
	if c.IsIdentity {
		sb.WriteString(fmt.Sprintf(" IDENTITY(%d,%d)", c.IdentitySeed, c.IdentityIncrement))
	}

	// Nullability
	if c.IsNullable {
		sb.WriteString(" NULL")
	} else {
		sb.WriteString(" NOT NULL")
	}

	// Default value
	if c.HasDefault && c.DefaultValue != "" {
		sb.WriteString(fmt.Sprintf(" DEFAULT %s", c.DefaultValue))
	}

	return sb.String()
}

// IndexColumn represents a column in an index
type IndexColumn struct {
	Name       string
	Position   int
	IsDescending bool
	IsIncluded bool
}

// Index represents a table index
type Index struct {
	Name           string
	SchemaName     string
	TableName      string
	IsPrimaryKey   bool
	IsUnique       bool
	IsClustered    bool
	IsDisabled     bool
	FilterDefinition string
	Columns        []IndexColumn
}

// GenerateSQL generates the CREATE INDEX statement
func (i *Index) GenerateSQL() string {
	var sb strings.Builder

	if i.IsPrimaryKey {
		return "" // PKs are generated as constraints
	}

	sb.WriteString("CREATE ")
	if i.IsUnique {
		sb.WriteString("UNIQUE ")
	}
	if i.IsClustered {
		sb.WriteString("CLUSTERED ")
	} else {
		sb.WriteString("NONCLUSTERED ")
	}

	sb.WriteString(fmt.Sprintf("INDEX [%s] ON [%s].[%s] (\n", i.Name, i.SchemaName, i.TableName))

	// Key columns
	var keyCols []string
	var includeCols []string
	for _, col := range i.Columns {
		colDef := fmt.Sprintf("    [%s]", col.Name)
		if col.IsDescending {
			colDef += " DESC"
		}
		if col.IsIncluded {
			includeCols = append(includeCols, fmt.Sprintf("    [%s]", col.Name))
		} else {
			keyCols = append(keyCols, colDef)
		}
	}

	sb.WriteString(strings.Join(keyCols, ",\n"))
	sb.WriteString("\n)")

	// Include columns
	if len(includeCols) > 0 {
		sb.WriteString(" INCLUDE (\n")
		sb.WriteString(strings.Join(includeCols, ",\n"))
		sb.WriteString("\n)")
	}

	// Filter
	if i.FilterDefinition != "" {
		sb.WriteString(fmt.Sprintf(" WHERE %s", i.FilterDefinition))
	}

	return sb.String()
}

// ForeignKeyColumn represents a column mapping in a foreign key
type ForeignKeyColumn struct {
	ColumnName           string
	ReferencedColumnName string
}

// ForeignKey represents a foreign key constraint
type ForeignKey struct {
	Name                   string
	SchemaName             string
	TableName              string
	ReferencedSchemaName   string
	ReferencedTableName    string
	DeleteAction           string
	UpdateAction           string
	Columns                []ForeignKeyColumn
}

// GenerateSQL generates the foreign key constraint SQL
func (fk *ForeignKey) GenerateSQL() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("ALTER TABLE [%s].[%s] ADD CONSTRAINT [%s] FOREIGN KEY (\n",
		fk.SchemaName, fk.TableName, fk.Name))

	var cols []string
	var refCols []string
	for _, c := range fk.Columns {
		cols = append(cols, fmt.Sprintf("    [%s]", c.ColumnName))
		refCols = append(refCols, fmt.Sprintf("    [%s]", c.ReferencedColumnName))
	}

	sb.WriteString(strings.Join(cols, ",\n"))
	sb.WriteString(fmt.Sprintf("\n) REFERENCES [%s].[%s] (\n", fk.ReferencedSchemaName, fk.ReferencedTableName))
	sb.WriteString(strings.Join(refCols, ",\n"))
	sb.WriteString("\n)")

	if fk.DeleteAction != "" && fk.DeleteAction != "NO_ACTION" {
		sb.WriteString(fmt.Sprintf(" ON DELETE %s", strings.ReplaceAll(fk.DeleteAction, "_", " ")))
	}
	if fk.UpdateAction != "" && fk.UpdateAction != "NO_ACTION" {
		sb.WriteString(fmt.Sprintf(" ON UPDATE %s", strings.ReplaceAll(fk.UpdateAction, "_", " ")))
	}

	return sb.String()
}

// CheckConstraint represents a check constraint
type CheckConstraint struct {
	Name       string
	SchemaName string
	TableName  string
	Definition string
	IsDisabled bool
}

// GenerateSQL generates the check constraint SQL
func (cc *CheckConstraint) GenerateSQL() string {
	return fmt.Sprintf("ALTER TABLE [%s].[%s] ADD CONSTRAINT [%s] CHECK %s",
		cc.SchemaName, cc.TableName, cc.Name, cc.Definition)
}

// DefaultConstraint represents a default constraint
type DefaultConstraint struct {
	Name       string
	SchemaName string
	TableName  string
	ColumnName string
	Definition string
}

// Table represents a database table
type Table struct {
	SchemaName       string
	Name             string
	Columns          []Column
	PrimaryKey       *Index
	Indexes          []Index
	ForeignKeys      []ForeignKey
	CheckConstraints []CheckConstraint
}

// GenerateSQL generates the CREATE TABLE statement
func (t *Table) GenerateSQL() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("CREATE TABLE [%s].[%s] (\n", t.SchemaName, t.Name))

	// Columns
	var colDefs []string
	for _, col := range t.Columns {
		colDefs = append(colDefs, "    "+col.GenerateSQL())
	}

	// Primary Key constraint inline
	if t.PrimaryKey != nil && len(t.PrimaryKey.Columns) > 0 {
		var pkCols []string
		for _, col := range t.PrimaryKey.Columns {
			colDef := fmt.Sprintf("[%s]", col.Name)
			if col.IsDescending {
				colDef += " DESC"
			}
			pkCols = append(pkCols, colDef)
		}
		clustered := "CLUSTERED"
		if !t.PrimaryKey.IsClustered {
			clustered = "NONCLUSTERED"
		}
		pkDef := fmt.Sprintf("    CONSTRAINT [%s] PRIMARY KEY %s (%s)",
			t.PrimaryKey.Name, clustered, strings.Join(pkCols, ", "))
		colDefs = append(colDefs, pkDef)
	}

	sb.WriteString(strings.Join(colDefs, ",\n"))
	sb.WriteString("\n)")

	return sb.String()
}

// View represents a database view
type View struct {
	SchemaName string
	Name       string
	Definition string
}

// GenerateSQL returns the view definition
func (v *View) GenerateSQL() string {
	return v.Definition
}

// StoredProcedure represents a stored procedure
type StoredProcedure struct {
	SchemaName string
	Name       string
	Definition string
}

// GenerateSQL returns the procedure definition
func (sp *StoredProcedure) GenerateSQL() string {
	return sp.Definition
}

// Function represents a user-defined function
type Function struct {
	SchemaName string
	Name       string
	Definition string
	FuncType   string // SCALAR, TABLE, INLINE
}

// GenerateSQL returns the function definition
func (f *Function) GenerateSQL() string {
	return f.Definition
}

// Trigger represents a database trigger
type Trigger struct {
	SchemaName  string
	TableName   string
	Name        string
	Definition  string
	IsDisabled  bool
}

// GenerateSQL returns the trigger definition
func (tr *Trigger) GenerateSQL() string {
	return tr.Definition
}

// Schema represents a database schema
type Schema struct {
	Name  string
	Owner string
}

// GenerateSQL generates the CREATE SCHEMA statement
func (s *Schema) GenerateSQL() string {
	if s.Owner != "" {
		return fmt.Sprintf("CREATE SCHEMA [%s] AUTHORIZATION [%s]", s.Name, s.Owner)
	}
	return fmt.Sprintf("CREATE SCHEMA [%s]", s.Name)
}

// DatabaseSchema represents the complete database schema
type DatabaseSchema struct {
	DatabaseName     string
	Schemas          []Schema
	Tables           []Table
	Views            []View
	StoredProcedures []StoredProcedure
	Functions        []Function
	Triggers         []Trigger
}

// DumpOptions defines options for DDL extraction
type DumpOptions struct {
	IncludeTables       bool
	IncludeViews        bool
	IncludeProcedures   bool
	IncludeFunctions    bool
	IncludeTriggers     bool
	IncludeIndexes      bool
	IncludeForeignKeys  bool
	IncludeConstraints  bool
	SchemaFilter        []string // Filter by schema names
	TableFilter         []string // Filter by table names
	OutputFormat        string   // "sql", "json"
}

// DefaultDumpOptions returns default options with all objects included
func DefaultDumpOptions() *DumpOptions {
	return &DumpOptions{
		IncludeTables:      true,
		IncludeViews:       true,
		IncludeProcedures:  true,
		IncludeFunctions:   true,
		IncludeTriggers:    true,
		IncludeIndexes:     true,
		IncludeForeignKeys: true,
		IncludeConstraints: true,
		OutputFormat:       "sql",
	}
}
