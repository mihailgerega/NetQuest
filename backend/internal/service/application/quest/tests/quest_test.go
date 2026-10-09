package tests

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	errs "github.com/netquest/netquest/backend/internal/errors"
	"github.com/netquest/netquest/backend/internal/model"
	questService "github.com/netquest/netquest/backend/internal/service/application/quest"
	"github.com/netquest/netquest/backend/internal/service/application/quest/mocks"
	"github.com/netquest/netquest/backend/internal/service/domain/catalog"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// Unit-тесты сервиса квестов. Хранилище и чекер — моки: проверяются правила
// самого сервиса — синхронизация каталога, статусы, подсчёт подсказок, сохранение
// результата проверки. Каталог настоящий.

const (
	userID    = "user-1"
	attemptID = "attempt-1"
	questID   = "quest-dns-lookup"
)

// questServiceAPI — методы сервиса глазами теста (тип сервиса неэкспортируемый).
type questServiceAPI interface {
	List(ctx context.Context, userID string) ([]model.Quest, error)
	Get(ctx context.Context, userID, questID string) (model.Quest, error)
	Check(ctx context.Context, userID, attemptID string, in input.CheckQuestInput) (model.Attempt, model.QuestResult, error)
	Reset(ctx context.Context, userID, attemptID string) (model.Quest, model.Attempt, error)
	RevealHint(ctx context.Context, userID, attemptID string, revealedHintsCount int) (model.Attempt, error)
}

// newService собирает сервис с новыми моками; каталог синхронизируется перед
// каждой операцией, поэтому UpsertQuests разрешён всегда (Maybe).
func newService(t *testing.T) (questServiceAPI, *mocks.QuestRepository, *mocks.QuestChecker) {
	t.Helper()

	repo := mocks.NewQuestRepository(t)
	checker := mocks.NewQuestChecker(t)
	repo.EXPECT().UpsertQuests(mock.Anything, mock.Anything).Return(nil).Maybe()

	return questService.New(repo, checker, catalog.Catalog()), repo, checker
}

func progressiveHintsCount(t *testing.T) int {
	t.Helper()

	for _, quest := range catalog.Catalog() {
		if quest.ID == questID {
			return len(quest.ProgressiveHints)
		}
	}

	t.Fatalf("квест %s не найден", questID)

	return 0
}

// TestRevealHint: число открытых подсказок не уменьшается и не превышает
// число подсказок квеста; отрицательное значение считается нулём.
func TestRevealHint(t *testing.T) {
	t.Parallel()

	hints := progressiveHintsCount(t)

	tests := []struct {
		name      string
		revealed  int // уже открыто в попытке
		requested int // прислал клиент
		want      int // уйдёт в хранилище
	}{
		{name: "открыть две", revealed: 0, requested: 2, want: 2},
		{name: "меньше уже открытых — счётчик не уменьшается", revealed: 2, requested: 1, want: 2},
		{name: "больше, чем подсказок в квесте", revealed: 0, requested: 999, want: hints},
		{name: "отрицательное значение", revealed: 0, requested: -5, want: 0},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			svc, repo, _ := newService(t)

			repo.EXPECT().GetAttemptForOwner(ctx, attemptID, userID).
				Return(model.Attempt{ID: attemptID, QuestID: questID, UserID: userID, RevealedHintsCount: tc.revealed}, nil).Once()
			repo.EXPECT().UpdateRevealedHintsCount(ctx, attemptID, userID, tc.want).
				Return(model.Attempt{ID: attemptID, RevealedHintsCount: tc.want}, nil).Once()

			attempt, err := svc.RevealHint(ctx, userID, attemptID, tc.requested)

			require.NoError(t, err)
			assert.Equal(t, tc.want, attempt.RevealedHintsCount)
		})
	}
}

// TestReset: сброшенная попытка возвращается вместе с квестом в её статусе.
func TestReset(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, repo, _ := newService(t)

	repo.EXPECT().ResetAttempt(ctx, attemptID, userID).
		Return(model.Attempt{ID: attemptID, QuestID: questID, Status: model.AttemptInProgress}, nil).Once()

	quest, attempt, err := svc.Reset(ctx, userID, attemptID)

	require.NoError(t, err)
	assert.Equal(t, questID, quest.ID)
	assert.Equal(t, model.AttemptInProgress, quest.AttemptStatus)
	assert.Equal(t, 0, attempt.RevealedHintsCount)
}

