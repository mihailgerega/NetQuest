package quests

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/netquest/netquest/backend/pkg/apperrors"
	"github.com/netquest/netquest/backend/pkg/idgen"
)

type Store interface {
	UpsertQuests(ctx context.Context, quests []Quest) error
	LatestAttemptStatuses(ctx context.Context, userID string) (map[string]AttemptStatus, error)
	CreateAttempt(ctx context.Context, attempt Attempt) (Attempt, error)
	GetAttemptForOwner(ctx context.Context, attemptID, userID string) (Attempt, error)
	UpdateAttemptAfterCheck(ctx context.Context, attemptID, userID string, status AttemptStatus, result Result) (Attempt, error)
	InsertCheckResult(ctx context.Context, id, attemptID, userID string, result Result) error
	ResetAttempt(ctx context.Context, attemptID, userID string) (Attempt, error)
	UpdateRevealedHintsCount(ctx context.Context, attemptID, userID string, count int) (Attempt, error)
}

type Service struct {
	store   Store
	checker Checker
	catalog []Quest
}

func NewService(store Store, checker Checker) *Service {
	return &Service{store: store, checker: checker, catalog: Catalog()}
}

func (s *Service) List(ctx context.Context, userID string) ([]Quest, error) {
	if err := s.ensureCatalog(ctx); err != nil {
		return nil, err
	}
	statuses, err := s.store.LatestAttemptStatuses(ctx, userID)
	if err != nil {
		return nil, err
	}
	quests := cloneCatalog(s.catalog)
	for i := range quests {
		if status, ok := statuses[quests[i].ID]; ok {
			quests[i].AttemptStatus = status
		} else {
			quests[i].AttemptStatus = AttemptNotStarted
		}
	}
	return quests, nil
}

func (s *Service) Get(ctx context.Context, userID, questID string) (Quest, error) {
	if err := s.ensureCatalog(ctx); err != nil {
		return Quest{}, err
	}
	quest, ok := s.findQuest(questID)
	if !ok {
		return Quest{}, apperrors.NotFound("quest not found")
	}
	statuses, err := s.store.LatestAttemptStatuses(ctx, userID)
	if err != nil {
		return Quest{}, err
	}
	if status, ok := statuses[quest.ID]; ok {
		quest.AttemptStatus = status
	} else {
		quest.AttemptStatus = AttemptNotStarted
	}
	return quest, nil
}

func (s *Service) Start(ctx context.Context, userID, questID string) (StartResponse, error) {
	if err := s.ensureCatalog(ctx); err != nil {
		return StartResponse{}, err
	}
	quest, ok := s.findQuest(questID)
	if !ok {
		return StartResponse{}, apperrors.NotFound("quest not found")
	}
	id, err := idgen.NewUUID()
	if err != nil {
		return StartResponse{}, err
	}
	attempt, err := s.store.CreateAttempt(ctx, Attempt{
		ID:            id,
		QuestID:       quest.ID,
		UserID:        userID,
		Status:        AttemptInProgress,
		AttemptsCount: 0,
	})
	if err != nil {
		return StartResponse{}, err
	}
	quest.AttemptStatus = attempt.Status
	return StartResponse{Quest: quest, Attempt: attempt}, nil
}

func (s *Service) GetAttempt(ctx context.Context, userID, attemptID string) (Attempt, error) {
	return s.store.GetAttemptForOwner(ctx, attemptID, userID)
}

