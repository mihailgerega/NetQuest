package input

import "encoding/json"

// CreateTopologyInput — вход сохранения новой версии топологии.
// Data — документ как прислал клиент; проверяет его валидатор.
type CreateTopologyInput struct {
	Name string
	Data json.RawMessage
}
