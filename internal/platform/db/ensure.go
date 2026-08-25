package db

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/jackc/pgx/v5"
)

// EnsureDatabase connects to the DB server, verifies credentials, and creates
// the target database when it is missing. SQLite is a no-op: the file is
// created on open. MySQL and Postgres require the DSN to name a database.
func EnsureDatabase(driver, dsn string) error {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case DriverSQLite:
		return nil
	case DriverMySQL:
		return ensureMySQLDatabase(dsn)
	case DriverPostgres:
		return ensurePostgresDatabase(dsn)
	default:
		return fmt.Errorf("db: unknown driver %q", driver)
	}
}

func ensureMySQLDatabase(dsn string) error {
	cfg, err := gomysql.ParseDSN(dsn)
	if err != nil {
		return fmt.Errorf("db: parse mysql dsn: %w", err)
	}
	dbName := strings.TrimSpace(cfg.DBName)
	if dbName == "" {
		return fmt.Errorf("db: mysql dsn missing database name")
	}
	server := *cfg
	server.DBName = ""
	conn, err := sql.Open("mysql", server.FormatDSN())
	if err != nil {
		return fmt.Errorf("db: open mysql server: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if err := conn.Ping(); err != nil {
		return fmt.Errorf("db: connect mysql server: %w", err)
	}
	if _, err := conn.Exec("CREATE DATABASE IF NOT EXISTS " + quoteMySQLIdent(dbName)); err != nil {
		return fmt.Errorf("db: create mysql database: %w", err)
	}
	return nil
}

func ensurePostgresDatabase(dsn string) error {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("db: parse postgres dsn: %w", err)
	}
	dbName := strings.TrimSpace(cfg.Database)
	if dbName == "" {
		return fmt.Errorf("db: postgres dsn missing database name")
	}
	ctx := context.Background()
	var lastErr error
	for _, maint := range []string{"postgres", "template1"} {
		server := cfg.Copy()
		server.Database = maint
		conn, err := pgx.ConnectConfig(ctx, server)
		if err != nil {
			lastErr = err
			continue
		}
		var exists bool
		if err := conn.QueryRow(ctx,
			"SELECT EXISTS(SELECT 1 FROM pg_database WHERE datname = $1)",
			dbName,
		).Scan(&exists); err != nil {
			_ = conn.Close(ctx)
			lastErr = err
			continue
		}
		if !exists {
			if _, err := conn.Exec(ctx, "CREATE DATABASE "+quotePostgresIdent(dbName)); err != nil {
				_ = conn.Close(ctx)
				lastErr = fmt.Errorf("db: create postgres database: %w", err)
				continue
			}
		}
		_ = conn.Close(ctx)
		return nil
	}
	return fmt.Errorf("db: connect postgres server: %v", lastErr)
}

func quoteMySQLIdent(s string) string {
	return "`" + strings.ReplaceAll(s, "`", "``") + "`"
}

func quotePostgresIdent(s string) string {
	return `"` + strings.ReplaceAll(s, `"`, `""`) + `"`
}

// DropDatabase removes the target database. SQLite is a no-op. Used by
// integration tests to clean up auto-created databases.
func DropDatabase(driver, dsn string) error {
	switch strings.ToLower(strings.TrimSpace(driver)) {
	case DriverMySQL:
		return dropMySQLDatabase(dsn)
	case DriverPostgres:
		return dropPostgresDatabase(dsn)
	default:
		return nil
	}
}

func dropMySQLDatabase(dsn string) error {
	cfg, err := gomysql.ParseDSN(dsn)
	if err != nil {
		return fmt.Errorf("db: parse mysql dsn: %w", err)
	}
	server := *cfg
	server.DBName = ""
	conn, err := sql.Open("mysql", server.FormatDSN())
	if err != nil {
		return fmt.Errorf("db: open mysql server: %w", err)
	}
	defer func() { _ = conn.Close() }()
	if _, err := conn.Exec("DROP DATABASE IF EXISTS " + quoteMySQLIdent(cfg.DBName)); err != nil {
		return fmt.Errorf("db: drop mysql database: %w", err)
	}
	return nil
}

func dropPostgresDatabase(dsn string) error {
	cfg, err := pgx.ParseConfig(dsn)
	if err != nil {
		return fmt.Errorf("db: parse postgres dsn: %w", err)
	}
	ctx := context.Background()
	server := cfg.Copy()
	server.Database = "postgres"
	conn, err := pgx.ConnectConfig(ctx, server)
	if err != nil {
		return fmt.Errorf("db: connect postgres server: %w", err)
	}
	defer func() { _ = conn.Close(ctx) }()
	if _, err := conn.Exec(ctx, "DROP DATABASE "+quotePostgresIdent(cfg.Database)); err != nil {
		return fmt.Errorf("db: drop postgres database: %w", err)
	}
	return nil
}
