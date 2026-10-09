package project

import (
	"context"
)

// EnsureOwner проверяет, что проект существует и принадлежит ownerID.
// Нужна сервисам топологий и симуляций до работы с данными проекта.
func (s *service) EnsureOwner(ctx context.Context, projectID, ownerID string) error {
	_, err := s.projectRepository.GetByOwner(ctx, ownerID, projectID)
	return err
}
