package nats

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

// Ping проверяет, что брокер отвечает: PING → PONG по общему соединению.
// Если соединения нет, сначала подключается. Сбой записи или чтения рвёт
// соединение — следующая операция откроет новое.
func (c *Client) Ping(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnectedLocked(ctx); err != nil {
		return err
	}

	if err := c.setConnDeadline(ctx, c.conn); err != nil {
		return err
	}

	if _, err := fmt.Fprint(c.conn, "PING\r\n"); err != nil {
		c.closeLocked()
		return fmt.Errorf("write nats ping: %w", err)
	}

	if err := readPONG(c.reader); err != nil {
		c.closeLocked()
		return err
	}

	// Дедлайн снимаем: соединение долгоживущее, следующей операции нужен свой срок.
	clearDeadline(c.conn)

	return nil
}

// Publish публикует payload в subject.
//
// После PUB сразу идёт PING: NATS подтверждает PONG только после того, как
// обработал все предыдущие команды соединения. Дождались PONG — значит, PUB
// принят сервером (verbose-режим с +OK на каждую команду выключен в CONNECT).
func (c *Client) Publish(ctx context.Context, subject string, payload []byte) error {
	if err := validateSubject(subject); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnectedLocked(ctx); err != nil {
		return err
	}

	if err := c.setConnDeadline(ctx, c.conn); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(c.conn, "PUB %s %d\r\n%s\r\nPING\r\n", subject, len(payload), payload); err != nil {
		c.closeLocked()
		return fmt.Errorf("publish nats message: %w", err)
	}

	if err := readPONG(c.reader); err != nil {
		c.closeLocked()
		return err
	}

	clearDeadline(c.conn)

	return nil
}

// PublishAndConsume — глубокая проверка брокера: подписаться на subject,
// опубликовать в него payload и дождаться, что сообщение вернулось.
//
// Работает на отдельном, одноразовом соединении: подписка на общем соединении
// подмешала бы MSG в ответы для Publish и Ping. Порядок ответов сервера
// гарантирован протоколом: MSG по нашей подписке приходит раньше PONG на PING,
// отправленный после PUB. Поэтому PONG без MSG — это «сообщение потерялось».
func (c *Client) PublishAndConsume(ctx context.Context, subject string, payload []byte) error {
	if err := validateSubject(subject); err != nil {
		return err
	}

	conn, reader, err := c.dial(ctx)
	if err != nil {
		return err
	}
	defer conn.Close()

	if err := c.setConnDeadline(ctx, conn); err != nil {
		return err
	}

	if _, err := fmt.Fprintf(conn, "SUB %s 1\r\nPUB %s %d\r\n%s\r\nPING\r\n", subject, subject, len(payload), payload); err != nil {
		return fmt.Errorf("write nats deep check: %w", err)
	}

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read nats deep check: %w", err)
		}

		line = strings.TrimSpace(line)

		switch {
		case line == "PONG":
			return errors.New("nats deep check did not receive message before pong")
		case strings.HasPrefix(line, "MSG "):
			return readMessagePayload(reader, line, payload)
		case strings.HasPrefix(line, "-ERR"):
			return fmt.Errorf("nats error: %s", line)
		}
	}
}

// readMessagePayload читает тело MSG и сравнивает его с отправленным.
//
// Заголовок MSG: "MSG <subject> <sid> <size>"; за ним идёт тело длиной size
// и завершающие \r\n — поэтому читаем size+2 байта.
func readMessagePayload(reader interface{ Read(p []byte) (int, error) }, header string, want []byte) error {
	parts := strings.Fields(header)
	if len(parts) < 4 {
		return fmt.Errorf("invalid nats message header %q", header)
	}

	size, err := strconv.Atoi(parts[3])
	if err != nil {
		return fmt.Errorf("invalid nats message size: %w", err)
	}

	data := make([]byte, size+2)
	if _, err := reader.Read(data); err != nil {
		return fmt.Errorf("read nats message payload: %w", err)
	}

	if string(data[:size]) != string(want) {
		return errors.New("nats publish/consume payload mismatch")
	}

	return nil
}
