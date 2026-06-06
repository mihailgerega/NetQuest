package storage

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/netquest/netquest/backend/internal/config"
)

type NATSClient struct {
	addr    string
	timeout time.Duration
	mu      sync.Mutex
	conn    net.Conn
	reader  *bufio.Reader
}

func NewNATS(cfg config.NATSConfig) (*NATSClient, error) {
	addr, err := parseNATSAddr(cfg.URL)
	if err != nil {
		return nil, err
	}
	client := &NATSClient{addr: addr, timeout: cfg.Timeout}
	if err := client.Connect(context.Background()); err != nil {
		return client, fmt.Errorf("connect nats: %w", err)
	}
	return client, nil
}

func (c *NATSClient) Connect(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.connectLocked(ctx)
}

func (c *NATSClient) IsConnected() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn != nil
}

func (c *NATSClient) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closeLocked()
}

func (c *NATSClient) Ping(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnectedLocked(ctx); err != nil {
		return err
	}
	if err := c.setDeadline(ctx); err != nil {
		return err
	}
	if _, err := fmt.Fprint(c.conn, "PING\r\n"); err != nil {
		c.closeLocked()
		return fmt.Errorf("write nats ping: %w", err)
	}
	if err := c.readPONGLocked(); err != nil {
		c.closeLocked()
		return err
	}
	_ = c.conn.SetDeadline(noDeadline())
	return nil
}

func (c *NATSClient) Publish(ctx context.Context, subject string, payload []byte) error {
	if err := validateSubject(subject); err != nil {
		return err
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.ensureConnectedLocked(ctx); err != nil {
		return err
	}
	if err := c.setDeadline(ctx); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.conn, "PUB %s %d\r\n%s\r\nPING\r\n", subject, len(payload), payload); err != nil {
		c.closeLocked()
		return fmt.Errorf("publish nats message: %w", err)
	}
	if err := c.readPONGLocked(); err != nil {
		c.closeLocked()
		return err
	}
	_ = c.conn.SetDeadline(noDeadline())
	return nil
}

func (c *NATSClient) PublishAndConsume(ctx context.Context, subject string, payload []byte) error {
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
			return fmt.Errorf("nats deep check did not receive message before pong")
		case strings.HasPrefix(line, "MSG "):
			parts := strings.Fields(line)
			if len(parts) < 4 {
				return fmt.Errorf("invalid nats message header %q", line)
			}
			size, err := strconv.Atoi(parts[3])
			if err != nil {
				return fmt.Errorf("invalid nats message size: %w", err)
			}
			data := make([]byte, size+2)
			if _, err := reader.Read(data); err != nil {
				return fmt.Errorf("read nats message payload: %w", err)
			}
			if string(data[:size]) != string(payload) {
				return fmt.Errorf("nats publish/consume payload mismatch")
			}
			return nil
		case strings.HasPrefix(line, "-ERR"):
			return fmt.Errorf("nats error: %s", line)
		}
	}
}

func (c *NATSClient) ensureConnectedLocked(ctx context.Context) error {
	if c.conn != nil {
		return nil
	}
	return c.connectLocked(ctx)
}

func (c *NATSClient) connectLocked(ctx context.Context) error {
	c.closeLocked()
	conn, reader, err := c.dial(ctx)
	if err != nil {
		return err
	}
	c.conn = conn
	c.reader = reader
	return nil
}

func (c *NATSClient) dial(ctx context.Context) (net.Conn, *bufio.Reader, error) {
	dialer := net.Dialer{}
	conn, err := dialer.DialContext(ctx, "tcp", c.addr)
	if err != nil {
		return nil, nil, fmt.Errorf("dial nats: %w", err)
	}

	if err := c.setConnDeadline(ctx, conn); err != nil {
		conn.Close()
		return nil, nil, err
	}

	reader := bufio.NewReader(conn)
	line, err := reader.ReadString('\n')
	if err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("read nats info: %w", err)
	}
	if !strings.HasPrefix(strings.TrimSpace(line), "INFO") {
		conn.Close()
		return nil, nil, fmt.Errorf("unexpected nats greeting %q", strings.TrimSpace(line))
	}

	if _, err := fmt.Fprint(conn, "CONNECT {\"verbose\":false,\"pedantic\":false}\r\nPING\r\n"); err != nil {
		conn.Close()
		return nil, nil, fmt.Errorf("write nats connect: %w", err)
	}

	if err := readPONG(reader); err != nil {
		conn.Close()
		return nil, nil, err
	}
	_ = conn.SetDeadline(noDeadline())

	return conn, reader, nil
}

func (c *NATSClient) closeLocked() {
	if c.conn != nil {
		_ = c.conn.Close()
		c.conn = nil
		c.reader = nil
	}
}

func (c *NATSClient) setDeadline(ctx context.Context) error {
	return c.setConnDeadline(ctx, c.conn)
}

func (c *NATSClient) readPONGLocked() error {
	return readPONG(c.reader)
}

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
		case line == "+OK" || strings.HasPrefix(line, "INFO"):
			continue
		case strings.HasPrefix(line, "-ERR"):
			return fmt.Errorf("nats error: %s", line)
		default:
			continue
		}
	}
}

func parseNATSAddr(raw string) (string, error) {
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
	return "", fmt.Errorf("nats url must include host and port")
}

func (c *NATSClient) setConnDeadline(ctx context.Context, conn net.Conn) error {
	if conn == nil {
		return fmt.Errorf("nats connection is not available")
	}

	deadline, ok := ctx.Deadline()
	if !ok {
		timeout := c.timeout
		if timeout <= 0 {
			timeout = 2 * time.Second
		}
		deadline = time.Now().Add(timeout)
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return fmt.Errorf("set nats deadline: %w", err)
	}
	return nil
}

func noDeadline() time.Time {
	return time.Time{}
}

func validateSubject(subject string) error {
	if strings.TrimSpace(subject) == "" {
		return fmt.Errorf("nats subject is required")
	}
	if strings.ContainsAny(subject, "\r\n\t ") {
		return fmt.Errorf("nats subject contains invalid whitespace")
	}
	return nil
}
