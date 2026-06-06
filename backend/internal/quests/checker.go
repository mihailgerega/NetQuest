package quests

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/netquest/netquest/backend/internal/simulation"
	"github.com/netquest/netquest/backend/internal/topology"
	"github.com/netquest/netquest/backend/pkg/idgen"
)

type Checker struct {
	Engine    simulation.SimulationEngine
	Validator topology.Validator
}

func NewChecker(engine simulation.SimulationEngine, validator topology.Validator) Checker {
	return Checker{Engine: engine, Validator: validator}
}

func (c Checker) Check(ctx context.Context, quest Quest, data json.RawMessage, seed int64) Result {
	if seed == 0 {
		seed = 7
	}
	result := Result{Checks: make([]CheckResult, 0, len(quest.ExpectedChecks)), Hints: []string{}}
	validation := c.Validator.ValidateRaw(data)
	if !validation.Valid {
		for _, spec := range quest.ExpectedChecks {
			result.Checks = append(result.Checks, CheckResult{
				ID:      spec.ID,
				Passed:  false,
				Message: "Topology не проходит validation: " + validation.Errors[0].Message,
				Details: map[string]any{"validation": validation},
			})
			if spec.Hint != "" {
				result.Hints = appendUnique(result.Hints, spec.Hint)
			} else if hint := progressiveHintForCheck(quest, spec.ID); hint != "" {
				result.Hints = appendUnique(result.Hints, hint)
			}
		}
		result.Score = score(result.Checks)
		if len(result.Hints) == 0 {
			for _, hint := range quest.ProgressiveHints {
				if hint.Body != "" {
					result.Hints = appendUnique(result.Hints, hint.Body)
					break
				}
			}
		}
		return result
	}

	var doc topology.Document
	if err := json.Unmarshal(data, &doc); err != nil {
		return Result{Passed: false, Score: 0, Checks: []CheckResult{{ID: "decode_topology", Passed: false, Message: "Topology JSON не удалось прочитать."}}, Hints: quest.Hints}
	}

	for _, spec := range quest.ExpectedChecks {
		check := c.checkOne(ctx, doc, data, seed, spec)
		result.Checks = append(result.Checks, check)
		if !check.Passed && spec.Hint != "" {
			result.Hints = appendUnique(result.Hints, spec.Hint)
		} else if !check.Passed {
			if hint := progressiveHintForCheck(quest, spec.ID); hint != "" {
				result.Hints = appendUnique(result.Hints, hint)
			}
		}
	}
	for _, hint := range quest.Hints {
		if !allPassed(result.Checks) {
			result.Hints = appendUnique(result.Hints, hint)
		}
	}
	result.Passed = allPassed(result.Checks)
	result.Score = score(result.Checks)
	if result.Passed {
		result.AfterSolutionExplanation = quest.AfterSolution
	} else if len(result.Hints) == 0 {
		for _, hint := range quest.ProgressiveHints {
			if hint.Body != "" {
				result.Hints = appendUnique(result.Hints, hint.Body)
				break
			}
		}
	}
	return result
}

func (c Checker) checkOne(ctx context.Context, doc topology.Document, data json.RawMessage, seed int64, spec CheckSpec) CheckResult {
	switch spec.Type {
	case CheckDNS:
		return checkDNS(doc, spec)
	case CheckFirewall:
		return checkFirewall(doc, spec)
	case CheckLB:
		return checkLB(doc, spec)
	case CheckReachability:
		return c.checkScenario(ctx, data, seed, spec)
	case CheckRoute:
		return c.checkRoute(ctx, data, seed, spec)
	case CheckLatency:
		return c.checkLatency(ctx, data, seed, spec)
	case CheckFailover:
		return c.checkFailover(ctx, data, seed, spec)
	case CheckSecurity:
		return c.checkSecurity(ctx, data, seed, spec)
	case CheckAdvisor:
		return checkAdvisorLike(doc, spec)
	default:
		return CheckResult{ID: spec.ID, Passed: false, Message: "Неизвестный тип проверки: " + string(spec.Type)}
	}
}

