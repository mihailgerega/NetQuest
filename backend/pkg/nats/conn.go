package nats

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
	"time"
)

// dial открывает TCP-соединение и проходит рукопожатие NATS:
//
//	сервер: INFO {...}       — приветствие сразу после подключения
//	клиент: CONNECT {...}    — verbose=false: сервер не шлёт +OK на каждую команду
//	клиент: PING → сервер: PONG — CONNECT принят
//
// При любой ошибке соединение закрывается здесь же, наружу не утекает.
func (c *Client) dial(ctx context.Context) (net.Conn, *bufio.Reader, error) {
	dialer := net.Dialer{}

	conn, err := dialer.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return nil, nil, fmt.Errorf("dial nats: %w", err)
	}

	if err := c.setConnDeadline(ctx, conn); err != nil {
		closeQuietly(conn)
		return nil, nil, err
	}

	reader := bufio.NewReader(conn)

	line, err := reader.ReadString('\n')
	if err != nil {
		closeQuietly(conn)
		return nil, nil, fmt.Errorf("read nats info: %w", err)
	}

	if !strings.HasPrefix(strings.TrimSpace(line), "INFO") {
		closeQuietly(conn)
		return nil, nil, fmt.Errorf("unexpected nats greeting %q", strings.TrimSpace(line))
	}

	if _, err := fmt.Fprint(conn, "CONNECT {\"verbose\":false,\"pedantic\":false}\r\nPING\r\n"); err != nil {
		closeQuietly(conn)
		return nil, nil, fmt.Errorf("write nats connect: %w", err)
	}

	if err := readPONG(reader); err != nil {
		closeQuietly(conn)
		return nil, nil, err
	}

	clearDeadline(conn)

	return conn, reader, nil
}

// setConnDeadline ставит соединению дедлайн: из ctx, а если в ctx его нет —
// timeout клиента (или defaultTimeout). Без дедлайна зависший брокер
// подвесил бы чтение навсегда, а вместе с ним — держателя mu.
func (c *Client) setConnDeadline(ctx context.Context, conn net.Conn) error {
	if conn == nil {
		return errors.New("nats connection is not available")
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		timeout := c.timeout
		if timeout <= 0 {
			timeout = defaultTimeout
		}

		deadline = time.Now().Add(timeout)
	}

	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set nats deadline: %w", err)
	}

	return nil
}

// readPONG читает ответы сервера до PONG. +OK, INFO (сервер может прислать
// обновлённый INFO в любой момент) и прочие строки пропускает, -ERR — ошибка.
func readPONG(reader *bufio.Reader) error {
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return fmt.Errorf("read nats pong: %w", err)
		}

		line = strings.TrimSpace(line)

		switch {
		case line == "PONG":
			return nil
		case strings.HasPrefix(line, "-ERR"):
			return fmt.Errorf("nats error: %s", line)
		default:
			continue
		}
	}
}

// parseAddr превращает URL вида nats://host:port в host:port для net.Dial.
// Строка без схемы, но с портом ("nats:4222") тоже принимается.
func parseAddr(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("parse nats url: %w", err)
	}

	if parsed.Host != "" {
		return parsed.Host, nil
	}

	if strings.Contains(raw, ":") {
		return strings.TrimPrefix(raw, "nats://"), nil
	}

	return "", errors.New("nats url must include host and port")
}

// clearDeadline снимает дедлайн с долгоживущего соединения: следующей операции
// нужен свой срок. Ошибку не проверяем — у живого TCP-соединения её не бывает,
// а мёртвое проявится на следующей записи.
func clearDeadline(conn net.Conn) {
	_ = conn.SetDeadline(time.Time{}) //nolint:gosec // G104: см. комментарий к функции
}

// closeQuietly закрывает соединение, которое уже выбрасывается. Ошибка закрытия
// ничего не меняет: наружу уходит исходная причина, по которой его закрыли.
func closeQuietly(conn net.Conn) {
	_ = conn.Close() //nolint:gosec // G104: см. комментарий к функции
}

// validateSubject не даёт пробелам и переводам строки попасть в subject:
// они разорвали бы текстовую команду PUB и превратили хвост в чужую команду.
func validateSubject(subject string) error {
	if strings.TrimSpace(subject) == "" {
		return errors.New("nats subject is required")
	}

	if strings.ContainsAny(subject, "\r\n\t ") {
		return errors.New("nats subject contains invalid whitespace")
	}

	return nil
}
