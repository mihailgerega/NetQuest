// Package validator — доменный сервис проверки топологии перед сохранением
// и симуляцией.
//
// Место в цепочке: его вызывают service/application/topology (при сохранении
// версии), движок симуляции (перед расчётом), чекер квестов и советник.
// Валидатор ничего не знает ни про HTTP, ни про PostgreSQL: на входе сырой JSON,
// на выходе model.ValidationResult со списком проблем.
//
// Проверки идут слоями, и каждый следующий слой запускается, только если
// предыдущий дал документ, который вообще можно разбирать дальше:
//
//	JSON-объект → поля nodes и links → массивы узлов и каналов
//	→ лимиты размера → ID и типы узлов → концы каналов
//	→ настройки узлов (пул Load Balancer, открытые порты Server)
package validator

import (
	"encoding/json"

	"github.com/netquest/netquest/backend/internal/model"
)

// Validator проверяет топологию. Поля экспортированы, чтобы тест мог сузить
// лимиты, не собирая документ на сотню узлов.
//
// Валидатор не меняется после создания, поэтому один экземпляр безопасно
// делят все горутины-запросы.
type Validator struct {
	// AllowedTypes — типы узлов, которые понимает движок симуляции.
	AllowedTypes map[model.NodeType]struct{}
	// MaxNodes и MaxLinks — лимиты размера; 0 и меньше — лимит по умолчанию.
	MaxNodes int
	MaxLinks int
}

// New создаёт валидатор со всеми известными типами узлов и лимитами по умолчанию.
func New() *Validator {
	allowed := map[model.NodeType]struct{}{
		model.NodeTypeClient:       {},
		model.NodeTypeServer:       {},
		model.NodeTypeRouter:       {},
		model.NodeTypeSwitch:       {},
		model.NodeTypeDNS:          {},
		model.NodeTypeFirewall:     {},
		model.NodeTypeLoadBalancer: {},
		model.NodeTypeProxy:        {},
		model.NodeTypeNATGateway:   {},
		model.NodeTypeVPNGateway:   {},
		model.NodeTypeDatabase:     {},
		model.NodeTypeInternet:     {},
	}

	return &Validator{
		AllowedTypes: allowed,
		MaxNodes:     model.MaxNodes,
		MaxLinks:     model.MaxLinks,
	}
}

// ValidateRaw проверяет топологию в виде сырого JSON и возвращает все найденные
// проблемы сразу, а не первую: пользователь исправит их за один заход.
//
// Ошибка разбора самого JSON — не error, а запись в результате: для клиента
// это такая же проблема топологии, как ссылка на несуществующий узел.
func (v *Validator) ValidateRaw(data json.RawMessage) model.ValidationResult {
	rep := newReport()

	doc, ok := decodeDocument(data, rep)
	if ok {
		v.checkLimits(doc, rep)

		nodesByID := checkNodes(doc, v.AllowedTypes, rep)
		checkLinks(doc, nodesByID, rep)
		checkNodeConfigs(doc, nodesByID, rep)
	}

	return rep.result()
}

// checkLimits сообщает о превышении лимитов размера. Проверка не прерывает
// валидацию: остальные проблемы документа тоже попадут в отчёт.
func (v *Validator) checkLimits(doc model.Document, rep *report) {
	if len(doc.Nodes) > v.limitNodes() {
		rep.addf("nodes", "too many nodes: max %d", v.limitNodes())
	}

	if len(doc.Links) > v.limitLinks() {
		rep.addf("links", "too many links: max %d", v.limitLinks())
	}
}

func (v *Validator) limitNodes() int {
	if v.MaxNodes <= 0 {
		return model.MaxNodes
	}

	return v.MaxNodes
}

func (v *Validator) limitLinks() int {
	if v.MaxLinks <= 0 {
		return model.MaxLinks
	}

	return v.MaxLinks
}
