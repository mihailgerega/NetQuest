package converter

import (
	projectv1 "github.com/netquest/netquest/backend/internal/contract/project/v1"
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// ToCreateProjectInput переводит запрос создания проекта во вход сервиса.
func ToCreateProjectInput(req projectv1.CreateRequest) input.CreateProjectInput {
	return input.CreateProjectInput{
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
	}
}

// ToUpdateProjectInput переводит PATCH-запрос во вход сервиса; nil-поля остаются nil.
func ToUpdateProjectInput(req projectv1.UpdateRequest) input.UpdateProjectInput {
	return input.UpdateProjectInput{
		Name:        req.Name,
		Description: req.Description,
		Visibility:  req.Visibility,
	}
}

// ProjectToDTO переводит проект в DTO ответа.
func ProjectToDTO(project model.Project) projectv1.Project {
	return projectv1.Project{
		ID:          project.ID,
		OwnerID:     project.OwnerID,
		Name:        project.Name,
		Description: project.Description,
		Visibility:  project.Visibility,
		CreatedAt:   project.CreatedAt,
		UpdatedAt:   project.UpdatedAt,
		DeletedAt:   project.DeletedAt,
	}
}

// ProjectsToDTO переводит список проектов; пустой список — [], не null.
func ProjectsToDTO(projects []model.Project) []projectv1.Project {
	result := make([]projectv1.Project, 0, len(projects))
	for _, project := range projects {
		result = append(result, ProjectToDTO(project))
	}

	return result
}
