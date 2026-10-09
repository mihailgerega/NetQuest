// Package httpx — общие HTTP-хелперы API: разбор JSON-тела, запись JSON-ответов,
// перевод ошибок в HTTP-коды (error.go), IP клиента и ID запроса.
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	errs "github.com/netquest/netquest/backend/internal/errors"
)

// DecodeJSON читает тело запроса как ровно один JSON-объект в dst.
//
// Правила строже, чем у json.Unmarshal: неизвестные поля — ошибка (опечатка
// в имени поля не пройдёт молча), второй объект после первого — ошибка.
// Тело читается не дальше maxBytes+1 байт: длиннее — обрежется и не разберётся.
// Все ошибки — 400 bad_request с текстом encoding/json.
func DecodeJSON(r *http.Request, dst any, maxBytes int64) error {
	reader := io.Reader(r.Body)
	if maxBytes > 0 {
		reader = io.LimitReader(r.Body, maxBytes+1)
	}

	decoder := json.NewDecoder(reader)
	decoder.DisallowUnknownFields()

	if err := decoder.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errs.ErrRequestBodyRequired
		}

		return &errs.InvalidJSONError{Cause: err}
	}

	if !errors.Is(decoder.Decode(&struct{}{}), io.EOF) {
		return errs.ErrRequestBodyNotSingleObject
	}

	return nil
}

// WriteJSON пишет ответ с кодом status и телом payload в JSON.
//
// Заголовки ставятся до WriteHeader: после него они уже ушли клиенту.
// Если payload не сериализуется, код уже отправлен — остаётся дописать
// в тело текст ошибки (net/http предупредит в логе о повторном WriteHeader).
func WriteJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(payload); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}
