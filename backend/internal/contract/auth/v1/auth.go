// Package authv1 — HTTP-контракт аутентификации: /api/v1/auth/*.
//
// Имена типов запросов значимы: encoding/json вставляет имя типа в текст
// ошибки разбора ("Go struct field RegisterRequest.email"), а этот текст уходит
// клиенту в ответе 400.
package authv1

import "time"

// RegisterRequest — тело POST /api/v1/auth/register.
type RegisterRequest struct {
	Email       string `json:"email"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
}

// LoginRequest — тело POST /api/v1/auth/login.
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// RefreshRequest — тело POST /api/v1/auth/refresh. Тело необязательно:
// без него токен берётся из httpOnly-cookie.
type RefreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// LogoutRequest — тело POST /api/v1/auth/logout; как и у refresh, необязательно.
type LogoutRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// User — профиль пользователя в ответах. Хеша пароля здесь нет намеренно.
type User struct {
	ID          string     `json:"id"`
	Email       string     `json:"email"`
	DisplayName string     `json:"displayName"`
	AvatarURL   string     `json:"avatarUrl,omitempty"`
	Role        string     `json:"role"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}

// AuthResponse — ответ входа и регистрации.
//
// Refresh-токена в теле нет: он уходит только в httpOnly-cookie, чтобы
// JavaScript страницы (и XSS в нём) не мог его прочитать.
type AuthResponse struct {
	User        User   `json:"user"`
	AccessToken string `json:"accessToken"`
	TokenType   string `json:"tokenType"`
	ExpiresIn   int64  `json:"expiresIn"` // срок жизни access-токена в секундах
}

// MeResponse — ответ GET /api/v1/auth/me.
type MeResponse struct {
	User User `json:"user"`
}
