// Package input — входные структуры методов сервисного слоя.
//
// API-конвертер собирает их из HTTP-запросов, а сервис проверяет по своим
// правилам: пустые значения, длины, допустимые варианты. Правила живут
// в сервисе, а не в API, потому что это правила предметной области,
// а не формат транспорта. Структура нужна, когда у метода несколько полей
// (часть — необязательные); простые параметры передаются как есть.
package input

// RegisterInput — вход Register: email, пароль в открытом виде и имя.
type RegisterInput struct {
	Email       string
	Password    string
	DisplayName string // пусто — имя возьмётся из email
}

// LoginInput — вход Login.
type LoginInput struct {
	Email    string
	Password string
}

// ClientInfo — откуда пришёл запрос входа. Сохраняется вместе с refresh-токеном,
// чтобы пользователь (в будущем — в списке сессий) видел, где он вошёл.
type ClientInfo struct {
	UserAgent string
	IPAddress string
}
