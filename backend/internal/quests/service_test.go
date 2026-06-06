package quests

import (
	"context"
	"encoding/json"
	"testing"
	"time"
)

func TestRevealHintPersistsAndDoesNotDecrease(t *testing.T) {
	store := newFakeQuestStore()
	service := NewService(store, Checker{})
	attempt := store.seedAttempt(t, "quest-dns-lookup")

	res, err := service.RevealHint(context.Background(), attempt.UserID, attempt.ID, RevealHintRequest{RevealedHintsCount: 2})
	if err != nil {
		t.Fatalf("reveal hint: %v", err)
	}
	if res.Attempt.RevealedHintsCount != 2 {
		t.Fatalf("expected 2 revealed hints, got %d", res.Attempt.RevealedHintsCount)
	}

	res, err = service.RevealHint(context.Background(), attempt.UserID, attempt.ID, RevealHintRequest{RevealedHintsCount: 1})
	if err != nil {
		t.Fatalf("reveal hint second call: %v", err)
	}
	if res.Attempt.RevealedHintsCount != 2 {
		t.Fatalf("revealed hints count should not decrease, got %d", res.Attempt.RevealedHintsCount)
	}
}

func TestRevealHintClampsToQuestHintCount(t *testing.T) {
	store := newFakeQuestStore()
	service := NewService(store, Checker{})
	attempt := store.seedAttempt(t, "quest-dns-lookup")
	quest := questByID(t, attempt.QuestID)

	res, err := service.RevealHint(context.Background(), attempt.UserID, attempt.ID, RevealHintRequest{RevealedHintsCount: 999})
	if err != nil {
		t.Fatalf("reveal hint: %v", err)
	}
	if res.Attempt.RevealedHintsCount != len(quest.ProgressiveHints) {
		t.Fatalf("expected clamp to %d, got %d", len(quest.ProgressiveHints), res.Attempt.RevealedHintsCount)
	}
}

func TestResetAttemptClearsRevealedHints(t *testing.T) {
	store := newFakeQuestStore()
	service := NewService(store, Checker{})
	attempt := store.seedAttempt(t, "quest-dns-lookup")
	attempt.RevealedHintsCount = 3
	store.attempts[attempt.ID] = attempt

	res, err := service.Reset(context.Background(), attempt.UserID, attempt.ID)
	if err != nil {
		t.Fatalf("reset attempt: %v", err)
	}
	if res.Attempt.RevealedHintsCount != 0 {
		t.Fatalf("expected reset to clear revealed hints, got %d", res.Attempt.RevealedHintsCount)
	}
	if res.Attempt.AttemptsCount != 0 || res.Attempt.Status != AttemptInProgress {
		t.Fatalf("unexpected reset attempt state: %#v", res.Attempt)
	}
}

type fakeQuestStore struct {
	catalog  []Quest
	attempts map[string]Attempt
}

func newFakeQuestStore() *fakeQuestStore {
	return &fakeQuestStore{attempts: map[string]Attempt{}}
}

func (s *fakeQuestStore) UpsertQuests(_ context.Context, quests []Quest) error {
	s.catalog = cloneCatalog(quests)
	return nil
}

func (s *fakeQuestStore) LatestAttemptStatuses(_ context.Context, _ string) (map[string]AttemptStatus, error) {
	statuses := map[string]AttemptStatus{}
	for _, attempt := range s.attempts {
		statuses[attempt.QuestID] = attempt.Status
	}
	return statuses, nil
}

func (s *fakeQuestStore) CreateAttempt(_ context.Context, attempt Attempt) (Attempt, error) {
	now := time.Now().UTC()
	attempt.CreatedAt = now
	attempt.UpdatedAt = now
	s.attempts[attempt.ID] = attempt
	return attempt, nil
}

func (s *fakeQuestStore) GetAttemptForOwner(_ context.Context, attemptID, userID string) (Attempt, error) {
	attempt := s.attempts[attemptID]
	if attempt.UserID != userID {
		return Attempt{}, errFakeNotFound
	}
	return attempt, nil
}

func (s *fakeQuestStore) UpdateAttemptAfterCheck(_ context.Context, attemptID, userID string, status AttemptStatus, result Result) (Attempt, error) {
	attempt, err := s.GetAttemptForOwner(context.Background(), attemptID, userID)
	if err != nil {
		return Attempt{}, err
	}
	attempt.Status = status
	attempt.AttemptsCount++
	attempt.UpdatedAt = time.Now().UTC()
	raw, _ := json.Marshal(result)
	attempt.LastCheckResult = raw
	s.attempts[attemptID] = attempt
	return attempt, nil
}

func (s *fakeQuestStore) InsertCheckResult(_ context.Context, _, _, _ string, _ Result) error {
	return nil
}

func (s *fakeQuestStore) ResetAttempt(_ context.Context, attemptID, userID string) (Attempt, error) {
	attempt, err := s.GetAttemptForOwner(context.Background(), attemptID, userID)
	if err != nil {
		return Attempt{}, err
	}
	attempt.Status = AttemptInProgress
	attempt.AttemptsCount = 0
	attempt.RevealedHintsCount = 0
	attempt.LastCheckResult = nil
	attempt.CompletedAt = nil
	attempt.UpdatedAt = time.Now().UTC()
	s.attempts[attemptID] = attempt
	return attempt, nil
}

func (s *fakeQuestStore) UpdateRevealedHintsCount(_ context.Context, attemptID, userID string, count int) (Attempt, error) {
	attempt, err := s.GetAttemptForOwner(context.Background(), attemptID, userID)
	if err != nil {
		return Attempt{}, err
	}
	if count > attempt.RevealedHintsCount {
		attempt.RevealedHintsCount = count
	}
	attempt.UpdatedAt = time.Now().UTC()
	s.attempts[attemptID] = attempt
	return attempt, nil
}

func (s *fakeQuestStore) seedAttempt(t *testing.T, questID string) Attempt {
	t.Helper()
	attempt := Attempt{
		ID:            "attempt-1",
		QuestID:       questID,
		UserID:        "user-1",
		Status:        AttemptInProgress,
		AttemptsCount: 0,
		CreatedAt:     time.Now().UTC(),
		UpdatedAt:     time.Now().UTC(),
	}
	s.attempts[attempt.ID] = attempt
	return attempt
}

type fakeNotFoundError struct{}

func (fakeNotFoundError) Error() string { return "not found" }

var errFakeNotFound = fakeNotFoundError{}
