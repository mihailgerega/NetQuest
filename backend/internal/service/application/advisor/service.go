// Package advisor — сервисный слой Validation Advisor: диагностика топологии
// до запуска симуляции.
//
// Место в цепочке: api/advisor/v1 → service/application/advisor →
// {repository/topology (сохранённая версия), domain/validator}.
//
// Советник не запускает симуляцию: он смотрит на настройки узлов и каналов
// и находит типовые ошибки — нет DNS-записи, пустой пул Load Balancer'а,
// firewall закрывает HTTPS, медленный канал, нет маршрута. Каждое правило
// живёт в своём файле и возвращает список замечаний (model.Issue).
package advisor

// service — советник. Неэкспортируемый тип: снаружи с ним работают через
// интерфейс AdvisorService из deps.go API-слоя.
type service struct {
	topologyReader TopologyReader
	validator      TopologyValidator
}

// New создаёт советника.
func New(topologyReader TopologyReader, validator TopologyValidator) *service {
	return &service{
		topologyReader: topologyReader,
		validator:      validator,
	}
}
