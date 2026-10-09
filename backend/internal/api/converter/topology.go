package converter

import (
	topologyv1 "github.com/netquest/netquest/backend/internal/contract/topology/v1"
	"github.com/netquest/netquest/backend/internal/model"
	"github.com/netquest/netquest/backend/internal/service/input"
)

// ToCreateTopologyInput переводит запрос сохранения версии во вход сервиса.
func ToCreateTopologyInput(req topologyv1.CreateRequest) input.CreateTopologyInput {
	return input.CreateTopologyInput{
		Name: req.Name,
		Data: req.Data,
	}
}

// TopologyToDTO переводит версию топологии в DTO ответа.
func TopologyToDTO(topology model.Topology) topologyv1.Topology {
	return topologyv1.Topology{
		ID:        topology.ID,
		ProjectID: topology.ProjectID,
		Version:   topology.Version,
		Name:      topology.Name,
		Data:      topology.Data,
		CreatedAt: topology.CreatedAt,
		UpdatedAt: topology.UpdatedAt,
		DeletedAt: topology.DeletedAt,
		CreatedBy: topology.CreatedBy,
	}
}

// TopologiesToDTO переводит список версий; пустой список — [], не null.
func TopologiesToDTO(topologies []model.Topology) []topologyv1.Topology {
	result := make([]topologyv1.Topology, 0, len(topologies))
	for _, topology := range topologies {
		result = append(result, TopologyToDTO(topology))
	}

	return result
}
