package converter

import (
	"encoding/json"

	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/repository/record"
)

// ProjectToModel переводит строку projects в доменную модель.
func ProjectToModel(rec record.Project) model.Project {
	return model.Project{
		ID:          rec.ID,
		OwnerID:     rec.OwnerID,
		Name:        rec.Name,
		Description: rec.Description,
		Visibility:  rec.Visibility,
		CreatedAt:   rec.CreatedAt,
		UpdatedAt:   rec.UpdatedAt,
		DeletedAt:   rec.DeletedAt,
	}
}

// ProjectsToModel переводит список строк; пустой список — пустой срез, не nil.
func ProjectsToModel(recs []record.Project) []model.Project {
	projects := make([]model.Project, 0, len(recs))
	for _, rec := range recs {
		projects = append(projects, ProjectToModel(rec))
	}

	return projects
}

// TopologyToModel переводит строку topologies в доменную модель.
// Текст документа становится json.RawMessage без разбора.
func TopologyToModel(rec record.Topology) model.Topology {
	return model.Topology{
		ID:        rec.ID,
		ProjectID: rec.ProjectID,
		Version:   rec.Version,
		Name:      rec.Name,
		Data:      json.RawMessage(rec.Data),
		CreatedAt: rec.CreatedAt,
		UpdatedAt: rec.UpdatedAt,
		DeletedAt: rec.DeletedAt,
		CreatedBy: rec.CreatedBy,
	}
}

// TopologiesToModel переводит список строк; пустой список — пустой срез, не nil.
func TopologiesToModel(recs []record.Topology) []model.Topology {
	topologies := make([]model.Topology, 0, len(recs))
	for _, rec := range recs {
		topologies = append(topologies, TopologyToModel(rec))
	}

	return topologies
}
