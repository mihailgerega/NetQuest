package config

// migrationsConfig — где CLI миграций ищет файлы *.up.sql.
// Путь относительный: в Docker-образе миграции лежат в /app/migrations,
// а рабочий каталог — /app; локально CLI запускается из backend/.
type migrationsConfig struct {
	Dir string
}

func loadMigrationsConfig(lookup LookupFunc) migrationsConfig {
	return migrationsConfig{
		Dir: getString(lookup, "MIGRATIONS_DIR", "migrations"),
	}
}
