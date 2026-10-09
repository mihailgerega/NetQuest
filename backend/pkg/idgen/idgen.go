// Package idgen — детерминированные идентификаторы в формате UUID.
//
// Случайные UUID (v4) сервис берёт из github.com/google/uuid. Здесь — только то,
// чего там нет: UUID, который всегда одинаков для одного и того же имени.
// Из него строятся ID событий симуляции и ID demo-пользователя, поэтому
// алгоритм менять нельзя: ID уже сохранённых событий и demo-аккаунта разойдутся
// с новыми.
package idgen

import (
	"crypto/sha1" //nolint:gosec // G505: SHA-1 здесь не для защиты, а как стабильный хеш имени (как в UUID v5)
	"encoding/hex"
)

// DeterministicUUID строит UUID по имени: SHA-1 от имени, затем биты версии (5)
// и варианта (RFC 4122).
//
// Это не uuid.NewSHA1: тот хеширует имя вместе с namespace UUID, а здесь — одно
// имя. Результаты двух алгоритмов не совпадают, поэтому функция своя.
func DeterministicUUID(name string) string {
	sum := sha1.Sum([]byte(name)) //nolint:gosec // G401: см. комментарий к импорту
	b := sum

	// Старшие 4 бита байта 6 — версия UUID (5), старшие 2 бита байта 8 —
	// вариант RFC 4122 (10xx). Остальные биты — из хеша.
	b[6] = (b[6] & 0x0f) | 0x50
	b[8] = (b[8] & 0x3f) | 0x80

	return formatUUID(b[:16])
}

// formatUUID печатает 16 байт в каноническом виде 8-4-4-4-12.
func formatUUID(b []byte) string {
	buf := make([]byte, 36)

	hex.Encode(buf[0:8], b[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], b[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], b[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], b[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], b[10:16])

	return string(buf)
}
