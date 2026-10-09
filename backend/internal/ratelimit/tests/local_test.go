package tests

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/netquest/netquest/backend/internal/ratelimit"
)

// TestLocalLimiter: в окне пропускается ровно limit запросов с ключа,
// ключи считаются раздельно, лимит 0 — без ограничений.
func TestLocalLimiter(t *testing.T) {
	t.Parallel()

	limiter := ratelimit.NewLocalLimiter(3)

	for i := range 3 {
		assert.True(t, limiter.Allow("10.0.0.1"), "запрос %d", i+1)
	}

	assert.False(t, limiter.Allow("10.0.0.1"), "четвёртый запрос сверх лимита")
	assert.True(t, limiter.Allow("10.0.0.2"), "у другого IP свой счётчик")

	unlimited := ratelimit.NewLocalLimiter(0)
	for range 100 {
		assert.True(t, unlimited.Allow("10.0.0.1"))
	}
}

// TestLocalLimiterConcurrent: параллельные запросы не теряют инкременты —
// из 100 одновременных пропускается ровно limit.
func TestLocalLimiterConcurrent(t *testing.T) {
	t.Parallel()

	limiter := ratelimit.NewLocalLimiter(10)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		allowed int
	)

	for range 100 {
		wg.Add(1)

		go func() {
			defer wg.Done()

			if limiter.Allow("10.0.0.1") {
				mu.Lock()
				allowed++
				mu.Unlock()
			}
		}()
	}

	wg.Wait()

	assert.Equal(t, 10, allowed)
}
