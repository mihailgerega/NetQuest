// Package topology — репозиторий версий топологии поверх PostgreSQL (pgx/v5).
//
// Место в цепочке: service/application/{topology,simulation,advisor} →
// repository/topology → таблица topologies (+ projects для проверки владельца).
//
// Владельца у версии топологии нет — он у проекта, поэтому чтение идёт через
// JOIN с projects и условие p.owner_id. Версия удалённого проекта недоступна,
// даже если сама она не помечена удалённой.
package topology

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// topologyColumns — колонки в порядке и под именами полей record.Topology.
// Префикс t. — запросы чтения соединяют topologies (t) с projects (p).
const topologyColumns = `t.id::text AS id, t.project_id::text AS project_id, t.version, t.name,
	t.data::text AS data, t.created_at, t.updated_at, t.deleted_at, t.created_by::text AS created_by`

// repository читает и пишет версии топологии через общий пул соединений.
type repository struct {
	pool *pgxpool.Pool
}

// New создаёт репозиторий топологий поверх пула.
func New(pool *pgxpool.Pool) *repository {
	return &repository{
		pool: pool,
	}
}
