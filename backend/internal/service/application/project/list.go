package project

import (
	"context"
	"fmt"

	"github.com/netquest/netquest/backend/internal/model"
)

// List возвращает проекты владельца.
func (s *service) List(ctx context.Context, ownerID string) ([]model.Project, error) {
	projects, err := s.projectRepository.List(ctx, ownerID)
	if err != nil {
		return nil, fmt.Errorf("получить проекты: %w", err)
	}

	return projects, nil
}
