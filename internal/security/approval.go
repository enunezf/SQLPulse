// Package security provides the approval system for SQL operations.
// This implements a dry-run mechanism that requires user confirmation
// before executing potentially dangerous operations.
package security

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// ApprovalLevel defines the risk level of an operation
type ApprovalLevel int

const (
	// ReadOnly operations don't require confirmation
	ReadOnly ApprovalLevel = iota
	// Modification operations require simple y/n confirmation
	Modification
	// Destructive operations require confirmation + typing a word
	Destructive
)

// String returns the string representation of the approval level
func (a ApprovalLevel) String() string {
	switch a {
	case ReadOnly:
		return "ReadOnly"
	case Modification:
		return "Modification"
	case Destructive:
		return "Destructive"
	default:
		return "Unknown"
	}
}

// ApprovalRequest represents a request for user approval
type ApprovalRequest struct {
	Operation     string        // Description of the operation
	SQL           string        // SQL script to execute
	Level         ApprovalLevel // Risk level
	ImpactSummary string        // Summary of the impact
}

// Approver defines the interface for approval handling
type Approver interface {
	RequestApproval(req ApprovalRequest) (bool, error)
}

// InteractiveApprover implements approval via terminal interaction
type InteractiveApprover struct {
	reader *bufio.Reader
}

// NewInteractiveApprover creates a new interactive approver
func NewInteractiveApprover() *InteractiveApprover {
	return &InteractiveApprover{
		reader: bufio.NewReader(os.Stdin),
	}
}

// RequestApproval prompts the user for confirmation based on the operation level
func (a *InteractiveApprover) RequestApproval(req ApprovalRequest) (bool, error) {
	switch req.Level {
	case ReadOnly:
		// No confirmation needed for read-only operations
		return true, nil

	case Modification:
		return a.requestSimpleConfirmation(req)

	case Destructive:
		return a.requestStrictConfirmation(req)

	default:
		return false, fmt.Errorf("unknown approval level: %d", req.Level)
	}
}

// requestSimpleConfirmation asks for y/n confirmation
func (a *InteractiveApprover) requestSimpleConfirmation(req ApprovalRequest) (bool, error) {
	a.displayOperationDetails(req)

	fmt.Print("\n\033[33m⚠ This operation will modify data.\033[0m\n")
	fmt.Print("Do you want to proceed? [y/N]: ")

	response, err := a.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes", nil
}

// requestStrictConfirmation asks for confirmation + typing a specific word
func (a *InteractiveApprover) requestStrictConfirmation(req ApprovalRequest) (bool, error) {
	a.displayOperationDetails(req)

	fmt.Print("\n\033[31m⛔ WARNING: This is a DESTRUCTIVE operation!\033[0m\n")
	fmt.Print("\033[31mThis action cannot be undone.\033[0m\n\n")

	confirmWord := "CONFIRM"
	fmt.Printf("Type '%s' to proceed: ", confirmWord)

	response, err := a.reader.ReadString('\n')
	if err != nil {
		return false, fmt.Errorf("failed to read response: %w", err)
	}

	response = strings.TrimSpace(response)
	if response != confirmWord {
		fmt.Println("\n\033[31mOperation cancelled. Confirmation word did not match.\033[0m")
		return false, nil
	}

	return true, nil
}

// displayOperationDetails shows the operation information to the user
func (a *InteractiveApprover) displayOperationDetails(req ApprovalRequest) {
	fmt.Println("\n" + strings.Repeat("─", 60))
	fmt.Printf("\033[1mOperation:\033[0m %s\n", req.Operation)
	fmt.Printf("\033[1mRisk Level:\033[0m %s\n", req.Level)

	if req.ImpactSummary != "" {
		fmt.Printf("\033[1mImpact:\033[0m %s\n", req.ImpactSummary)
	}

	if req.SQL != "" {
		fmt.Println("\n\033[1mSQL to execute:\033[0m")
		fmt.Println("\033[36m" + req.SQL + "\033[0m")
	}

	fmt.Println(strings.Repeat("─", 60))
}

// AutoApprover always approves operations (for testing or automation)
type AutoApprover struct {
	approve bool
}

// NewAutoApprover creates an auto-approver with the specified behavior
func NewAutoApprover(approve bool) *AutoApprover {
	return &AutoApprover{approve: approve}
}

// RequestApproval returns the configured approval decision
func (a *AutoApprover) RequestApproval(req ApprovalRequest) (bool, error) {
	return a.approve, nil
}

// DryRunApprover displays what would happen but never approves
type DryRunApprover struct{}

// NewDryRunApprover creates a new dry-run approver
func NewDryRunApprover() *DryRunApprover {
	return &DryRunApprover{}
}

// RequestApproval displays the operation but always returns false
func (a *DryRunApprover) RequestApproval(req ApprovalRequest) (bool, error) {
	fmt.Println("\n\033[34m[DRY-RUN MODE]\033[0m The following operation would be executed:")
	fmt.Println(strings.Repeat("─", 60))
	fmt.Printf("\033[1mOperation:\033[0m %s\n", req.Operation)
	fmt.Printf("\033[1mRisk Level:\033[0m %s\n", req.Level)

	if req.ImpactSummary != "" {
		fmt.Printf("\033[1mImpact:\033[0m %s\n", req.ImpactSummary)
	}

	if req.SQL != "" {
		fmt.Println("\n\033[1mSQL that would execute:\033[0m")
		fmt.Println("\033[36m" + req.SQL + "\033[0m")
	}

	fmt.Println(strings.Repeat("─", 60))
	fmt.Println("\033[34mNo changes were made (dry-run mode).\033[0m")

	return false, nil
}
