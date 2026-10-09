package model

import "time"

// Видимость проекта. Хранится строкой в projects.visibility.
const (
	VisibilityPrivate  = "private"
	VisibilityPublic   = "public"
	VisibilityUnlisted = "unlisted"
)

// ProjectVisibilities — все допустимые значения видимости в порядке,
// в котором их перечисляет ошибка валидации.
var ProjectVisibilities = []string{VisibilityPrivate, VisibilityPublic, VisibilityUnlisted}

// Project — проект пользователя: контейнер для версий топологии и симуляций.
// Удаление мягкое: DeletedAt проставляется, строка остаётся.
type Project struct {
	ID          string
	OwnerID     string
	Name        string
	Description string
	Visibility  string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   *time.Time // nil — проект не удалён
}
