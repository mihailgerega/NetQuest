package record

import "time"

// Attempt — строка таблицы quest_attempts. last_check_result (jsonb) читается
// текстом; NULL — проверок ещё не было.
type Attempt struct {
	ID                 string     `db:"id"`
	QuestID            string     `db:"quest_id"`
	UserID             string     `db:"user_id"`
	ProjectID          *string    `db:"project_id"`          // NULL → nil
	CurrentTopologyID  *string    `db:"current_topology_id"` // NULL → nil
	Status             string     `db:"status"`
	AttemptsCount      int        `db:"attempts_count"`
	RevealedHintsCount int        `db:"revealed_hints_count"`
	LastCheckResult    *string    `db:"last_check_result"` // NULL → nil
	CompletedAt        *time.Time `db:"completed_at"`      // NULL → nil
	CreatedAt          time.Time  `db:"created_at"`
	UpdatedAt          time.Time  `db:"updated_at"`
}

// AttemptStatus — статус последней попытки пользователя по квесту.
type AttemptStatus struct {
	QuestID string `db:"quest_id"`
	Status  string `db:"status"`
}
