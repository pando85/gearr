package migrations

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
)

var (
	ErrMigrationNotFound = fmt.Errorf("migration not found")
)

type Migration struct {
	Version   string
	Name      string
	Content   string
	AppliedAt *time.Time
}

type Migrator struct {
	db         *sql.DB
	migrations map[string]*Migration
}

var (
	//go:embed files/*.sql
	migrationFS embed.FS
)

func NewMigrator(db *sql.DB) (*Migrator, error) {
	m := &Migrator{
		db:         db,
		migrations: make(map[string]*Migration),
	}

	if err := m.loadMigrationFiles(); err != nil {
		return nil, fmt.Errorf("failed to load migration files: %w", err)
	}

	return m, nil
}

func (m *Migrator) loadMigrationFiles() error {
	entries, err := migrationFS.ReadDir("files")
	if err != nil {
		return fmt.Errorf("failed to read migration directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		content, err := migrationFS.ReadFile("files/" + entry.Name())
		if err != nil {
			return fmt.Errorf("failed to read migration file %s: %w", entry.Name(), err)
		}

		version := strings.TrimSuffix(entry.Name(), ".sql")
		m.migrations[version] = &Migration{
			Version: version,
			Name:    entry.Name(),
			Content: string(content),
		}
	}

	return nil
}

func (m *Migrator) ensureMigrationsTable(ctx context.Context) error {
	_, err := m.db.ExecContext(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	return err
}

func (m *Migrator) getAppliedMigrations(ctx context.Context) (map[string]time.Time, error) {
	applied := make(map[string]time.Time)

	rows, err := m.db.QueryContext(ctx, "SELECT version, applied_at FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var version string
		var appliedAt time.Time
		if err := rows.Scan(&version, &appliedAt); err != nil {
			return nil, err
		}
		applied[version] = appliedAt
	}

	return applied, rows.Err()
}

func (m *Migrator) Migrate(ctx context.Context) error {
	if err := m.ensureMigrationsTable(ctx); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	versions := make([]string, 0, len(m.migrations))
	for v := range m.migrations {
		versions = append(versions, v)
	}
	sort.Strings(versions)

	for _, version := range versions {
		if _, ok := applied[version]; ok {
			log.Debugf("migration %s already applied", version)
			continue
		}

		migration := m.migrations[version]
		log.Infof("applying migration %s", version)

		tx, err := m.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction for migration %s: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx, migration.Content); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to apply migration %s: %w", version, err)
		}

		if _, err := tx.ExecContext(ctx,
			"INSERT INTO schema_migrations (version) VALUES ($1)",
			version,
		); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to record migration %s: %w", version, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit migration %s: %w", version, err)
		}

		log.Infof("migration %s applied successfully", version)
	}

	return nil
}

func (m *Migrator) GetPendingMigrations(ctx context.Context) ([]*Migration, error) {
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	var pending []*Migration
	for version, migration := range m.migrations {
		if _, ok := applied[version]; !ok {
			pending = append(pending, migration)
		}
	}

	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Version < pending[j].Version
	})

	return pending, nil
}

func (m *Migrator) GetAppliedMigrations(ctx context.Context) ([]*Migration, error) {
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	var appliedList []*Migration
	for version, appliedAt := range applied {
		if migration, ok := m.migrations[version]; ok {
			migration.AppliedAt = &appliedAt
			appliedList = append(appliedList, migration)
		}
	}

	sort.Slice(appliedList, func(i, j int) bool {
		return appliedList[i].Version < appliedList[j].Version
	})

	return appliedList, nil
}
