package idgen

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

// uuidV5 — формат UUID версии 5 с вариантом RFC 4122.
var uuidV5 = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-5[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

// TestDeterministicUUID: одно имя — один и тот же UUID в формате v5; разные
// имена — разные UUID. Значение для demo-пользователя зафиксировано: если
// алгоритм изменится, demo-аккаунт в уже развёрнутых базах получит другой ID.
func TestDeterministicUUID(t *testing.T) {
	t.Parallel()

	first := DeterministicUUID("netquest-demo-user")

	assert.Equal(t, first, DeterministicUUID("netquest-demo-user"))
	assert.Regexp(t, uuidV5, first)
	assert.NotEqual(t, first, DeterministicUUID("netquest-demo-user-2"))
	assert.Equal(t, "f6771ce3-c0e4-5d2e-882b-903c7af4389e", first)
}