func (c Checker) checkScenario(ctx context.Context, data json.RawMessage, seed int64, spec CheckSpec) CheckResult {
	run, err := c.run(ctx, data, seed, spec)
	if err != nil {
		return CheckResult{ID: spec.ID, Passed: false, Message: spec.Title + ": simulation не запустилась.", Details: map[string]any{"error": err.Error()}}
	}
	passed := string(run.Status) == expectedStatus(spec)
	message := spec.Title
	if passed {
		message += " выполнено."
	} else {
		message += fmt.Sprintf(" не выполнено: ожидался status %s, получен %s.", expectedStatus(spec), run.Status)
	}
	return CheckResult{ID: spec.ID, Passed: passed, Message: message, Details: scenarioDetails(run)}
}

func (c Checker) checkRoute(ctx context.Context, data json.RawMessage, seed int64, spec CheckSpec) CheckResult {
	run, err := c.run(ctx, data, seed, spec)
	if err != nil {
		return CheckResult{ID: spec.ID, Passed: false, Message: spec.Title + ": simulation не запустилась.", Details: map[string]any{"error": err.Error()}}
	}
	passed := string(run.Status) == expectedStatus(spec)
	for _, nodeID := range spec.MustIncludePath {
		if !stringInSlice(run.Summary.Path, nodeID) {
			passed = false
		}
	}
	for _, nodeID := range spec.MustExcludePath {
		if stringInSlice(run.Summary.Path, nodeID) {
			passed = false
		}
	}
	message := spec.Title
	if passed {
		message += " выполнено."
	} else {
		message += ". Проверьте выбранный route."
	}
	return CheckResult{ID: spec.ID, Passed: passed, Message: message, Details: scenarioDetails(run)}
}

func (c Checker) checkLatency(ctx context.Context, data json.RawMessage, seed int64, spec CheckSpec) CheckResult {
	run, err := c.run(ctx, data, seed, spec)
	if err != nil {
		return CheckResult{ID: spec.ID, Passed: false, Message: spec.Title + ": simulation не запустилась.", Details: map[string]any{"error": err.Error()}}
	}
	passed := string(run.Status) == expectedStatus(spec)
	if spec.MaxTotalLatencyMs > 0 && run.Summary.TotalLatencyMs >= spec.MaxTotalLatencyMs {
		passed = false
	}
	for _, nodeID := range spec.MustExcludePath {
		if stringInSlice(run.Summary.Path, nodeID) {
			passed = false
		}
	}
	message := spec.Title
	if passed {
		message += " выполнено."
	} else {
		message += fmt.Sprintf(": totalLatencyMs=%d, threshold=%d.", run.Summary.TotalLatencyMs, spec.MaxTotalLatencyMs)
	}
	return CheckResult{ID: spec.ID, Passed: passed, Message: message, Details: scenarioDetails(run)}
}

func (c Checker) checkFailover(ctx context.Context, data json.RawMessage, seed int64, spec CheckSpec) CheckResult {
	run, err := c.run(ctx, data, seed, spec)
	if err != nil {
		return CheckResult{ID: spec.ID, Passed: false, Message: spec.Title + ": simulation не запустилась.", Details: map[string]any{"error": err.Error()}}
	}
	passed := string(run.Status) == expectedStatus(spec)
	if spec.DownBackendID != "" {
		if run.Summary.SelectedBackendNodeID == spec.DownBackendID {
			passed = false
		}
		skipped := false
		for _, item := range run.Summary.SkippedBackends {
			if item.NodeID == spec.DownBackendID {
				skipped = true
				break
			}
		}
		if !skipped {
			passed = false
		}
	}
	message := spec.Title
	if passed {
		message += " выполнено."
	} else {
		message += ". Down backend всё ещё не исключается корректно."
	}
	return CheckResult{ID: spec.ID, Passed: passed, Message: message, Details: scenarioDetails(run)}
}

func (c Checker) checkSecurity(ctx context.Context, data json.RawMessage, seed int64, spec CheckSpec) CheckResult {
	run, err := c.run(ctx, data, seed, spec)
	if err != nil {
		return CheckResult{ID: spec.ID, Passed: false, Message: spec.Title + ": simulation не запустилась.", Details: map[string]any{"error": err.Error()}}
	}
	passed := string(run.Status) == expectedStatus(spec)
	if spec.ForbiddenTarget != "" && stringInSlice(run.Summary.Path, spec.ForbiddenTarget) && run.Status == simulation.StatusCompleted {
		passed = false
	}
	message := spec.Title
	if passed {
		message += " выполнено."
	} else {
		message += ". Direct access всё ещё проходит."
	}
	return CheckResult{ID: spec.ID, Passed: passed, Message: message, Details: scenarioDetails(run)}
}

