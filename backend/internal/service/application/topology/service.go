// Package topology — сервисный слой версий топологии: список версий проекта,
// сохранение новой версии, чтение и повторная валидация.
//
// Место в цепочке: api/topology/v1 → service/application/topology →
// {service/application/project (владелец), repository/topology, domain/validator}.
//
// Версия сохраняется, только если документ прошёл валидацию: в базе не бывает
// топологий, которые движок откажется считать.
package topology

// maxNameLength — предел длины имени версии топологии (в байтах).
const maxNameLength = 120

// service — сервис версий топологии.
type service struct {
	topologyRepository TopologyRepository
	projectAuthorizer  ProjectAuthorizer
	validator          TopologyValidator
}

// New создаёт сервис версий топологии.
func New(
	topologyRepository TopologyRepository,
	projectAuthorizer ProjectAuthorizer,
	validator TopologyValidator,
) *service {
	return &service{
		topologyRepository: topologyRepository,
		projectAuthorizer:  projectAuthorizer,
		validator:          validator,
	}
}
