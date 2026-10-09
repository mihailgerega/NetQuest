package input

import "encoding/json"

// CheckQuestInput — вход проверки решения квеста.
type CheckQuestInput struct {
	Topology json.RawMessage
	Seed     *int64 // nil — фиксированный seed проверок
}
