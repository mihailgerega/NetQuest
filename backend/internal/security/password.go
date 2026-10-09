// Package security — криптография аутентификации: хеширование паролей (bcrypt)
// и access-токены (JWT, HMAC-SHA256).
//
// Пакет не знает ни про HTTP, ни про базу: сервис аутентификации зовёт его,
// чтобы захешировать пароль и выпустить токен, а middleware — чтобы проверить токен.
package security

import (
	"errors"
	"fmt"

	"golang.org/x/crypto/bcrypt"
)

// minPasswordLength — минимум символов пароля (байт, как и считал прежний код).
const minPasswordLength = 12

// errPasswordTooShort — пароль короче minPasswordLength.
//
// Тексты ошибок HashPassword уходят клиенту как message ошибки 422
// (сервис делает из них ValidationError), поэтому они на английском.
var errPasswordTooShort = errors.New("password must be at least 12 characters")

// HashPassword проверяет длину пароля и хеширует его bcrypt с заданной сложностью.
//
// Соль bcrypt генерирует сам и записывает в результат вместе с cost
// ("$2a$12$<соль><хеш>"): одинаковые пароли двух пользователей дадут разные хеши,
// а VerifyPassword узнает cost из самого хеша. Пароль длиннее 72 байт bcrypt
// хешировать откажется — ошибка вернётся с префиксом "hash password: ".
func HashPassword(password string, cost int) (string, error) {
	if len(password) < minPasswordLength {
		return "", errPasswordTooShort
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}

	return string(hash), nil
}

// VerifyPassword сравнивает пароль с bcrypt-хешем. Испорченный хеш —
// тоже «не подошёл»: вход по нему невозможен в любом случае.
func VerifyPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
