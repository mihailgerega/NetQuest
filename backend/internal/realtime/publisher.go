package realtime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/netquest/netquest/backend/internal/storage"
)

type EventPublisher interface {
	Publish(ctx context.Context, subject string, payload any) error
}

type NATSPublisher struct {
	conn *storage.NATSClient
}

func NewNATSPublisher(conn *storage.NATSClient) *NATSPublisher {
	return &NATSPublisher{conn: conn}
}

func (p *NATSPublisher) Publish(ctx context.Context, subject string, payload any) error {
	if p == nil || p.conn == nil {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal realtime payload: %w", err)
	}
	return p.conn.Publish(ctx, subject, data)
}
