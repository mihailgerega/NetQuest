// Package errs — доменные ошибки NetQuest API.
//
// Путь ошибки: repository / service создаёт sentinel-ошибку или типизированную
// ошибку → слои выше оборачивают её через %w → API-хендлер отдаёт как есть →
// httpx.WriteError по errors.Is / errors.As выбирает HTTP-статус и код
// (см. internal/httpx/error.go — единственное место, которое знает про HTTP-коды).
//
// Имя пакета errs, а не errors, чтобы не конфликтовать со стандартным errors.
//
// Тексты ошибок — часть HTTP-контракта: фронтенд показывает message из ответа
// {"error": {"code", "message", ...}}. Поэтому они на английском, как и были,
// и менять их нельзя без правки клиента.
package errs

import "errors"

// 404 not_found — запрошенного ресурса нет или он принадлежит другому пользователю.
// Чужой проект отвечает так же, как несуществующий: по ответу нельзя узнать,
// что ресурс с таким ID вообще есть.
var (
	// ErrUserNotFound — пользователя нет или он удалён. Источник — repository/user.
	ErrUserNotFound = errors.New("user not found")
	// ErrProjectNotFound — проекта нет, он удалён или принадлежит другому. Источник — repository/project.
	ErrProjectNotFound = errors.New("project not found")
	// ErrTopologyNotFound — версии топологии нет или проект чужой. Источник — repository/topology.
	ErrTopologyNotFound = errors.New("topology not found")
	// ErrSimulationNotFound — симуляции нет или проект чужой. Источник — repository/simulation.
	ErrSimulationNotFound = errors.New("simulation not found")
	// ErrQuestNotFound — квеста с таким ID или slug нет в каталоге. Источник — service/application/quest.
	ErrQuestNotFound = errors.New("quest not found")
	// ErrAttemptNotFound — попытки нет или она чужая. Источник — repository/quest.
	ErrAttemptNotFound = errors.New("quest attempt not found")
)

// 401 unauthorized — запрос не связан с пользователем или токен не подходит.
var (
	// ErrAuthenticationRequired — в ctx нет пользователя: хендлер вызван без RequireAuth.
	ErrAuthenticationRequired = errors.New("authentication is required")
	// ErrInvalidCredentials — неверный email или пароль. Одна ошибка на оба случая,
	// чтобы по ответу нельзя было узнать, зарегистрирован ли email.
	ErrInvalidCredentials = errors.New("email or password is invalid")
	// ErrRefreshTokenRequired — refresh-токена нет ни в теле, ни в cookie.
	ErrRefreshTokenRequired = errors.New("refresh token is required")
	// ErrRefreshTokenInvalid — токен не найден, отозван или истёк. Источник — repository/refresh_token.
	ErrRefreshTokenInvalid = errors.New("refresh token is invalid")
	// ErrBearerTokenRequired — нет заголовка Authorization. Источник — middleware.RequireAuth.
	ErrBearerTokenRequired = errors.New("authorization bearer token is required")
	// ErrBearerTokenInvalid — заголовок не в формате Bearer или JWT не прошёл проверку.
	ErrBearerTokenInvalid = errors.New("authorization bearer token is invalid")
	// ErrWebSocketTokenInvalid — JWT из ?token= или Authorization у WebSocket не прошёл проверку.
	ErrWebSocketTokenInvalid = errors.New("websocket token is invalid")
)

// 403 forbidden — действие понятно, но запрещено настройками сервиса.
var (
	// ErrDemoAuthDisabled — demo-вход выключен (DEMO_AUTH_ENABLED=false).
	ErrDemoAuthDisabled = errors.New("demo auth is disabled")
)

// 409 conflict — запрос корректен, но конфликтует с уже сохранёнными данными.
var (
	// ErrEmailTaken — email уже зарегистрирован. Источник — repository/user
	// (unique_violation 23505 на users.email).
	ErrEmailTaken = errors.New("email is already registered")
)

// 400 bad_request — тело запроса не удалось разобрать как один JSON-объект.
var (
	// ErrRequestBodyRequired — тело пустое. Источник — httpx.DecodeJSON.
	ErrRequestBodyRequired = errors.New("request body is required")
	// ErrRequestBodyNotSingleObject — после JSON-объекта в теле есть ещё данные.
	ErrRequestBodyNotSingleObject = errors.New("request body must contain a single JSON object")
)

// 500 internal_error — сбой самого сервиса. Клиенту — общая фраза, детали — в лог.
var (
	// ErrUnexpected — паника в хендлере, перехваченная middleware.Recover.
	ErrUnexpected = errors.New("unexpected server error")
)
