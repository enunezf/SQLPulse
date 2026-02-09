// SQLPulse - SQL Server Administration CLI
//
// SQLPulse is a command-line tool for safe SQL Server administration.
// It implements a mandatory approval system for destructive operations.
package main

import (
	"github.com/enunezf/SQLPulse/internal/cli"
)

func main() {
	cli.Execute()
}
