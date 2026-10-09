// Package checker — доменный сервис проверки решения квеста.
//
// Решение проверяется не по картинке на рабочем поле, а по поведению сети:
// для каждой проверки квеста (model.CheckSpec) чекер либо смотрит настройки
// узлов в документе (DNS-запись, правило firewall, пул Load Balancer'а), либо
// запускает симуляцию и проверяет её результат (статус, путь, задержку,
// выбранный сервер).
//
// Место в цепочке: service/application/quest → Checker.Check → движок симуляции.
// Чекер ничего не сохраняет: только считает model.QuestResult.
package checker

import (
	"context"
	"encoding/json"

	"github.com/netquest/netquest/backend/internal/model"
)

// defaultSeed — seed симуляций проверки, если клиент его не прислал.
// Фиксированный: одно и то же решение должно проверяться одинаково.
const defaultSeed = 7

// Checker проверяет решения квестов. Состояния не хранит — безопасен
// для параллельных запросов.
type Checker struct {
	engine    SimulationEngine
	validator TopologyValidator
}

// New создаёт чекер поверх движка симуляции и валидатора топологии.
func New(engine SimulationEngine, validator TopologyValidator) *Checker {
	return &Checker{
		engine:    engine,
		validator: validator,
	}
}

// Check проверяет топологию-решение по всем проверкам квеста.
//
// Невалидная топология проваливает все проверки сразу — с первой ошибкой
// валидации в тексте. К каждой проваленной проверке подбирается подсказка:
// её собственная (CheckSpec.Hint), иначе связанная прогрессивная подсказка.
// Решение засчитано, только если пройдены все проверки.
func (c *Checker) Check(ctx context.Context, quest model.Quest, data json.RawMessage, seed int64) model.QuestResult {
	if seed == 0 {
		seed = defaultSeed
	}

	validation := c.validator.ValidateRaw(data)
	if !validation.Valid {
		return invalidTopologyResult(quest, validation)
	}

	var doc model.Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return model.QuestResult{
			Passed: false,
			Score:  0,
			Checks: []model.CheckResult{{ID: "decode_topology", Passed: false, Message: "Topology JSON не удалось прочитать."}},
			Hints:  quest.Hints,
		}
	}

	result := model.QuestResult{
		Checks: make([]model.CheckResult, 0, len(quest.ExpectedChecks)),
		Hints:  []string{},
	}

	for _, spec := range quest.ExpectedChecks {
		check := c.checkOne(ctx, doc, data, seed, spec)
		result.Checks = append(result.Checks, check)

		if !check.Passed {
			result.Hints = appendUnique(result.Hints, hintForCheck(quest, spec))
		}
	}

	// Общие подсказки квеста добавляются только к непройденному решению.
	if !allPassed(result.Checks) {
		for _, hint := range quest.Hints {
			result.Hints = appendUnique(result.Hints, hint)
		}
	}

	result.Passed = allPassed(result.Checks)
	result.Score = score(result.Checks)

	switch {
	case result.Passed:
		result.AfterSolutionExplanation = quest.AfterSolution
	case len(result.Hints) == 0:
		result.Hints = appendFirstProgressiveHint(result.Hints, quest)
	}

	return result
}

// invalidTopologyResult — все проверки провалены одной причиной:
// топология не прошла валидацию.
func invalidTopologyResult(quest model.Quest, validation model.ValidationResult) model.QuestResult {
	result := model.QuestResult{
		Checks: make([]model.CheckResult, 0, len(quest.ExpectedChecks)),
		Hints:  []string{},
	}

	for _, spec := range quest.ExpectedChecks {
		result.Checks = append(result.Checks, model.CheckResult{
			ID:      spec.ID,
			Passed:  false,
			Message: "Topology не проходит validation: " + validation.Errors[0].Message,
			Details: map[string]any{"validation": validation},
		})
		result.Hints = appendUnique(result.Hints, hintForCheck(quest, spec))
	}

	result.Score = score(result.Checks)

	if len(result.Hints) == 0 {
		result.Hints = appendFirstProgressiveHint(result.Hints, quest)
	}

	return result
}

// checkOne выбирает способ проверки по типу: статические проверки читают
// документ, сценарные — запускают симуляцию.
func (c *Checker) checkOne(ctx context.Context, doc model.Document, data json.RawMessage, seed int64, spec model.CheckSpec) model.CheckResult {
	switch spec.Type {
	case model.CheckDNS:
		return checkDNS(doc, spec)
	case model.CheckFirewall:
		return checkFirewall(doc, spec)
	case model.CheckLB:
		return checkLB(doc, spec)
	case model.CheckAdvisor:
		return checkAdvisorLike(doc, spec)
	case model.CheckReachability:
		return c.checkScenario(ctx, data, seed, spec)
	case model.CheckRoute:
		return c.checkRoute(ctx, data, seed, spec)
	case model.CheckLatency:
		return c.checkLatency(ctx, data, seed, spec)
	case model.CheckFailover:
		return c.checkFailover(ctx, data, seed, spec)
	case model.CheckSecurity:
		return c.checkSecurity(ctx, data, seed, spec)
	default:
		return model.CheckResult{ID: spec.ID, Passed: false, Message: "Неизвестный тип проверки: " + string(spec.Type)}
	}
}
