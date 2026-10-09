// Package quest — сервисный слой Quest Mode: каталог квестов со статусами
// пользователя, попытки, проверка решения, сброс и прогрессивные подсказки.
//
// Место в цепочке: api/quest/v1 → service/application/quest →
// {repository/quest, domain/checker}; каталог приходит из domain/catalog.
//
// Каталог живёт в коде. Перед каждой операцией сервис синхронизирует его
// в таблицу quests (ensureCatalog): попытки ссылаются на quests.id внешним
// ключом, и новый квест из свежей версии кода должен оказаться в базе раньше,
// чем на него появится первая попытка.
package quest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/netquest/netquest/backend/internal/model"
)

// service — сервис квестов.
type service struct {
	questRepository QuestRepository
	checker         QuestChecker
	// catalog — каталог квестов. Сервис его не меняет: наружу отдаются копии.
	catalog []model.Quest
}

// New создаёт сервис квестов над каталогом.
func New(questRepository QuestRepository, checker QuestChecker, catalog []model.Quest) *service {
	return &service{
		questRepository: questRepository,
		checker:         checker,
		catalog:         catalog,
	}
}

// ensureCatalog синхронизирует каталог в таблицу quests (UPSERT каждого квеста).
func (s *service) ensureCatalog(ctx context.Context) error {
	if err := s.questRepository.UpsertQuests(ctx, s.catalog); err != nil {
		return fmt.Errorf("синхронизировать каталог квестов: %w", err)
	}

	return nil
}

// findQuest ищет квест по ID или slug и возвращает его копию.
func (s *service) findQuest(idOrSlug string) (model.Quest, bool) {
	needle := strings.TrimSpace(idOrSlug)

	for _, quest := range s.catalog {
		if quest.ID == needle || quest.Slug == needle {
			return cloneQuest(quest), true
		}
	}

	return model.Quest{}, false
}

// cloneCatalog копирует каталог для ответа (см. cloneQuest).
func cloneCatalog(items []model.Quest) []model.Quest {
	result := make([]model.Quest, len(items))
	for i, item := range items {
		result[i] = cloneQuest(item)
	}

	return result
}

// cloneQuest копирует квест вместе со срезами. Присваивание структуры копирует
// только заголовки срезов — элементы остались бы общими с каталогом, который
// делят все запросы, и правка ответа одному пользователю задела бы остальных.
func cloneQuest(item model.Quest) model.Quest {
	cloned := item
	cloned.LearningObjectives = append([]string{}, item.LearningObjectives...)
	cloned.ExpectedChecks = append([]model.CheckSpec{}, item.ExpectedChecks...)
	cloned.Hints = append([]string{}, item.Hints...)
	cloned.ProgressiveHints = append([]model.ProgressiveHint{}, item.ProgressiveHints...)
	cloned.GlossaryTerms = append([]model.GlossaryTerm{}, item.GlossaryTerms...)

	if item.InitialTopology != nil {
		cloned.InitialTopology = append(json.RawMessage{}, item.InitialTopology...)
	}

	return cloned
}

// withAttemptStatus проставляет квесту статус последней попытки пользователя,
// а без попыток — not_started.
func withAttemptStatus(quest *model.Quest, statuses map[string]model.AttemptStatus) {
	if status, ok := statuses[quest.ID]; ok {
		quest.AttemptStatus = status
		return
	}

	quest.AttemptStatus = model.AttemptNotStarted
}
