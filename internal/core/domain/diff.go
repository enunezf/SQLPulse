package domain

import (
	"fmt"
	"strings"
)

// DiffType represents the type of difference found
type DiffType string

const (
	DiffAdded    DiffType = "ADDED"    // Object exists in target but not in source
	DiffRemoved  DiffType = "REMOVED"  // Object exists in source but not in target
	DiffModified DiffType = "MODIFIED" // Object exists in both but has differences
)

// DiffCategory represents the category of the difference
type DiffCategory string

const (
	DiffCategorySchema     DiffCategory = "SCHEMA"
	DiffCategoryTable      DiffCategory = "TABLE"
	DiffCategoryColumn     DiffCategory = "COLUMN"
	DiffCategoryIndex      DiffCategory = "INDEX"
	DiffCategoryForeignKey DiffCategory = "FOREIGN_KEY"
	DiffCategoryConstraint DiffCategory = "CONSTRAINT"
	DiffCategoryView       DiffCategory = "VIEW"
	DiffCategoryProcedure  DiffCategory = "PROCEDURE"
	DiffCategoryFunction   DiffCategory = "FUNCTION"
	DiffCategoryTrigger    DiffCategory = "TRIGGER"
)

// Difference represents a single difference between source and target
type Difference struct {
	Type        DiffType
	Category    DiffCategory
	ObjectName  string // Full object name (e.g., "dbo.Users")
	PropertyName string // Property that differs (e.g., "DataType", "MaxLength")
	SourceValue string // Value in source database
	TargetValue string // Value in target database
	Description string // Human-readable description
	MigrationSQL string // SQL to apply the change (from source to target)
}

// String returns a git-diff style representation
func (d *Difference) String() string {
	var prefix string
	switch d.Type {
	case DiffAdded:
		prefix = "\033[32m+\033[0m" // Green +
	case DiffRemoved:
		prefix = "\033[31m-\033[0m" // Red -
	case DiffModified:
		prefix = "\033[33m~\033[0m" // Yellow ~
	}

	return fmt.Sprintf("%s [%s] %s: %s", prefix, d.Category, d.ObjectName, d.Description)
}

// DiffResult contains all differences between two databases
type DiffResult struct {
	SourceDatabase string
	TargetDatabase string
	Differences    []Difference
	Summary        DiffSummary
}

// DiffSummary provides a summary count of differences
type DiffSummary struct {
	TotalDifferences int
	Added            int
	Removed          int
	Modified         int
	ByCategory       map[DiffCategory]int
}

// HasDifferences returns true if there are any differences
func (r *DiffResult) HasDifferences() bool {
	return len(r.Differences) > 0
}

// FilterByType returns differences of a specific type
func (r *DiffResult) FilterByType(diffType DiffType) []Difference {
	var filtered []Difference
	for _, d := range r.Differences {
		if d.Type == diffType {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// FilterByCategory returns differences of a specific category
func (r *DiffResult) FilterByCategory(category DiffCategory) []Difference {
	var filtered []Difference
	for _, d := range r.Differences {
		if d.Category == category {
			filtered = append(filtered, d)
		}
	}
	return filtered
}

// GenerateMigrationScript generates SQL to migrate from source to target
func (r *DiffResult) GenerateMigrationScript() string {
	var sb strings.Builder

	sb.WriteString("-- ============================================\n")
	sb.WriteString("-- Migration Script\n")
	sb.WriteString(fmt.Sprintf("-- From: %s\n", r.SourceDatabase))
	sb.WriteString(fmt.Sprintf("-- To:   %s\n", r.TargetDatabase))
	sb.WriteString("-- ============================================\n\n")

	// Group by category for organized output
	categories := []DiffCategory{
		DiffCategorySchema,
		DiffCategoryTable,
		DiffCategoryColumn,
		DiffCategoryIndex,
		DiffCategoryForeignKey,
		DiffCategoryConstraint,
		DiffCategoryView,
		DiffCategoryProcedure,
		DiffCategoryFunction,
		DiffCategoryTrigger,
	}

	for _, cat := range categories {
		diffs := r.FilterByCategory(cat)
		if len(diffs) == 0 {
			continue
		}

		sb.WriteString(fmt.Sprintf("-- %s Changes\n", cat))
		sb.WriteString("-- " + strings.Repeat("-", 40) + "\n\n")

		for _, d := range diffs {
			if d.MigrationSQL != "" {
				sb.WriteString(fmt.Sprintf("-- %s\n", d.Description))
				sb.WriteString(d.MigrationSQL)
				sb.WriteString("\nGO\n\n")
			}
		}
	}

	return sb.String()
}

// PrintGitStyle prints differences in git-diff style
func (r *DiffResult) PrintGitStyle() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("diff --sqlpulse a/%s b/%s\n", r.SourceDatabase, r.TargetDatabase))
	sb.WriteString("--- a/" + r.SourceDatabase + "\n")
	sb.WriteString("+++ b/" + r.TargetDatabase + "\n")
	sb.WriteString("\n")

	// Group by category
	currentCategory := DiffCategory("")
	for _, d := range r.Differences {
		if d.Category != currentCategory {
			currentCategory = d.Category
			sb.WriteString(fmt.Sprintf("\n@@ %s @@\n", currentCategory))
		}
		sb.WriteString(d.String() + "\n")
	}

	return sb.String()
}

// CalculateSummary calculates the summary statistics
func (r *DiffResult) CalculateSummary() {
	r.Summary = DiffSummary{
		ByCategory: make(map[DiffCategory]int),
	}

	for _, d := range r.Differences {
		r.Summary.TotalDifferences++
		r.Summary.ByCategory[d.Category]++

		switch d.Type {
		case DiffAdded:
			r.Summary.Added++
		case DiffRemoved:
			r.Summary.Removed++
		case DiffModified:
			r.Summary.Modified++
		}
	}
}

// DiffOptions configures the comparison behavior
type DiffOptions struct {
	IncludeTables      bool
	IncludeViews       bool
	IncludeProcedures  bool
	IncludeFunctions   bool
	IncludeTriggers    bool
	IncludeIndexes     bool
	IncludeForeignKeys bool
	IncludeConstraints bool
	SchemaFilter       []string
	TableFilter        []string
	IgnoreCollation    bool
	IgnoreWhitespace   bool // For procedure/view definitions
}

// DefaultDiffOptions returns default comparison options
func DefaultDiffOptions() *DiffOptions {
	return &DiffOptions{
		IncludeTables:      true,
		IncludeViews:       true,
		IncludeProcedures:  true,
		IncludeFunctions:   true,
		IncludeTriggers:    true,
		IncludeIndexes:     true,
		IncludeForeignKeys: true,
		IncludeConstraints: true,
		IgnoreCollation:    false,
		IgnoreWhitespace:   true,
	}
}
