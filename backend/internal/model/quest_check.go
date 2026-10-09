package model

// CheckType — вид проверки решения квеста.
type CheckType string

// Статические проверки (dns, firewall, lb, advisor) смотрят на настройки узлов
// в документе; остальные запускают симуляцию и проверяют её результат.
const (
	CheckReachability CheckType = "reachability_check"
	CheckDNS          CheckType = "dns_check"
	CheckFirewall     CheckType = "firewall_check"
	CheckLB           CheckType = "lb_check"
	CheckRoute        CheckType = "route_check"
	CheckSecurity     CheckType = "security_check"
	CheckLatency      CheckType = "latency_check"
	CheckFailover     CheckType = "failover_check"
	CheckAdvisor      CheckType = "advisor_check"
)

// CheckSpec — описание одной проверки квеста. Какие поля важны, зависит от Type:
// DNS-проверке нужны Hostname и ExpectedIP, проверке маршрута — MustIncludePath
// и MustExcludePath, проверке failover — DownBackendID и т.д.
//
// Хранится в quests.expected_checks (jsonb) и отдаётся клиенту, поэтому json-теги — контракт.
type CheckSpec struct {
	ID                 string    `json:"id"`
	Type               CheckType `json:"type"`
	Title              string    `json:"title"`
	Message            string    `json:"message,omitempty"`
	Hint               string    `json:"hint,omitempty"`
	SourceNodeID       string    `json:"sourceNodeId,omitempty"`
	Target             string    `json:"target,omitempty"`
	ScenarioType       string    `json:"scenarioType,omitempty"`
	ExpectedStatus     string    `json:"expectedStatus,omitempty"`
	Hostname           string    `json:"hostname,omitempty"`
	ExpectedIP         string    `json:"expectedIp,omitempty"`
	ResolverNodeID     string    `json:"resolverNodeId,omitempty"`
	NodeID             string    `json:"nodeId,omitempty"`
	ExpectedAction     string    `json:"expectedAction,omitempty"`
	Protocol           string    `json:"protocol,omitempty"`
	Port               int       `json:"port,omitempty"`
	RequiredBackends   []string  `json:"requiredBackends,omitempty"`
	AnyOfBackends      []string  `json:"anyOfBackends,omitempty"`
	DownBackendID      string    `json:"downBackendId,omitempty"`
	MustIncludePath    []string  `json:"mustIncludePath,omitempty"`
	MustExcludePath    []string  `json:"mustExcludePath,omitempty"`
	ForbiddenTarget    string    `json:"forbiddenTarget,omitempty"`
	MaxTotalLatencyMs  int64     `json:"maxTotalLatencyMs,omitempty"`
	MaxLinkLatencyMs   int64     `json:"maxLinkLatencyMs,omitempty"`
	ForbiddenIssueCode string    `json:"forbiddenIssueCode,omitempty"`
	DependsOn          []string  `json:"dependsOn,omitempty"`
}

// CheckResult — итог одной проверки.
type CheckResult struct {
	ID      string         `json:"id"`
	Passed  bool           `json:"passed"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

// QuestResult — итог проверки решения целиком: пройдены ли все проверки,
// процент пройденных (Score) и подсказки к проваленным.
//
// Сохраняется в quest_attempts.last_check_result и quest_check_results.result (jsonb).
type QuestResult struct {
	Passed                   bool          `json:"passed"`
	Score                    int           `json:"score"`
	Checks                   []CheckResult `json:"checks"`
	Hints                    []string      `json:"hints"`
	AfterSolutionExplanation string        `json:"afterSolutionExplanation,omitempty"`
}
