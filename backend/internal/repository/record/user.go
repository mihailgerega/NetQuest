// Package record — строки таблиц PostgreSQL в том виде, в каком их читает pgx.
//
// Теги db совпадают с именами колонок в SELECT/RETURNING: pgx.RowToStructByName
// раскладывает строку по ним, поэтому каждый запрос репозитория перечисляет
// колонки явно, с псевдонимами (AS) для выражений вроде COALESCE и ::text.
// RowToStructByName строгий: лишняя или недостающая колонка — ошибка,
// а не молча нулевое поле.
//
// UUID-колонки читаются как текст (id::text): ID в сервисе — строки, и так их
// не нужно конвертировать в uuid.UUID и обратно.
package record

import "time"

// User — строка таблицы users. Колонки, которые могут быть NULL
// (display_name, avatar_url, password_hash), читаются через COALESCE в пустую строку.
type User struct {
	ID           string     `db:"id"`
	Email        string     `db:"email"`
	DisplayName  string     `db:"display_name"`
	AvatarURL    string     `db:"avatar_url"`
	Role         string     `db:"role"`
	CreatedAt    time.Time  `db:"created_at"`
	UpdatedAt    time.Time  `db:"updated_at"`
	DeletedAt    *time.Time `db:"deleted_at"` // NULL → nil
	PasswordHash string     `db:"password_hash"`
}

// RefreshToken — строка таблицы refresh_tokens.
type RefreshToken struct {
	ID        string     `db:"id"`
	UserID    string     `db:"user_id"`
	TokenHash string     `db:"token_hash"`
	UserAgent string     `db:"user_agent"`
	IPAddress string     `db:"ip_address"` // inet → текст; NULL → ""
	ExpiresAt time.Time  `db:"expires_at"`
	RevokedAt *time.Time `db:"revoked_at"` // NULL → nil
	CreatedAt time.Time  `db:"created_at"`
}
