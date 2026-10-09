package record

import "time"

// Project — строка таблицы projects. description может быть NULL → "".
type Project struct {
	ID          string     `db:"id"`
	OwnerID     string     `db:"owner_id"`
	Name        string     `db:"name"`
	Description string     `db:"description"`
	Visibility  string     `db:"visibility"`
	CreatedAt   time.Time  `db:"created_at"`
	UpdatedAt   time.Time  `db:"updated_at"`
	DeletedAt   *time.Time `db:"deleted_at"` // NULL → nil
}

// Topology — строка таблицы topologies. data (jsonb) читается текстом
// (data::text): документ уходит клиенту без разбора и перекодирования.
type Topology struct {
	ID        string     `db:"id"`
	ProjectID string     `db:"project_id"`
	Version   int        `db:"version"`
	Name      string     `db:"name"`
	Data      string     `db:"data"`
	CreatedAt time.Time  `db:"created_at"`
	UpdatedAt time.Time  `db:"updated_at"`
	DeletedAt *time.Time `db:"deleted_at"` // NULL → nil
	CreatedBy *string    `db:"created_by"` // NULL → nil
}
