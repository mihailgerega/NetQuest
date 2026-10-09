package ratelimit

import (
	"sync"
	"time"
)

// staleWindowAge — через сколько без запросов окно IP забывается.
const staleWindowAge = 5 * time.Minute

// LocalLimiter — счётчик фиксированного окна в памяти процесса: запасной
// вариант на время недоступности Redis.
//
// Карта окон общая для всех горутин-запросов, поэтому каждый Allow держит mu
// целиком: прочитать окно, увеличить счётчик и записать обратно — одна
// неделимая операция, иначе два параллельных запроса потеряли бы инкремент.
type LocalLimiter struct {
	mu      sync.Mutex
	limit   int
	windows map[string]localWindow
}

// localWindow — счётчик одного IP в текущей минуте.
type localWindow struct {
	Count    int
	ResetAt  time.Time // начало следующей минуты
	LastSeen time.Time // для очистки давно молчащих IP
}

// NewLocalLimiter создаёт счётчик на limit запросов в минуту с ключа.
func NewLocalLimiter(limit int) *LocalLimiter {
	return &LocalLimiter{
		limit:   limit,
		windows: make(map[string]localWindow),
	}
}

// Allow засчитывает запрос с ключа и сообщает, укладывается ли он в лимит.
//
// Заодно чистит карту от IP, молчащих дольше staleWindowAge: без этого
// каждый новый адрес навсегда оставался бы в памяти. Чистка — полный обход
// карты на каждый запрос; при лимите в сотни запросов в минуту это дёшево.
func (l *LocalLimiter) Allow(key string) bool {
	if l.limit <= 0 {
		return true
	}

	now := time.Now()
	resetAt := now.Truncate(time.Minute).Add(time.Minute)

	l.mu.Lock()
	defer l.mu.Unlock()

	for existingKey, window := range l.windows {
		if now.Sub(window.LastSeen) > staleWindowAge {
			delete(l.windows, existingKey)
		}
	}

	window := l.windows[key]
	if now.After(window.ResetAt) {
		window = localWindow{ResetAt: resetAt}
	}

	window.Count++
	window.LastSeen = now
	l.windows[key] = window

	return window.Count <= l.limit
}
