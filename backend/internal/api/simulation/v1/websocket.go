package v1

import (
	"crypto/sha1" //nolint:gosec // G505: SHA-1 требует сам протокол WebSocket (RFC 6455), это не защита данных
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
)

// Минимальная серверная часть WebSocket (RFC 6455) без внешней библиотеки:
// сервису нужно только отправить текстовые кадры и закрыть соединение.
// Кадры от клиента не читаются.

// websocketGUID — константа из RFC 6455 для вычисления Sec-WebSocket-Accept.
const websocketGUID = "258EAFA5-E914-47DA-95CA-C5AB0DC85B11"

// Коды операций кадра WebSocket.
const (
	opcodeText  byte = 0x1
	opcodeClose byte = 0x8
	finBit      byte = 0x80 // кадр — последний во фрагментированном сообщении
)

// Границы длины тела, после которых длина кодируется в 2 и в 8 байт.
const (
	maxShortPayload  = 125
	maxMediumPayload = 65535
	lengthMedium     = 126
	lengthLong       = 127
)

// upgradeWebSocket переключает HTTP-соединение на WebSocket: проверяет
// заголовки upgrade, забирает TCP-соединение у net/http (Hijack) и отвечает
// 101 Switching Protocols. Дальше соединением владеет вызывающий и сам его закрывает.
//
// Ошибки до hijack — обычные HTTP-ответы текстом (http.Error).
func upgradeWebSocket(w http.ResponseWriter, r *http.Request) (net.Conn, error) {
	if !strings.EqualFold(r.Header.Get("Upgrade"), "websocket") {
		http.Error(w, "websocket upgrade required", http.StatusBadRequest)
		return nil, errors.New("клиент не запросил upgrade до websocket")
	}

	key := r.Header.Get("Sec-WebSocket-Key")
	if key == "" {
		http.Error(w, "websocket key required", http.StatusBadRequest)
		return nil, errors.New("нет заголовка Sec-WebSocket-Key")
	}

	hijacker, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "websocket hijack unsupported", http.StatusInternalServerError)
		return nil, errors.New("ResponseWriter не поддерживает Hijack")
	}

	conn, rw, err := hijacker.Hijack()
	if err != nil {
		return nil, fmt.Errorf("забрать соединение у net/http: %w", err)
	}

	if _, err := fmt.Fprintf(rw, "HTTP/1.1 101 Switching Protocols\r\nUpgrade: websocket\r\nConnection: Upgrade\r\nSec-WebSocket-Accept: %s\r\n\r\n", websocketAccept(key)); err != nil {
		_ = conn.Close() //nolint:gosec // G104: соединение выбрасывается, важна ошибка записи
		return nil, fmt.Errorf("записать ответ 101: %w", err)
	}

	if err := rw.Flush(); err != nil {
		_ = conn.Close() //nolint:gosec // G104: соединение выбрасывается, важна ошибка отправки
		return nil, fmt.Errorf("отправить ответ 101: %w", err)
	}

	return conn, nil
}

// websocketAccept — значение Sec-WebSocket-Accept: base64(SHA-1(key + GUID)).
// Так клиент убеждается, что сервер действительно понимает WebSocket.
func websocketAccept(key string) string {
	sum := sha1.Sum([]byte(key + websocketGUID)) //nolint:gosec // G401: см. комментарий к импорту

	return base64.StdEncoding.EncodeToString(sum[:])
}

// writeWebSocketText отправляет текстовый кадр.
func writeWebSocketText(conn net.Conn, payload []byte) error {
	return writeWebSocketFrame(conn, opcodeText, payload)
}

// writeWebSocketClose отправляет кадр закрытия без кода причины.
func writeWebSocketClose(conn net.Conn) error {
	return writeWebSocketFrame(conn, opcodeClose, nil)
}

// writeWebSocketFrame отправляет один немаскированный кадр (сервер кадры
// не маскирует). Длина тела кодируется по RFC 6455: до 125 — в самом
// заголовке, до 65535 — в 2 байтах, больше — в 8 байтах (big-endian).
// byte(x) берёт младший байт — так и раскладывается число по байтам,
// переполнения здесь нет по построению.
func writeWebSocketFrame(conn net.Conn, opcode byte, payload []byte) error {
	header := []byte{finBit | opcode}
	length := len(payload)

	switch {
	case length <= maxShortPayload:
		header = append(header, byte(length))
	case length <= maxMediumPayload:
		header = append(header, lengthMedium, byte(length>>8), byte(length)) //nolint:gosec // G115: побайтовая раскладка, см. выше
	default:
		header = append(header, lengthLong,
			byte(length>>56), byte(length>>48), byte(length>>40), byte(length>>32), //nolint:gosec // G115: побайтовая раскладка
			byte(length>>24), byte(length>>16), byte(length>>8), byte(length)) //nolint:gosec // G115: побайтовая раскладка
	}

	if _, err := conn.Write(header); err != nil {
		return err
	}

	_, err := conn.Write(payload)

	return err
}
