package checker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"

	"github.com/netquest/netquest/backend/internal/model"
)

// Проверки, которые запускают симуляцию по сценарию из CheckSpec
// и сравнивают её результат с ожиданием.

// Хвосты сообщений проверок: «<Title> выполнено.» или «<Title><провал>».
const (
	messageDoneSuffix = " выполнено."
)

// checkScenario — проверка достижимости: симуляция завершилась ожидаемым
// статусом (по умолчанию completed).
func (c *Checker) checkScenario(ctx context.Context, data json.RawMessage, seed int64, spec model.CheckSpec) model.CheckResult {
	run, err := c.run(ctx, data, seed, spec)
	if err != nil {
		return runFailed(spec, err)
	}

	passed := string(run.Status) == expectedStatus(spec)

	message := spec.Title + messageDoneSuffix
	if !passed {
		message = spec.Title + fmt.Sprintf(" не выполнено: ожидался status %s, получен %s.", expectedStatus(spec), run.Status)
	}

	return model.CheckResult{ID: spec.ID, Passed: passed, Message: message, Details: scenarioDetails(run)}
}

// checkRoute — статус плюс состав пути: все MustIncludePath в пути,
// ни одного MustExcludePath.
func (c *Checker) checkRoute(ctx context.Context, data json.RawMessage, seed int64, spec model.CheckSpec) model.CheckResult {
	run, err := c.run(ctx, data, seed, spec)
	if err != nil {
		return runFailed(spec, err)
	}

	passed := string(run.Status) == expectedStatus(spec) &&
		containsAll(run.Summary.Path, spec.MustIncludePath) &&
		containsNone(run.Summary.Path, spec.MustExcludePath)

	return scenarioResult(spec, run, passed, ". Проверьте выбранный route.")
}

// checkLatency — статус плюс задержка строго меньше порога и путь
// без запрещённых узлов (например, без медленного роутера).
func (c *Checker) checkLatency(ctx context.Context, data json.RawMessage, seed int64, spec model.CheckSpec) model.CheckResult {
	run, err := c.run(ctx, data, seed, spec)
	if err != nil {
		return runFailed(spec, err)
	}

	passed := string(run.Status) == expectedStatus(spec)

	if spec.MaxTotalLatencyMs > 0 && run.Summary.TotalLatencyMs >= spec.MaxTotalLatencyMs {
		passed = false
	}

	if !containsNone(run.Summary.Path, spec.MustExcludePath) {
		passed = false
	}

	failure := fmt.Sprintf(": totalLatencyMs=%d, threshold=%d.", run.Summary.TotalLatencyMs, spec.MaxTotalLatencyMs)

	return scenarioResult(spec, run, passed, failure)
}

// checkFailover — статус плюс выключенный сервер (DownBackendID) не выбран
// и явно попал в список пропущенных. Просто «не выбран» мало: Load Balancer
// должен заметить отказ, а не выбрать другой сервер случайно.
func (c *Checker) checkFailover(ctx context.Context, data json.RawMessage, seed int64, spec model.CheckSpec) model.CheckResult {
	run, err := c.run(ctx, data, seed, spec)
	if err != nil {
		return runFailed(spec, err)
	}

	passed := string(run.Status) == expectedStatus(spec)

	if spec.DownBackendID != "" {
		if run.Summary.SelectedBackendNodeID == spec.DownBackendID || !backendSkipped(run.Summary.SkippedBackends, spec.DownBackendID) {
			passed = false
		}
	}

	return scenarioResult(spec, run, passed, ". Down backend всё ещё не исключается корректно.")
}

// checkSecurity — статус плюс запрещённая цель (ForbiddenTarget) недостижима:
// успешная симуляция, путь которой проходит через неё, — провал.
func (c *Checker) checkSecurity(ctx context.Context, data json.RawMessage, seed int64, spec model.CheckSpec) model.CheckResult {
	run, err := c.run(ctx, data, seed, spec)
	if err != nil {
		return runFailed(spec, err)
	}

	passed := string(run.Status) == expectedStatus(spec)

	if spec.ForbiddenTarget != "" && stringInSlice(run.Summary.Path, spec.ForbiddenTarget) && run.Status == model.SimulationStatusCompleted {
		passed = false
	}

	return scenarioResult(spec, run, passed, ". Direct access всё ещё проходит.")
}

// run запускает симуляцию по сценарию проверки. У каждого запуска свой
// случайный ID: результаты проверок не сохраняются как симуляции,
// ID нужен движку только для ID событий.
func (c *Checker) run(ctx context.Context, data json.RawMessage, seed int64, spec model.CheckSpec) (model.RunResult, error) {
	return c.engine.Run(ctx, model.RunRequest{
		SimulationID: uuid.NewString(),
		Topology:     data,
		Seed:         seed,
		Scenario: model.Scenario{
			Type:         defaultString(spec.ScenarioType, model.ScenarioHTTPSRequest),
			SourceNodeID: spec.SourceNodeID,
			Target:       spec.Target,
			Method:       "GET",
		},
	})
}

// runFailed — проверка провалена, потому что симуляцию не удалось даже запустить.
func runFailed(spec model.CheckSpec, err error) model.CheckResult {
	return model.CheckResult{
		ID:      spec.ID,
		Passed:  false,
		Message: spec.Title + ": simulation не запустилась.",
		Details: map[string]any{"error": err.Error()},
	}
}

// scenarioResult собирает результат сценарной проверки с сообщением
// «<Title> выполнено.» или «<Title><failure>».
func scenarioResult(spec model.CheckSpec, run model.RunResult, passed bool, failure string) model.CheckResult {
	message := spec.Title + messageDoneSuffix
	if !passed {
		message = spec.Title + failure
	}

	return model.CheckResult{ID: spec.ID, Passed: passed, Message: message, Details: scenarioDetails(run)}
}

// scenarioDetails — то из результата симуляции, что поможет пользователю
// понять провал проверки.
func scenarioDetails(run model.RunResult) map[string]any {
	return map[string]any{
		"status":                run.Status,
		"path":                  run.Summary.Path,
		"totalLatencyMs":        run.Summary.TotalLatencyMs,
		"selectedBackendNodeId": run.Summary.SelectedBackendNodeID,
		"skippedBackends":       run.Summary.SkippedBackends,
		"errors":                run.Summary.Errors,
	}
}

// expectedStatus — ожидаемый статус симуляции; по умолчанию completed.
func expectedStatus(spec model.CheckSpec) string {
	if spec.ExpectedStatus != "" {
		return spec.ExpectedStatus
	}

	return string(model.SimulationStatusCompleted)
}

// backendSkipped сообщает, есть ли сервер в списке пропущенных Load Balancer'ом.
func backendSkipped(skipped []model.BackendSkip, nodeID string) bool {
	for _, item := range skipped {
		if item.NodeID == nodeID {
			return true
		}
	}

	return false
}
