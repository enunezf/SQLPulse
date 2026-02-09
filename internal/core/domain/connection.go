// Package domain contains the core domain models for SQLPulse.
package domain

import (
	"fmt"
	"net/url"
)

// ConnectionConfig holds the configuration for a database connection
type ConnectionConfig struct {
	Server       string // Server hostname or IP
	Port         int    // Port number (default 1433)
	Database     string // Database name
	User         string // Username for SQL authentication
	Password     string // Password for SQL authentication
	TrustedAuth  bool   // Use Windows/Integrated authentication
	Encrypt      bool   // Encrypt connection (default true)
	TrustServer  bool   // Trust server certificate
	AppName      string // Application name for connection
}

// NewConnectionConfig creates a new connection config with defaults
func NewConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		Port:       1433,
		Encrypt:    true,
		AppName:    "SQLPulse",
	}
}

// ConnectionString generates the MSSQL connection string
func (c *ConnectionConfig) ConnectionString() string {
	query := url.Values{}

	query.Add("database", c.Database)
	query.Add("app name", c.AppName)

	if c.Encrypt {
		query.Add("encrypt", "true")
	} else {
		query.Add("encrypt", "false")
	}

	if c.TrustServer {
		query.Add("TrustServerCertificate", "true")
	}

	var userInfo string
	if c.TrustedAuth {
		// Windows authentication
		query.Add("integrated security", "true")
		userInfo = ""
	} else {
		// SQL Server authentication
		userInfo = fmt.Sprintf("%s:%s@", url.PathEscape(c.User), url.PathEscape(c.Password))
	}

	return fmt.Sprintf("sqlserver://%s%s:%d?%s",
		userInfo,
		c.Server,
		c.Port,
		query.Encode(),
	)
}

// Validate checks if the connection config is valid
func (c *ConnectionConfig) Validate() error {
	if c.Server == "" {
		return fmt.Errorf("server is required")
	}

	if c.Database == "" {
		return fmt.Errorf("database is required")
	}

	if !c.TrustedAuth {
		if c.User == "" {
			return fmt.Errorf("user is required for SQL authentication")
		}
		if c.Password == "" {
			return fmt.Errorf("password is required for SQL authentication")
		}
	}

	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("port must be between 1 and 65535")
	}

	return nil
}

// SafeString returns the connection string with password masked
func (c *ConnectionConfig) SafeString() string {
	if c.TrustedAuth {
		return fmt.Sprintf("Server=%s:%d; Database=%s; TrustedAuth=true",
			c.Server, c.Port, c.Database)
	}
	return fmt.Sprintf("Server=%s:%d; Database=%s; User=%s; Password=***",
		c.Server, c.Port, c.Database, c.User)
}

// ServerInfo holds information about the connected server
type ServerInfo struct {
	Version     string // SQL Server version string
	Edition     string // SQL Server edition
	ProductName string // Product name
	ServerName  string // Server name
}
