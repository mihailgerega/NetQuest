package model

import (
	"encoding/json"
	"time"
)

// Difficulty — сложность квеста.
type Difficulty string

// Уровни сложности. У hard-квестов подсказок больше (минимум 5 вместо 4).
const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

// Quest — упражнение: сломанная топология, цель и проверки решения.
//
// Каталог квестов живёт в коде (service/domain/catalog) и при обращении
// синхронизируется в таблицу quests. Клиенту квест уходит этой же структурой,
// поэтому json-теги — контракт. AttemptStatus — не часть каталога: сервис
// заполняет его для конкретного пользователя перед ответом.
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

// ProgressiveHint — подсказка, которая открывается по одной: от общей идеи
// (Level "concept") к конкретному действию (Level "action").
// RelatedCheckID связывает подсказку с проверкой, которую она помогает пройти.
type ProgressiveHint struct {
	Title          string   `json:"title"`
	Body           string   `json:"body"`
	Level          string   `json:"level,omitempty"`
	RelatedCheckID string   `json:"relatedCheckId,omitempty"`
	Actions        []string `json:"actions,omitempty"`
}

// GlossaryTerm — термин из глоссария квеста.
type GlossaryTerm struct {
	Term       string `json:"term"`
	Definition string `json:"definition"`
}
