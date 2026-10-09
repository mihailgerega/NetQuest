package validator

import (
	"bytes"
	"encoding/json"

	"github.com/netquest/netquest/backend/internal/model"
)

// decodeDocument разбирает сырой JSON в документ топологии.
// Возвращает ok=false, если дальше проверять нечего: тогда причина уже в отчёте.
//
// Разбор идёт в два шага. Сначала — в map[string]json.RawMessage: так можно
// отдельно сказать «нет поля nodes» и «nodes — не массив». Потом каждое поле
// раскладывается в свой срез.
func decodeDocument(data json.RawMessage, rep *report) (model.Document, bool) {
	var doc model.Document

	if len(bytes.TrimSpace(data)) == 0 {
		rep.add("$", "topology JSON is required")
		return doc, false
	}

	var raw map[string]json.RawMessage

	// UseNumber — числа верхнего уровня не превращаются в float64; на этом шаге
	// значения не нужны, важна только структура.
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()

	if err := decoder.Decode(&raw); err != nil {
		rep.addf("$", "invalid JSON structure: %v", err)
		return doc, false
	}

	// Второй Decode без ошибки значит, что после объекта есть ещё один
	// JSON-значение: "{...} {...}". Мусор после объекта второй Decode вернёт
	// ошибкой синтаксиса — такой документ проверка пропускает, как и раньше.
	if decoder.Decode(&struct{}{}) == nil {
		rep.add("$", "topology JSON must contain a single object")
		return doc, false
	}

	nodesRaw, hasNodes := raw["nodes"]
	linksRaw, hasLinks := raw["links"]

	if !hasNodes {
		rep.add("nodes", "nodes field is required")
	}

	if !hasLinks {
		rep.add("links", "links field is required")
	}

	if !hasNodes || !hasLinks {
		return doc, false
	}

	if err := json.Unmarshal(nodesRaw, &doc.Nodes); err != nil {
		rep.add("nodes", "nodes must be an array")
	}

	if err := json.Unmarshal(linksRaw, &doc.Links); err != nil {
		rep.add("links", "links must be an array")
	}

	return doc, !rep.hasErrors()
}
