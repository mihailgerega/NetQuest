package project

import "context"

// Delete мягко удаляет проект владельца; чужой, удалённый или несуществующий →
// errs.ErrProjectNotFound. Версии топологии и симуляции проекта остаются
// в базе, но становятся недоступны: их запросы проверяют p.deleted_at.
func (s *service) Delete(ctx context.Context, ownerID, projectID string) error {
	return s.projectRepository.SoftDelete(ctx, ownerID, projectID)
}
