package storage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Migrator struct {
	Pool *pgxpool.Pool
	Dir  string
}

type Migration struct {
	Version int64
	Name    string
	Path    string
}

func (m Migrator) Up(ctx context.Context) error {
	if m.Pool == nil {
		return fmt.Errorf("postgres pool is required")
	}
	if err := m.ensureMigrationTable(ctx); err != nil {
		return err
	}

	applied, err := m.appliedVersions(ctx)
	if err != nil {
		return err
	}

	migrations, err := m.loadMigrations()
	if err != nil {
		return err
	}

	for _, migration := range migrations {
		if applied[migration.Version] {
			continue
		}
		if err := m.apply(ctx, migration); err != nil {
			return err
		}
	}

	return nil
}

func (m Migrator) Status(ctx context.Context) ([]Migration, map[int64]bool, error) {
	if err := m.ensureMigrationTable(ctx); err != nil {
		return nil, nil, err
	}
	applied, err := m.appliedVersions(ctx)
	if err != nil {
		return nil, nil, err
	}
	migrations, err := m.loadMigrations()
	if err != nil {
		return nil, nil, err
	}
	return migrations, applied, nil
}

func (m Migrator) ensureMigrationTable(ctx context.Context) error {
	_, err := m.Pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version bigint PRIMARY KEY,
			name text NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT now()
		)
	`)
	if err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}
	return nil
}

func (m Migrator) appliedVersions(ctx context.Context) (map[int64]bool, error) {
	rows, err := m.Pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("read applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[int64]bool)
	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("scan migration version: %w", err)
		}
		applied[version] = true
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate migration versions: %w", err)
	}
	return applied, nil
}

func (m Migrator) loadMigrations() ([]Migration, error) {
	entries, err := os.ReadDir(m.Dir)
	if err != nil {
		return nil, fmt.Errorf("read migrations dir: %w", err)
	}

	migrations := make([]Migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".up.sql") {
			continue
		}
		version, err := parseMigrationVersion(entry.Name())
		if err != nil {
			return nil, err
		}
		migrations = append(migrations, Migration{
			Version: version,
			Name:    entry.Name(),
			Path:    filepath.Join(m.Dir, entry.Name()),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func (m Migrator) apply(ctx context.Context, migration Migration) error {
	sqlBytes, err := os.ReadFile(migration.Path)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", migration.Name, err)
	}

	conn, err := m.Pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("acquire postgres connection: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "BEGIN"); err != nil {
		return fmt.Errorf("begin migration %s: %w", migration.Name, err)
	}

	if err := execSimpleSQL(ctx, conn, string(sqlBytes)); err != nil {
		_, _ = conn.Exec(context.Background(), "ROLLBACK")
		return fmt.Errorf("apply migration %s: %w", migration.Name, err)
	}

	if _, err := conn.Exec(ctx, `INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`, migration.Version, migration.Name); err != nil {
		_, _ = conn.Exec(context.Background(), "ROLLBACK")
		return fmt.Errorf("record migration %s: %w", migration.Name, err)
	}

	if _, err := conn.Exec(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("commit migration %s: %w", migration.Name, err)
	}
	return nil
}

func execSimpleSQL(ctx context.Context, conn *pgxpool.Conn, sql string) error {
	result := conn.Conn().PgConn().Exec(ctx, sql)
	_, err := result.ReadAll()
	return err
}

func parseMigrationVersion(name string) (int64, error) {
	parts := strings.SplitN(name, "_", 2)
	if len(parts) != 2 {
		return 0, fmt.Errorf("invalid migration name %q", name)
	}
	version, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid migration version %q: %w", name, err)
	}
	return version, nil
}
