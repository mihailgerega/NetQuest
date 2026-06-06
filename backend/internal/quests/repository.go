package quests

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/netquest/netquest/backend/pkg/apperrors"
)

type Repository struct {
	db *pgxpool.Pool
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db}
}

func (r *Repository) UpsertQuests(ctx context.Context, quests []Quest) error {
	if r.db == nil {
		return apperrors.Internal("postgres is not configured")
	}
	for _, quest := range quests {
		objectives, err := json.Marshal(quest.LearningObjectives)
		if err != nil {
			return fmt.Errorf("marshal quest objectives: %w", err)
		}
		checks, err := json.Marshal(quest.ExpectedChecks)
		if err != nil {
			return fmt.Errorf("marshal quest checks: %w", err)
		}
		hints, err := json.Marshal(quest.Hints)
		if err != nil {
			return fmt.Errorf("marshal quest hints: %w", err)
		}
		progressiveHints, err := json.Marshal(quest.ProgressiveHints)
		if err != nil {
			return fmt.Errorf("marshal quest progressive hints: %w", err)
		}
		glossaryTerms, err := json.Marshal(quest.GlossaryTerms)
		if err != nil {
			return fmt.Errorf("marshal quest glossary terms: %w", err)
		}
		_, err = r.db.Exec(ctx, `
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
				estimated_minutes = EXCLUDED.estimated_minutes
		`, quest.ID, quest.Slug, quest.Title, quest.Difficulty, quest.Category, quest.Description, quest.Goal,
			string(objectives), string(quest.InitialTopology), string(checks), string(hints),
			string(progressiveHints), quest.AfterSolution, string(glossaryTerms), quest.RealWorldImportance,
			quest.SuccessMessage, quest.FailureMessage, quest.EstimatedMinutes)
		if err != nil {
			return fmt.Errorf("upsert quest %s: %w", quest.ID, err)
		}
	}
	return nil
}

func (r *Repository) LatestAttemptStatuses(ctx context.Context, userID string) (map[string]AttemptStatus, error) {
	if r.db == nil {
		return nil, apperrors.Internal("postgres is not configured")
	}
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT ON (quest_id) quest_id, status
		FROM quest_attempts
		WHERE user_id = $1
		ORDER BY quest_id, updated_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("list quest attempt statuses: %w", err)
	}
	defer rows.Close()
	result := map[string]AttemptStatus{}
	for rows.Next() {
		var questID string
		var status AttemptStatus
		if err := rows.Scan(&questID, &status); err != nil {
			return nil, fmt.Errorf("scan quest attempt status: %w", err)
		}
		result[questID] = status
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate quest attempt statuses: %w", err)
	}
	return result, nil
}

func (r *Repository) CreateAttempt(ctx context.Context, attempt Attempt) (Attempt, error) {
	if r.db == nil {
		return Attempt{}, apperrors.Internal("postgres is not configured")
	}
	err := scanAttempt(r.db.QueryRow(ctx, `
		INSERT INTO quest_attempts (id, quest_id, user_id, project_id, current_topology_id, status, attempts_count, revealed_hints_count)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id::text, quest_id, user_id::text, project_id::text, current_topology_id::text,
		          status, attempts_count, revealed_hints_count, last_check_result::text, completed_at, created_at, updated_at
	`, attempt.ID, attempt.QuestID, attempt.UserID, attempt.ProjectID, attempt.CurrentTopologyID, attempt.Status, attempt.AttemptsCount, attempt.RevealedHintsCount), &attempt)
	if err != nil {
		return Attempt{}, fmt.Errorf("create quest attempt: %w", err)
	}
	return attempt, nil
}

func (r *Repository) GetAttemptForOwner(ctx context.Context, attemptID, userID string) (Attempt, error) {
	if r.db == nil {
		return Attempt{}, apperrors.Internal("postgres is not configured")
	}
	var attempt Attempt
	err := scanAttempt(r.db.QueryRow(ctx, `
		SELECT id::text, quest_id, user_id::text, project_id::text, current_topology_id::text,
		       status, attempts_count, revealed_hints_count, last_check_result::text, completed_at, created_at, updated_at
		FROM quest_attempts
		WHERE id = $1 AND user_id = $2
	`, attemptID, userID), &attempt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Attempt{}, apperrors.NotFound("quest attempt not found")
	}
	if err != nil {
		return Attempt{}, fmt.Errorf("get quest attempt: %w", err)
	}
	return attempt, nil
}

