package tests

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/domain/catalog"
)

// TestCatalogLearningMetadata: в каталоге не меньше 20 квестов с разбросом
// сложности, и у каждого после обогащения есть цель, проверки, топология,
// подсказки (4, а у hard — 5), объяснение решения без смеси языков и глоссарий.
func TestCatalogLearningMetadata(t *testing.T) {
	t.Parallel()

	quests := catalog.Catalog()
	assert.GreaterOrEqual(t, len(quests), 20)

	counts := map[model.Difficulty]int{}
	v2Counts := map[model.Difficulty]int{}

	for _, quest := range quests {
		counts[quest.Difficulty]++
		if strings.HasPrefix(quest.ID, "quest-v2-") {
			v2Counts[quest.Difficulty]++
		}

		assert.NotEmpty(t, quest.Goal, quest.ID)
		assert.NotEmpty(t, quest.ExpectedChecks, quest.ID)
		assert.NotEmpty(t, quest.InitialTopology, quest.ID)
		assert.NotEmpty(t, quest.GlossaryTerms, quest.ID)
		assert.NotEmpty(t, quest.AfterSolution, quest.ID)
		assert.NotContains(t, quest.AfterSolution, "backend-checker", quest.ID)
		assert.NotContains(t, quest.AfterSolution, "canvas", quest.ID)

		minHints := 4
		if quest.Difficulty == model.DifficultyHard {
			minHints = 5
		}

		assert.GreaterOrEqual(t, len(quest.ProgressiveHints), minHints, quest.ID)
	}

	assert.GreaterOrEqual(t, counts[model.DifficultyEasy], 6)
	assert.GreaterOrEqual(t, counts[model.DifficultyMedium], 7)
	assert.GreaterOrEqual(t, counts[model.DifficultyHard], 7)
	assert.Equal(t, map[model.Difficulty]int{
		model.DifficultyEasy:   3,
		model.DifficultyMedium: 4,
		model.DifficultyHard:   3,
	}, v2Counts)
}

// TestCatalogReturnsFreshCopies: правка результата не меняет следующий вызов —
// сервис квестов и тесты могут свободно менять свою копию.
func TestCatalogReturnsFreshCopies(t *testing.T) {
	t.Parallel()

	first := catalog.Catalog()
	first[0].Title = "changed"
	first[0].ExpectedChecks[0].Title = "changed"

	second := catalog.Catalog()
	assert.NotEqual(t, "changed", second[0].Title)
	assert.NotEqual(t, "changed", second[0].ExpectedChecks[0].Title)
}

// TestCatalogIDsAreUnique: ID и slug квестов уникальны — по ним квест ищется.
func TestCatalogIDsAreUnique(t *testing.T) {
	t.Parallel()

	seen := map[string]bool{}

	for _, quest := range catalog.Catalog() {
		assert.False(t, seen[quest.ID], "повтор ID %s", quest.ID)
		assert.False(t, seen[quest.Slug], "повтор slug %s", quest.Slug)
		seen[quest.ID] = true
		seen[quest.Slug] = true
	}
}
