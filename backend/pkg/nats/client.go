// Package nats — минимальный клиент NATS поверх TCP без внешних зависимостей.
//
// NetQuest нужно от NATS немного: опубликовать событие симуляции (PUB) и
// проверить брокер в health-check (PING, а в глубокой проверке — SUB+PUB).
// Для этого хватает текстового протокола NATS: команды — строки, оканчивающиеся
// на \r\n, ответы сервера — +OK, PONG, MSG, -ERR и INFO.
//
//	клиент → сервер: CONNECT {...}\r\n PING\r\n
//	сервер → клиент: INFO {...}\r\n (сразу после TCP-подключения), PONG\r\n
//
// Тексты ошибок клиента оставлены прежними (на английском): они попадают
// в ответы /health/ready и /health/deep, и мониторинг может на них опираться.
package nats

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// defaultTimeout — срок операции, если ни ctx, ни конфиг его не задали.
const defaultTimeout = 2 * time.Second

// Client — одно долгоживущее TCP-соединение с NATS.
//
// Соединение общее для всех горутин, а протокол строго последовательный:
// после PUB+PING клиент читает ответы до PONG. Два одновременных запроса
// перемешали бы свои команды и ответы, поэтому каждая операция держит mu
// всё время — от записи команды до чтения PONG.
type Client struct {
	addr    string
	timeout time.Duration

	mu     sync.Mutex
	conn   net.Conn      // nil — соединения нет (ещё не было или оборвалось)
	reader *bufio.Reader // буферизованное чтение того же conn
}

// New разбирает адрес и сразу пытается подключиться.
//
// Ошибка подключения не делает клиента бесполезным: New возвращает и клиента,
// и ошибку. Вызывающий может работать в degraded-режиме — каждая следующая
// операция сама попробует переподключиться (ensureConnectedLocked). А вот
// неразборчивый URL — это nil и ошибка: подключаться некуда.
func New(rawURL string, timeout time.Duration) (*Client, error) {
	addr, err := parseAddr(rawURL)
	if err != nil {
		return nil, err
	}

	client := &Client{addr: addr, timeout: timeout}
	if err := client.Connect(context.Background()); err != nil {
		return client, fmt.Errorf("connect nats: %w", err)
	}

	return client, nil
}

// Connect (пере)открывает соединение: старое закрывается, если было.
func (c *Client) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	return c.connectLocked(ctx)
}

// Close закрывает соединение. Повторный вызов безопасен.
func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.closeLocked()
}

// ensureConnectedLocked подключается, если соединения нет. Вызывать под mu.
func (c *Client) ensureConnectedLocked(ctx context.Context) error {
	if c.conn != nil {
		return nil
	}

	return c.connectLocked(ctx)
}

// connectLocked заменяет текущее соединение новым. Вызывать под mu.
func (c *Client) connectLocked(ctx context.Context) error {
	c.closeLocked()

	conn, reader, err := c.dial(ctx)
	if err != nil {
		return err
	}

	c.conn = conn
	c.reader = reader

	return nil
}

// closeLocked закрывает соединение и забывает его. Вызывать под mu.
func (c *Client) closeLocked() {
	if c.conn != nil {
		closeQuietly(c.conn)
		c.conn = nil
		c.reader = nil
	}
}