// TestCheck: результат чекера сохраняется в историю и в попытку; пройденный
// квест получает статус completed и время прохождения, проваленный — failed без него.
func TestCheck(t *testing.T) {
	t.Parallel()

	topology := json.RawMessage(`{"nodes":[],"links":[]}`)

	tests := []struct {
		name          string
		passed        bool
		wantStatus    model.AttemptStatus
		wantCompleted bool
	}{
		{name: "решение засчитано", passed: true, wantStatus: model.AttemptCompleted, wantCompleted: true},
		{name: "решение не засчитано", passed: false, wantStatus: model.AttemptFailed},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()
			svc, repo, checker := newService(t)
			result := model.QuestResult{Passed: tc.passed, Score: 50}

			repo.EXPECT().GetAttemptForOwner(ctx, attemptID, userID).
				Return(model.Attempt{ID: attemptID, QuestID: questID}, nil).Once()
			// Seed по умолчанию — 7: так одно решение всегда проверяется одинаково.
			checker.EXPECT().Check(ctx, mock.MatchedBy(func(q model.Quest) bool { return q.ID == questID }), topology, int64(7)).
				Return(result).Once()
			repo.EXPECT().InsertCheckResult(ctx, mock.AnythingOfType("string"), attemptID, userID, result).Return(nil).Once()
			repo.EXPECT().UpdateAttemptAfterCheck(ctx, attemptID, userID, tc.wantStatus, result, mock.Anything).
				Run(func(_ context.Context, _, _ string, _ model.AttemptStatus, _ model.QuestResult, completedAt *time.Time) {
					assert.Equal(t, tc.wantCompleted, completedAt != nil, "время прохождения")
				}).
				Return(model.Attempt{ID: attemptID, Status: tc.wantStatus}, nil).Once()

			attempt, got, err := svc.Check(ctx, userID, attemptID, input.CheckQuestInput{Topology: topology})

			require.NoError(t, err)
			assert.Equal(t, tc.wantStatus, attempt.Status)
			assert.Equal(t, result, got)
		})
	}
}

// TestCheckWithoutTopology: пустая топология — 422 после поиска попытки
// (чужая попытка с пустым телом даст 404, а не 422).
func TestCheckWithoutTopology(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, repo, _ := newService(t)

	repo.EXPECT().GetAttemptForOwner(ctx, attemptID, userID).
		Return(model.Attempt{ID: attemptID, QuestID: questID}, nil).Once()

	_, _, err := svc.Check(ctx, userID, attemptID, input.CheckQuestInput{})

	var validationErr *errs.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Equal(t, "topology is required", validationErr.Message)
}

// TestListStatuses: квесты без попыток — not_started, с попыткой — статус последней.
func TestListStatuses(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, repo, _ := newService(t)

	repo.EXPECT().LatestAttemptStatuses(ctx, userID).
		Return(map[string]model.AttemptStatus{questID: model.AttemptCompleted}, nil).Once()

	quests, err := svc.List(ctx, userID)
	require.NoError(t, err)
	require.Len(t, quests, len(catalog.Catalog()))

	for _, quest := range quests {
		want := model.AttemptNotStarted
		if quest.ID == questID {
			want = model.AttemptCompleted
		}

		assert.Equal(t, want, quest.AttemptStatus, quest.ID)
	}
}

// TestGetBySlugAndMissing: квест ищется и по ID, и по slug; неизвестный — ErrQuestNotFound.
func TestGetBySlugAndMissing(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	svc, repo, _ := newService(t)

	repo.EXPECT().LatestAttemptStatuses(ctx, userID).Return(map[string]model.AttemptStatus{}, nil).Once()

	quest, err := svc.Get(ctx, userID, "pochini-dns-lookup")
	require.NoError(t, err)
	assert.Equal(t, questID, quest.ID)
	assert.Equal(t, model.AttemptNotStarted, quest.AttemptStatus)

	_, err = svc.Get(ctx, userID, "quest-nope")
	assert.ErrorIs(t, err, errs.ErrQuestNotFound)
}
