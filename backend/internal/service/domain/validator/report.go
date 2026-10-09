package validator

import (
	"fmt"

	"github.com/netquest/netquest/backend/internal/model"
)

// report копит проблемы по ходу проверки. Указатель передаётся во все
// проверки, и каждая дописывает свои — порядок записей совпадает с порядком
// проверок и элементов в документе.
type report struct {
	errors []model.ValidationError
}

// newReport создаёт пустой отчёт. Срез сразу не nil: для валидной топологии
// клиент получает "errors": [], а не "errors": null.
func newReport() *report {
	return &report{errors: []model.ValidationError{}}
}

// add записывает проблему по пути в документе.
func (r *report) add(path, message string) {
	r.errors = append(r.errors, model.ValidationError{Path: path, Message: message})
}

// addf — add с форматированием сообщения.
func (r *report) addf(path, format string, args ...any) {
	r.add(path, fmt.Sprintf(format, args...))
}

// hasErrors сообщает, найдена ли уже хоть одна проблема.
func (r *report) hasErrors() bool {
	return len(r.errors) > 0
}

// result собирает итог: топология валидна, только если проблем нет.
func (r *report) result() model.ValidationResult {
	return model.ValidationResult{
		Valid:  !r.hasErrors(),
		Errors: r.errors,
	}
}