func (c Checker) run(ctx context.Context, data json.RawMessage, seed int64, spec CheckSpec) (simulation.RunResult, error) {
	simulationID, err := idgen.NewUUID()
	if err != nil {
		return simulation.RunResult{}, err
	}
	return c.Engine.Run(ctx, simulation.RunRequest{
		SimulationID: simulationID,
		Topology:     data,
		Seed:         seed,
		Scenario: simulation.Scenario{
			Type:         defaultString(spec.ScenarioType, "https_request"),
			SourceNodeID: spec.SourceNodeID,
			Target:       spec.Target,
			Method:       "GET",
		},
	})
}

func checkDNS(doc topology.Document, spec CheckSpec) CheckResult {
	for _, node := range doc.Nodes {
		if node.Type != topology.NodeTypeDNS {
			continue
		}
		for _, record := range anySlice(node.Config["records"]) {
			m := anyMap(record)
			if strings.EqualFold(stringValue(m["name"]), spec.Hostname) && strings.EqualFold(defaultString(stringValue(m["type"]), "A"), "A") {
				value := stringValue(m["value"])
				passed := value == spec.ExpectedIP
				message := "DNS record найден."
				if !passed {
					message = fmt.Sprintf("DNS record указывает на %s, ожидался %s.", value, spec.ExpectedIP)
				}
				return CheckResult{ID: spec.ID, Passed: passed, Message: message, Details: map[string]any{"nodeId": node.ID, "hostname": spec.Hostname, "value": value}}
			}
		}
	}
	return CheckResult{ID: spec.ID, Passed: false, Message: "DNS record " + spec.Hostname + " отсутствует.", Details: map[string]any{"hostname": spec.Hostname, "expectedIp": spec.ExpectedIP}}
}

func checkFirewall(doc topology.Document, spec CheckSpec) CheckResult {
	for _, node := range doc.Nodes {
		if node.Type != topology.NodeTypeFirewall || (spec.NodeID != "" && node.ID != spec.NodeID) {
			continue
		}
		for _, raw := range anySlice(node.Config["rules"]) {
			rule := anyMap(raw)
			if !strings.EqualFold(defaultString(stringValue(rule["protocol"]), "tcp"), defaultString(spec.Protocol, "tcp")) {
				continue
			}
			if spec.Port > 0 && intValue(rule["port"], 0) != spec.Port {
				continue
			}
			if spec.ExpectedIP != "" && !cidrOrIPMatches(defaultString(stringValue(rule["destination"]), "0.0.0.0/0"), spec.ExpectedIP) {
				continue
			}
			action := defaultString(stringValue(rule["action"]), "deny")
			passed := strings.EqualFold(action, spec.ExpectedAction)
			return CheckResult{ID: spec.ID, Passed: passed, Message: firewallMessage(passed, action, spec.ExpectedAction), Details: map[string]any{"nodeId": node.ID, "rule": rule}}
		}
		defaultPolicy := defaultString(stringValue(node.Config["defaultPolicy"]), "deny")
		passed := strings.EqualFold(defaultPolicy, spec.ExpectedAction)
		return CheckResult{ID: spec.ID, Passed: passed, Message: firewallMessage(passed, defaultPolicy, spec.ExpectedAction), Details: map[string]any{"nodeId": node.ID, "defaultPolicy": defaultPolicy}}
	}
	return CheckResult{ID: spec.ID, Passed: false, Message: "Firewall node не найден."}
}

func checkLB(doc topology.Document, spec CheckSpec) CheckResult {
	for _, node := range doc.Nodes {
		if node.Type != topology.NodeTypeLoadBalancer || (spec.NodeID != "" && node.ID != spec.NodeID) {
			continue
		}
		backends := anySlice(node.Config["backends"])
		pool := make(map[string]bool, len(backends))
		for _, item := range backends {
			m := anyMap(item)
			if id := stringValue(m["nodeId"]); id != "" {
				pool[id] = true
			}
		}
		if len(pool) == 0 {
			return CheckResult{ID: spec.ID, Passed: false, Message: "Load Balancer backend pool пустой.", Details: map[string]any{"nodeId": node.ID}}
		}
		for _, backendID := range spec.RequiredBackends {
			if !pool[backendID] {
				return CheckResult{ID: spec.ID, Passed: false, Message: "Backend pool не содержит " + backendID + ".", Details: map[string]any{"nodeId": node.ID, "pool": pool}}
			}
		}
		if len(spec.AnyOfBackends) > 0 {
			found := ""
			for _, backendID := range spec.AnyOfBackends {
				if pool[backendID] {
					found = backendID
					break
				}
			}
			if found == "" {
				return CheckResult{ID: spec.ID, Passed: false, Message: "Backend pool не содержит ни один допустимый healthy backend.", Details: map[string]any{"nodeId": node.ID, "pool": pool, "anyOfBackends": spec.AnyOfBackends}}
			}
		}
		return CheckResult{ID: spec.ID, Passed: true, Message: "Backend pool настроен.", Details: map[string]any{"nodeId": node.ID, "pool": pool}}
	}
	return CheckResult{ID: spec.ID, Passed: false, Message: "Load Balancer не найден."}
}

