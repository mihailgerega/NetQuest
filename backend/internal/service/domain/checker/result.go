package checker

import (
	"strings"

	"github.com/netquest/netquest/backend/internal/model"
)

// percent — множитель для перевода доли пройденных проверок в Score.
const percent = 100

// allPassed — решение засчитано: проверки есть и все пройдены.
// Квест без проверок не засчитывается никогда.
func allPassed(checks []model.CheckResult) bool {
	if len(checks) == 0 {
		return false
	}

	for _, check := range checks {
		if !check.Passed {
			return false
		}
	}

	return true
}

// score — процент пройденных проверок, округлённый вниз.
func score(checks []model.CheckResult) int {
	if len(checks) == 0 {
		return 0
	}

	passed := 0

	for _, check := range checks {
		if check.Passed {
			passed++
		}
	}

	return int(float64(passed) / float64(len(checks)) * percent)
}

// hintForCheck — подсказка к проваленной проверке: её собственная,
// а если её нет — прогрессивная подсказка квеста.
func hintForCheck(quest model.Quest, spec model.CheckSpec) string {
	if spec.Hint != "" {
		return spec.Hint
	}

	return progressiveHintForCheck(quest, spec.ID)
}

// progressiveHintForCheck — прогрессивная подсказка, связанная с проверкой
// (RelatedCheckID), иначе первая подсказка квеста.
func progressiveHintForCheck(quest model.Quest, checkID string) string {
	for _, hint := range quest.ProgressiveHints {
		if hint.RelatedCheckID == checkID && hint.Body != "" {
			return hint.Body
		}
	}

	if len(quest.ProgressiveHints) > 0 {
		return quest.ProgressiveHints[0].Body
	}

	return ""
}

// appendFirstProgressiveHint добавляет первую непустую прогрессивную подсказку —
// запасной вариант, когда к результату не подобралось ни одной подсказки.
func appendFirstProgressiveHint(hints []string, quest model.Quest) []string {
	for _, hint := range quest.ProgressiveHints {
		if hint.Body != "" {
			return appendUnique(hints, hint.Body)
		}
	}

	return hints
}

// appendUnique добавляет подсказку, если она не пустая и её ещё нет.
func appendUnique(items []string, value string) []string {
	if strings.TrimSpace(value) == "" {
		return items
	}

	for _, item := range items {
		if item == value {
			return items
		}
	}

	return append(items, value)
}

// containsAll — все needles есть в items (пустой needles — true).
func containsAll(items, needles []string) bool {
	for _, needle := range needles {
		if !stringInSlice(items, needle) {
			return false
		}
	}

	return true
}

// containsNone — ни одного из needles нет в items.
func containsNone(items, needles []string) bool {
	for _, needle := range needles {
		if stringInSlice(items, needle) {
			return false
		}
	}

	return true
}

// stringInSlice — есть ли строка в срезе.
func stringInSlice(items []string, needle string) bool {
	for _, item := range items {
		if item == needle {
			return true
		}
	}

	return false
}
