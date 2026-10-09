// Package simulation — репозиторий симуляций и их событий поверх PostgreSQL (pgx/v5).
//
// Место в цепочке: service/application/simulation → repository/simulation →
// таблицы simulations и simulation_events (+ projects для проверки владельца).
//
// Один репозиторий на агрегат: симуляция и её события пишутся и читаются
// вместе, поэтому здесь обе таблицы.
package simulation

import (
	"github.com/jackc/pgx/v5/pgxpool"
)

// simulationColumns — колонки в порядке и под именами полей record.Simulation.
const simulationColumns = `id::text AS id, project_id::text AS project_id, topology_id::text AS topology_id,
	user_id::text AS user_id, status, scenario::text AS scenario, seed, started_at, finished_at,
	error_message, created_at, updated_at`

// repository читает и пишет симуляции через общий пул соединений.
type repository struct {
	pool *pgxpool.Pool
}

// New создаёт репозиторий симуляций поверх пула.
func New(pool *pgxpool.Pool) *repository {
	return &repository{
		pool: pool,
	}
}
