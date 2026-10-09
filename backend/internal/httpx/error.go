package httpx

import (
	"errors"
	"log/slog"
	"net/http"

	commonv1 "github.com/netquest/netquest/backend/internal/contract/common/v1"
	errs "github.com/netquest/netquest/backend/internal/errors"
)

// Коды ошибок в ответе (error.code) — машиночитаемая категория для клиента.
const (
	codeBadRequest       = "bad_request"
	codeUnauthorized     = "unauthorized"
	codeForbidden        = "forbidden"
	codeNotFound         = "not_found"
	codeConflict         = "conflict"
	codeValidationFailed = "validation_failed"
	codeRateLimited      = "rate_limited"
	codeInternal         = "internal_error"
)

// rateLimitWindow — окно лимитера для подсказки клиенту в details.
const rateLimitWindow = "1m"

// sentinelStatuses — sentinel-ошибка → HTTP-статус и код.
//
// Срез, а не map: errors.Is проверяет цепочку %w, поэтому ключом map ошибка
// быть не может — нужен перебор.
var sentinelStatuses = []struct {
	err    error
	status int
	code   string
}{
	{errs.ErrRequestBodyRequired, http.StatusBadRequest, codeBadRequest},
	{errs.ErrRequestBodyNotSingleObject, http.StatusBadRequest, codeBadRequest},

	{errs.ErrAuthenticationRequired, http.StatusUnauthorized, codeUnauthorized},
	{errs.ErrInvalidCredentials, http.StatusUnauthorized, codeUnauthorized},
	{errs.ErrRefreshTokenRequired, http.StatusUnauthorized, codeUnauthorized},
	{errs.ErrRefreshTokenInvalid, http.StatusUnauthorized, codeUnauthorized},
	{errs.ErrBearerTokenRequired, http.StatusUnauthorized, codeUnauthorized},
	{errs.ErrBearerTokenInvalid, http.StatusUnauthorized, codeUnauthorized},
	{errs.ErrWebSocketTokenInvalid, http.StatusUnauthorized, codeUnauthorized},

	{errs.ErrDemoAuthDisabled, http.StatusForbidden, codeForbidden},

	{errs.ErrUserNotFound, http.StatusNotFound, codeNotFound},
	{errs.ErrProjectNotFound, http.StatusNotFound, codeNotFound},
	{errs.ErrTopologyNotFound, http.StatusNotFound, codeNotFound},
	{errs.ErrSimulationNotFound, http.StatusNotFound, codeNotFound},
	{errs.ErrQuestNotFound, http.StatusNotFound, codeNotFound},
	{errs.ErrAttemptNotFound, http.StatusNotFound, codeNotFound},

	{errs.ErrEmailTaken, http.StatusConflict, codeConflict},

	{errs.ErrUnexpected, http.StatusInternalServerError, codeInternal},
}

// apiError — ошибка в том виде, в каком она уходит клиенту.
type apiError struct {
	status  int
	code    string
	message string
	details any
}

// WriteError отвечает ошибкой в едином формате {"error": {code, message, details, requestId}}.
// Это единственное место в сервисе, которое знает про HTTP-коды ошибок.
func WriteError(w http.ResponseWriter, r *http.Request, err error) {
	apiErr := mapError(err)

	// Клиенту детали 500 не показываем, поэтому обязательно пишем их в лог.
	// Паника (ErrUnexpected) уже залогирована middleware.Recover вместе со стеком.
	if apiErr.status == http.StatusInternalServerError && !errors.Is(err, errs.ErrUnexpected) {
		slog.ErrorContext(r.Context(), "необработанная ошибка", "request_id", RequestID(r.Context()), "error", err)
	}

	WriteJSON(w, apiErr.status, commonv1.ErrorResponse{
		Error: commonv1.ErrorBody{
			Code:      apiErr.code,
			Details:   apiErr.details,
			Message:   apiErr.message,
			RequestID: RequestID(r.Context()),
		},
	})
}

// mapError выбирает статус, код и текст по ошибке.
//
// Текст sentinel-ошибки берётся из неё самой, а не из всей цепочки: обёртки
// вроде «получить проект: ...» — внутренние подробности, клиенту — ровно
// текст из контракта. У типизированных ошибок текст и подробности свои.
func mapError(err error) apiError {
	var (
		validationErr *errs.ValidationError
		invalidJSON   *errs.InvalidJSONError
		rateLimitErr  *errs.RateLimitError
	)

	switch {
	case errors.As(err, &validationErr):
		return apiError{http.StatusUnprocessableEntity, codeValidationFailed, validationErr.Message, validationErr.Details}
	case errors.As(err, &invalidJSON):
		return apiError{http.StatusBadRequest, codeBadRequest, invalidJSON.Error(), nil}
	case errors.As(err, &rateLimitErr):
		return apiError{http.StatusTooManyRequests, codeRateLimited, rateLimitErr.Error(), map[string]any{
			"limit":  rateLimitErr.Limit,
			"window": rateLimitWindow,
		}}
	}

	for _, mapping := range sentinelStatuses {
		if errors.Is(err, mapping.err) {
			return apiError{mapping.status, mapping.code, mapping.err.Error(), nil}
		}
	}

	// Сюда попадают сбои PostgreSQL, Redis и прочие ошибки, которых клиент не
	// мог избежать, а также отмена запроса.
	return apiError{http.StatusInternalServerError, codeInternal, errs.ErrUnexpected.Error(), nil}
}
