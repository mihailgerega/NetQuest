// Package auth — сервисный слой аутентификации: регистрация, вход по паролю,
// demo-вход, обновление и отзыв токенов.
//
// Место в цепочке: api/auth/v1 → service/application/auth →
// {repository/user, repository/refresh_token, security (bcrypt, JWT)}.
//
// Схема токенов:
//   - access-токен — JWT на AccessTokenTTL (15 мин): его проверяет middleware
//     без похода в базу, поэтому отозвать его до истечения нельзя;
//   - refresh-токен — случайные 32 байта на RefreshTokenTTL (30 дней) в httpOnly-cookie;
//     в базе лежит его SHA-256. Одноразовый: Refresh отзывает старый и выдаёт новый.
package auth

import "time"

// Settings — параметры выдачи токенов и хеширования паролей из конфига.
type Settings struct {
	AccessTokenTTL   time.Duration
	RefreshTokenTTL  time.Duration
	PasswordHashCost int
	DemoAuthEnabled  bool
}

// service — сервис аутентификации. Неэкспортируемый тип: снаружи с ним
// работают через интерфейс AuthService из deps.go API-слоя.
type service struct {
	userRepository         UserRepository
	refreshTokenRepository RefreshTokenRepository
	tokenIssuer            AccessTokenIssuer
	settings               Settings
}

// New создаёт сервис аутентификации.
func New(
	userRepository UserRepository,
	refreshTokenRepository RefreshTokenRepository,
	tokenIssuer AccessTokenIssuer,
	settings Settings,
) *service {
	return &service{
		userRepository:         userRepository,
		refreshTokenRepository: refreshTokenRepository,
		tokenIssuer:            tokenIssuer,
		settings:               settings,
	}
}
