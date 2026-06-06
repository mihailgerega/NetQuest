package audit

import (
	"context"
	"fmt"
	"time"

	"github.com/netquest/netquest/backend/pkg/idgen"
)

type Store interface {
	Insert(ctx context.Context, entry Entry) error
}

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Record(ctx context.Context, entry Entry) error {
	if s == nil || s.store == nil {
		return nil
	}
	if entry.Action == "" {
		return fmt.Errorf("audit action is required")
	}
	if entry.ID == "" {
		id, err := idgen.NewUUID()
		if err != nil {
			return err
		}
		entry.ID = id
	}
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = time.Now().UTC()
	}
	return s.store.Insert(ctx, entry)
}
