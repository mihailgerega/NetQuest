package quests

import (
	"encoding/json"
	"time"
)

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

type AttemptStatus string

const (
	AttemptNotStarted AttemptStatus = "not_started"
	AttemptInProgress AttemptStatus = "in_progress"
	AttemptCompleted  AttemptStatus = "completed"
	AttemptFailed     AttemptStatus = "failed"
)

type Quest struct {
	ID                  string            `json:"id"`
	Slug                string            `json:"slug"`
	Title               string            `json:"title"`
	Difficulty          Difficulty        `json:"difficulty"`
	Category            string            `json:"category"`
	Description         string            `json:"description"`
	Goal                string            `json:"goal"`
	LearningObjectives  []string          `json:"learningObjectives"`
	InitialTopology     json.RawMessage   `json:"initialTopology"`
	ExpectedChecks      []CheckSpec       `json:"expectedChecks"`
	Hints               []string          `json:"hints"`
	ProgressiveHints    []ProgressiveHint `json:"progressiveHints"`
	AfterSolution       string            `json:"afterSolutionExplanation"`
	GlossaryTerms       []GlossaryTerm    `json:"glossaryTerms"`
	RealWorldImportance string            `json:"realWorldImportance,omitempty"`
	SuccessMessage      string            `json:"successMessage"`
	FailureMessage      string            `json:"failureMessage"`
	EstimatedMinutes    int               `json:"estimatedMinutes"`
	CreatedAt           time.Time         `json:"createdAt,omitempty"`
	UpdatedAt           time.Time         `json:"updatedAt,omitempty"`
	AttemptStatus       AttemptStatus     `json:"attemptStatus,omitempty"`
}

type ProgressiveHint struct {
	Title          string   `json:"title"`
	Body           string   `json:"body"`
	Level          string   `json:"level,omitempty"`
	RelatedCheckID string   `json:"relatedCheckId,omitempty"`
	Actions        []string `json:"actions,omitempty"`
}

type GlossaryTerm struct {
	Term       string `json:"term"`
	Definition string `json:"definition"`
}

type CheckType string

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

type Attempt struct {
	ID                 string          `json:"id"`
	QuestID            string          `json:"questId"`
	UserID             string          `json:"userId,omitempty"`
	ProjectID          *string         `json:"projectId,omitempty"`
	CurrentTopologyID  *string         `json:"currentTopologyId,omitempty"`
	Status             AttemptStatus   `json:"status"`
	AttemptsCount      int             `json:"attemptsCount"`
	RevealedHintsCount int             `json:"revealedHintsCount"`
	LastCheckResult    json.RawMessage `json:"lastCheckResult,omitempty"`
	CompletedAt        *time.Time      `json:"completedAt,omitempty"`
	CreatedAt          time.Time       `json:"createdAt"`
	UpdatedAt          time.Time       `json:"updatedAt"`
}

type CheckResult struct {
	ID      string         `json:"id"`
	Passed  bool           `json:"passed"`
	Message string         `json:"message"`
	Details map[string]any `json:"details,omitempty"`
}

type Result struct {
	Passed                   bool          `json:"passed"`
	Score                    int           `json:"score"`
	Checks                   []CheckResult `json:"checks"`
	Hints                    []string      `json:"hints"`
	AfterSolutionExplanation string        `json:"afterSolutionExplanation,omitempty"`
}

type ListResponse struct {
	Quests []Quest `json:"quests"`
}

type StartResponse struct {
	Quest   Quest   `json:"quest"`
	Attempt Attempt `json:"attempt"`
}

type CheckRequest struct {
	Topology json.RawMessage `json:"topology"`
	Seed     *int64          `json:"seed,omitempty"`
}

type CheckResponse struct {
	Attempt Attempt `json:"attempt"`
	Result  Result  `json:"result"`
}

type ResetResponse struct {
	Quest   Quest   `json:"quest"`
	Attempt Attempt `json:"attempt"`
}

type RevealHintRequest struct {
	RevealedHintsCount int `json:"revealedHintsCount"`
}

type RevealHintResponse struct {
	Attempt Attempt `json:"attempt"`
}
