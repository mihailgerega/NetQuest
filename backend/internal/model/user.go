// Package model — доменные модели NetQuest API.
//
// Две группы моделей:
//
//   - строки хранилища, которые наружу уходят через DTO (User, Project, Topology,
//     Simulation, Attempt): у них нет json-тегов, их JSON-форму задают
//     internal/contract и конвертеры internal/api/converter;
//   - JSON-документы предметной области (топология, сценарий и события симуляции,
//     каталог квестов, результаты проверок, замечания советника): их JSON-форма
//     и есть сама модель — она лежит в jsonb-колонках, вкладывается в другие
//     документы и без изменений отдаётся клиенту. Такие модели несут json-теги,
//     и переименовывать поля в тегах нельзя без миграции данных и правки фронтенда.
//
// Как модели лежат в PostgreSQL, описывает repository/record.
package model

import "time"

// RoleUser — роль обычного пользователя. Других ролей пока нет, но роль уже
// хранится в users.role и уходит в JWT, чтобы их можно было добавить.
const RoleUser = "user"

// User — доменная модель пользователя.
//
// PasswordHash — bcrypt-хеш, а не пароль; у demo-пользователя он пустой
// (входит без пароля). Наружу хеш не уходит: в dto для него нет поля.
type User struct {
	ID           string
	Email        string
	DisplayName  string
	AvatarURL    string
	Role         string
	PasswordHash string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time // nil — пользователь не удалён
}