func checkAdvisorLike(doc topology.Document, spec CheckSpec) CheckResult {
	if spec.ForbiddenIssueCode == "HIGH_LATENCY_LINK" {
		for _, link := range doc.Links {
			if intValue(link.Config["latencyMs"], 0) >= 500 && link.Status != "down" {
				return CheckResult{ID: spec.ID, Passed: false, Message: "В topology остался active link с высокой latency.", Details: map[string]any{"linkId": link.ID}}
			}
		}
	}
	return CheckResult{ID: spec.ID, Passed: true, Message: spec.Title + " выполнено."}
}

func scenarioDetails(run simulation.RunResult) map[string]any {
	return map[string]any{
		"status":                run.Status,
		"path":                  run.Summary.Path,
		"totalLatencyMs":        run.Summary.TotalLatencyMs,
		"selectedBackendNodeId": run.Summary.SelectedBackendNodeID,
		"skippedBackends":       run.Summary.SkippedBackends,
		"errors":                run.Summary.Errors,
	}
}

func firewallMessage(passed bool, actual, expected string) string {
	if passed {
		return "Firewall rule настроен: " + expected + "."
	}
	return "Firewall action " + actual + ", ожидался " + expected + "."
}

func expectedStatus(spec CheckSpec) string {
	if spec.ExpectedStatus != "" {
		return spec.ExpectedStatus
	}
	return "completed"
}

func allPassed(checks []CheckResult) bool {
	if len(checks) == 0 {
		return false
	}
	for _, check := range checks {
		if !check.Passed {
			return false
		}
	}
	return true
}

func score(checks []CheckResult) int {
	if len(checks) == 0 {
		return 0
	}
	passed := 0
	for _, check := range checks {
		if check.Passed {
			passed++
		}
	}
	return int(float64(passed) / float64(len(checks)) * 100)
}

func appendUnique(items []string, value string) []string {
	if strings.TrimSpace(value) == "" {
		return items
	}
	for _, item := range items {
		if item == value {
			return items
		}
	}
	return append(items, value)
}

func progressiveHintForCheck(quest Quest, checkID string) string {
	for _, hint := range quest.ProgressiveHints {
		if hint.RelatedCheckID == checkID && hint.Body != "" {
			return hint.Body
		}
	}
	if len(quest.ProgressiveHints) > 0 {
		return quest.ProgressiveHints[0].Body
	}
	return ""
}

func stringInSlice(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}
	return false
}

func anyMap(value any) map[string]any {
	if m, ok := value.(map[string]any); ok {
		return m
	}
	return map[string]any{}
}

func anySlice(value any) []any {
	if items, ok := value.([]any); ok {
		return items
	}
	return nil
}

func stringValue(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s)
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func defaultString(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return strings.TrimSpace(value)
}

func intValue(value any, fallback int) int {
	switch v := value.(type) {
	case int:
		return v
	case int64:
		return int(v)
	case float64:
		return int(v)
	case json.Number:
		i, err := v.Int64()
		if err == nil {
			return int(i)
		}
	case string:
		var i int
		if _, err := fmt.Sscanf(v, "%d", &i); err == nil {
			return i
		}
	}
	return fallback
}

func cidrOrIPMatches(cidr, ipValue string) bool {
	if cidr == "" || cidr == "any" || cidr == "0.0.0.0/0" {
		return true
	}
	if cidr == ipValue {
		return true
	}
	ip := net.ParseIP(ipValue)
	if ip == nil {
		return false
	}
	if !strings.Contains(cidr, "/") {
		return false
	}
	_, network, err := net.ParseCIDR(cidr)
	return err == nil && network.Contains(ip)
}
