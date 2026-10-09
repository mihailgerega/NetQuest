package quest

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/netquest/netquest/backend/internal/model"
)

// UpsertQuests синхронизирует каталог из кода в таблицу quests: новые квесты
// вставляются, существующие (по id) — перезаписываются.
//
// Списки и вложенные структуры квеста лежат в jsonb-колонках; сериализует их
// encoding/json по тегам модели. Квесты пишутся по одному без транзакции:
// UPSERT идемпотентен, и оборванная синхронизация доделается следующим вызовом.
func (r *repository) UpsertQuests(ctx context.Context, quests []model.Quest) error {
	const query = `
		INSERT INTO quests (
			id, slug, title, difficulty, category, description, goal,
			learning_objectives, initial_topology, expected_checks, hints,
			progressive_hints, after_solution_explanation, glossary_terms, real_world_importance,
			success_message, failure_message, estimated_minutes
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8::jsonb, $9::jsonb, $10::jsonb, $11::jsonb, $12::jsonb, $13, $14::jsonb, $15, $16, $17, $18)
		ON CONFLICT (id) DO UPDATE SET
			slug = EXCLUDED.slug,
			title = EXCLUDED.title,
			difficulty = EXCLUDED.difficulty,
			category = EXCLUDED.category,
			description = EXCLUDED.description,
			goal = EXCLUDED.goal,
			learning_objectives = EXCLUDED.learning_objectives,
			initial_topology = EXCLUDED.initial_topology,
			expected_checks = EXCLUDED.expected_checks,
			hints = EXCLUDED.hints,
			progressive_hints = EXCLUDED.progressive_hints,
			after_solution_explanation = EXCLUDED.after_solution_explanation,
			glossary_terms = EXCLUDED.glossary_terms,
			real_world_importance = EXCLUDED.real_world_importance,
			success_message = EXCLUDED.success_message,
			failure_message = EXCLUDED.failure_message,
			estimated_minutes = EXCLUDED.estimated_minutes`

	for _, quest := range quests {
		columns, err := marshalQuestColumns(quest)
		if err != nil {
			return fmt.Errorf("сериализовать квест %s: %w", quest.ID, err)
		}

		if _, err := r.pool.Exec(ctx, query,
			quest.ID, quest.Slug, quest.Title, quest.Difficulty, quest.Category, quest.Description, quest.Goal,
			columns.learningObjectives, string(quest.InitialTopology), columns.expectedChecks, columns.hints,
			columns.progressiveHints, quest.AfterSolution, columns.glossaryTerms, quest.RealWorldImportance,
			quest.SuccessMessage, quest.FailureMessage, quest.EstimatedMinutes); err != nil {
			return fmt.Errorf("сохранить квест %s: %w", quest.ID, err)
		}
	}

	return nil
}

// questJSONColumns — jsonb-колонки квеста в виде текста для параметров запроса.
type questJSONColumns struct {
	learningObjectives string
	expectedChecks     string
	hints              string
	progressiveHints   string
	glossaryTerms      string
}

// marshalQuestColumns сериализует списки квеста для jsonb-колонок.
func marshalQuestColumns(quest model.Quest) (questJSONColumns, error) {
	var (
		columns questJSONColumns
		err     error
	)

	if columns.learningObjectives, err = marshalJSON(quest.LearningObjectives); err != nil {
		return questJSONColumns{}, fmt.Errorf("learning_objectives: %w", err)
	}

	if columns.expectedChecks, err = marshalJSON(quest.ExpectedChecks); err != nil {
		return questJSONColumns{}, fmt.Errorf("expected_checks: %w", err)
	}

	if columns.hints, err = marshalJSON(quest.Hints); err != nil {
		return questJSONColumns{}, fmt.Errorf("hints: %w", err)
	}

	if columns.progressiveHints, err = marshalJSON(quest.ProgressiveHints); err != nil {
		return questJSONColumns{}, fmt.Errorf("progressive_hints: %w", err)
	}

	if columns.glossaryTerms, err = marshalJSON(quest.GlossaryTerms); err != nil {
		return questJSONColumns{}, fmt.Errorf("glossary_terms: %w", err)
	}

	return columns, nil
}

// marshalJSON сериализует значение в JSON-текст для параметра ::jsonb.
func marshalJSON(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return string(data), nil
}