func (s *Service) Check(ctx context.Context, userID, attemptID string, req CheckRequest) (CheckResponse, error) {
	if err := s.ensureCatalog(ctx); err != nil {
		return CheckResponse{}, err
	}
	attempt, err := s.store.GetAttemptForOwner(ctx, attemptID, userID)
	if err != nil {
		return CheckResponse{}, err
	}
	quest, ok := s.findQuest(attempt.QuestID)
	if !ok {
		return CheckResponse{}, apperrors.NotFound("quest not found")
	}
	if len(req.Topology) == 0 {
		return CheckResponse{}, apperrors.Validation("topology is required", nil)
	}
	seed := int64(7)
	if req.Seed != nil {
		seed = *req.Seed
	}
	result := s.checker.Check(ctx, quest, req.Topology, seed)
	status := AttemptFailed
	if result.Passed {
		status = AttemptCompleted
	}
	resultID, err := idgen.NewUUID()
	if err != nil {
		return CheckResponse{}, err
	}
	if err := s.store.InsertCheckResult(ctx, resultID, attempt.ID, userID, result); err != nil {
		return CheckResponse{}, err
	}
	attempt, err = s.store.UpdateAttemptAfterCheck(ctx, attempt.ID, userID, status, result)
	if err != nil {
		return CheckResponse{}, err
	}
	return CheckResponse{Attempt: attempt, Result: result}, nil
}

func (s *Service) Reset(ctx context.Context, userID, attemptID string) (ResetResponse, error) {
	if err := s.ensureCatalog(ctx); err != nil {
		return ResetResponse{}, err
	}
	attempt, err := s.store.ResetAttempt(ctx, attemptID, userID)
	if err != nil {
		return ResetResponse{}, err
	}
	quest, ok := s.findQuest(attempt.QuestID)
	if !ok {
		return ResetResponse{}, apperrors.NotFound("quest not found")
	}
	quest.AttemptStatus = attempt.Status
	return ResetResponse{Quest: quest, Attempt: attempt}, nil
}

func (s *Service) RevealHint(ctx context.Context, userID, attemptID string, req RevealHintRequest) (RevealHintResponse, error) {
	if err := s.ensureCatalog(ctx); err != nil {
		return RevealHintResponse{}, err
	}
	attempt, err := s.store.GetAttemptForOwner(ctx, attemptID, userID)
	if err != nil {
		return RevealHintResponse{}, err
	}
	quest, ok := s.findQuest(attempt.QuestID)
	if !ok {
		return RevealHintResponse{}, apperrors.NotFound("quest not found")
	}
	count := req.RevealedHintsCount
	if count < 0 {
		count = 0
	}
	if len(quest.ProgressiveHints) > 0 && count > len(quest.ProgressiveHints) {
		count = len(quest.ProgressiveHints)
	}
	if count < attempt.RevealedHintsCount {
		count = attempt.RevealedHintsCount
	}
	attempt, err = s.store.UpdateRevealedHintsCount(ctx, attemptID, userID, count)
	if err != nil {
		return RevealHintResponse{}, err
	}
	return RevealHintResponse{Attempt: attempt}, nil
}

func (s *Service) ensureCatalog(ctx context.Context) error {
	return s.store.UpsertQuests(ctx, s.catalog)
}

func (s *Service) findQuest(idOrSlug string) (Quest, bool) {
	needle := strings.TrimSpace(idOrSlug)
	for _, quest := range s.catalog {
		if quest.ID == needle || quest.Slug == needle {
			return cloneQuest(quest), true
		}
	}
	return Quest{}, false
}

func cloneCatalog(items []Quest) []Quest {
	result := make([]Quest, len(items))
	for i, item := range items {
		result[i] = cloneQuest(item)
	}
	return result
}

func cloneQuest(item Quest) Quest {
	cloned := item
	cloned.LearningObjectives = append([]string{}, item.LearningObjectives...)
	cloned.ExpectedChecks = append([]CheckSpec{}, item.ExpectedChecks...)
	cloned.Hints = append([]string{}, item.Hints...)
	cloned.ProgressiveHints = append([]ProgressiveHint{}, item.ProgressiveHints...)
	cloned.GlossaryTerms = append([]GlossaryTerm{}, item.GlossaryTerms...)
	if item.InitialTopology != nil {
		cloned.InitialTopology = append(json.RawMessage{}, item.InitialTopology...)
	}
	return cloned
}