func (r *Repository) UpdateAttemptAfterCheck(ctx context.Context, attemptID, userID string, status AttemptStatus, result Result) (Attempt, error) {
	if r.db == nil {
		return Attempt{}, apperrors.Internal("postgres is not configured")
	}
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return Attempt{}, fmt.Errorf("marshal check result: %w", err)
	}
	var completedAt any
	if status == AttemptCompleted {
		completedAt = time.Now().UTC()
	}
	var attempt Attempt
	err = scanAttempt(r.db.QueryRow(ctx, `
		UPDATE quest_attempts
		SET status = $3,
		    attempts_count = attempts_count + 1,
		    last_check_result = $4::jsonb,
		    completed_at = COALESCE($5::timestamptz, completed_at)
		WHERE id = $1 AND user_id = $2
		RETURNING id::text, quest_id, user_id::text, project_id::text, current_topology_id::text,
		          status, attempts_count, revealed_hints_count, last_check_result::text, completed_at, created_at, updated_at
	`, attemptID, userID, status, string(resultBytes), completedAt), &attempt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Attempt{}, apperrors.NotFound("quest attempt not found")
	}
	if err != nil {
		return Attempt{}, fmt.Errorf("update quest attempt: %w", err)
	}
	return attempt, nil
}

func (r *Repository) InsertCheckResult(ctx context.Context, id, attemptID, userID string, result Result) error {
	if r.db == nil {
		return apperrors.Internal("postgres is not configured")
	}
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal check history result: %w", err)
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO quest_check_results (id, attempt_id, user_id, passed, score, result)
		VALUES ($1, $2, $3, $4, $5, $6::jsonb)
	`, id, attemptID, userID, result.Passed, result.Score, string(resultBytes))
	if err != nil {
		return fmt.Errorf("insert quest check result: %w", err)
	}
	return nil
}

func (r *Repository) ResetAttempt(ctx context.Context, attemptID, userID string) (Attempt, error) {
	if r.db == nil {
		return Attempt{}, apperrors.Internal("postgres is not configured")
	}
	var attempt Attempt
	err := scanAttempt(r.db.QueryRow(ctx, `
		UPDATE quest_attempts
		SET status = 'in_progress',
		    attempts_count = 0,
		    revealed_hints_count = 0,
		    last_check_result = NULL,
		    completed_at = NULL
		WHERE id = $1 AND user_id = $2
		RETURNING id::text, quest_id, user_id::text, project_id::text, current_topology_id::text,
		          status, attempts_count, revealed_hints_count, last_check_result::text, completed_at, created_at, updated_at
	`, attemptID, userID), &attempt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Attempt{}, apperrors.NotFound("quest attempt not found")
	}
	if err != nil {
		return Attempt{}, fmt.Errorf("reset quest attempt: %w", err)
	}
	return attempt, nil
}

func (r *Repository) UpdateRevealedHintsCount(ctx context.Context, attemptID, userID string, count int) (Attempt, error) {
	if r.db == nil {
		return Attempt{}, apperrors.Internal("postgres is not configured")
	}
	var attempt Attempt
	err := scanAttempt(r.db.QueryRow(ctx, `
		UPDATE quest_attempts
		SET revealed_hints_count = GREATEST(revealed_hints_count, $3)
		WHERE id = $1 AND user_id = $2
		RETURNING id::text, quest_id, user_id::text, project_id::text, current_topology_id::text,
		          status, attempts_count, revealed_hints_count, last_check_result::text, completed_at, created_at, updated_at
	`, attemptID, userID, count), &attempt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Attempt{}, apperrors.NotFound("quest attempt not found")
	}
	if err != nil {
		return Attempt{}, fmt.Errorf("update revealed hints count: %w", err)
	}
	return attempt, nil
}

type attemptScanner interface {
	Scan(dest ...any) error
}

func scanAttempt(row attemptScanner, attempt *Attempt) error {
	var projectID *string
	var topologyID *string
	var lastResult *string
	if err := row.Scan(
		&attempt.ID,
		&attempt.QuestID,
		&attempt.UserID,
		&projectID,
		&topologyID,
		&attempt.Status,
		&attempt.AttemptsCount,
		&attempt.RevealedHintsCount,
		&lastResult,
		&attempt.CompletedAt,
		&attempt.CreatedAt,
		&attempt.UpdatedAt,
	); err != nil {
		return err
	}
	attempt.ProjectID = projectID
	attempt.CurrentTopologyID = topologyID
	if lastResult != nil {
		attempt.LastCheckResult = json.RawMessage(*lastResult)
	}
	return nil
}
