// Package migrator — применение SQL-миграций из каталога файлов *.up.sql.
//
// Свой мигратор вместо goose/migrate: формат простой и уже используется
// в развёрнутых базах. Файл "000001_init.up.sql" — версия 1; применённые версии
// записываются в таблицу schema_migrations. Миграции только вперёд: *.down.sql
// лежат рядом для ручного отката и мигратором не читаются.
package migrator

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

// upSuffix — расширение файлов миграций вперёд.
const upSuffix = ".up.sql"

// Migration — один файл миграции.
type Migration struct {
	Version int64
	Name    string
	Path    string
}

// Migrator применяет миграции из Dir к базе за Pool.
type Migrator struct {
	Pool *pgxpool.Pool
	Dir  string
}

// Up применяет все ещё не применённые миграции по возрастанию версии.
// Каждая миграция — в своей транзакции вместе с записью в schema_migrations:
// упавшая миграция не оставит ни частичных изменений, ни отметки о применении.
func (m Migrator) Up(ctx context.Context) error {
	if m.Pool == nil {
		return errors.New("не задан пул PostgreSQL")
	}

	migrations, applied, err := m.Status(ctx)
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

// Status возвращает все миграции каталога и множество применённых версий.
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

// ensureMigrationTable создаёт таблицу учёта миграций, если её ещё нет.
func (m Migrator) ensureMigrationTable(ctx context.Context) error {
	const query = `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version bigint PRIMARY KEY,
			name text NOT NULL,
			applied_at timestamptz NOT NULL DEFAULT now()
		)`

	if _, err := m.Pool.Exec(ctx, query); err != nil {
		return fmt.Errorf("создать таблицу schema_migrations: %w", err)
	}

	return nil
}

// appliedVersions читает версии уже применённых миграций.
func (m Migrator) appliedVersions(ctx context.Context) (map[int64]bool, error) {
	rows, err := m.Pool.Query(ctx, `SELECT version FROM schema_migrations`)
	if err != nil {
		return nil, fmt.Errorf("прочитать применённые миграции: %w", err)
	}
	defer rows.Close()

	applied := make(map[int64]bool)

	for rows.Next() {
		var version int64
		if err := rows.Scan(&version); err != nil {
			return nil, fmt.Errorf("прочитать версию миграции: %w", err)
		}

		applied[version] = true
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("обойти версии миграций: %w", err)
	}

	return applied, nil
}

// loadMigrations находит файлы *.up.sql в каталоге и сортирует их по версии.
func (m Migrator) loadMigrations() ([]Migration, error) {
	entries, err := os.ReadDir(m.Dir)
	if err != nil {
		return nil, fmt.Errorf("прочитать каталог миграций: %w", err)
	}

	migrations := make([]Migration, 0, len(entries))

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), upSuffix) {
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

// apply применяет одну миграцию в транзакции.
//
// Транзакция открывается командами BEGIN/COMMIT на одном соединении, а SQL
// файла выполняется через простой протокол (PgConn().Exec): в файле может быть
// несколько команд через «;», а расширенный протокол (pool.Exec с параметрами)
// принимает только одну.
func (m Migrator) apply(ctx context.Context, migration Migration) error {
	sqlBytes, err := os.ReadFile(migration.Path)
	if err != nil {
		return fmt.Errorf("прочитать миграцию %s: %w", migration.Name, err)
	}

	conn, err := m.Pool.Acquire(ctx)
	if err != nil {
		return fmt.Errorf("взять соединение PostgreSQL: %w", err)
	}
	defer conn.Release()

	if _, err := conn.Exec(ctx, "BEGIN"); err != nil {
		return fmt.Errorf("начать миграцию %s: %w", migration.Name, err)
	}

	// ROLLBACK — с context.WithoutCancel: откатить нужно, даже если ctx уже истёк.
	rollback := func() {
		_, _ = conn.Exec(context.WithoutCancel(ctx), "ROLLBACK") //nolint:gosec // G104: важна исходная ошибка миграции
	}

	if _, err := conn.Conn().PgConn().Exec(ctx, string(sqlBytes)).ReadAll(); err != nil {
		rollback()
		return fmt.Errorf("применить миграцию %s: %w", migration.Name, err)
	}

	if _, err := conn.Exec(ctx, `INSERT INTO schema_migrations (version, name) VALUES ($1, $2)`, migration.Version, migration.Name); err != nil {
		rollback()
		return fmt.Errorf("записать миграцию %s: %w", migration.Name, err)
	}

	if _, err := conn.Exec(ctx, "COMMIT"); err != nil {
		return fmt.Errorf("зафиксировать миграцию %s: %w", migration.Name, err)
	}

	return nil
}

// parseMigrationVersion берёт версию из имени файла: "000003_quests.up.sql" → 3.
func parseMigrationVersion(name string) (int64, error) {
	prefix, _, found := strings.Cut(name, "_")
	if !found {
		return 0, fmt.Errorf("имя миграции %q не в формате <версия>_<имя>.up.sql", name)
	}

	version, err := strconv.ParseInt(prefix, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("версия миграции %q не число: %w", name, err)
	}

	return version, nil
}
