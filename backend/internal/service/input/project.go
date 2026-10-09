package input

// CreateProjectInput — вход создания проекта.
type CreateProjectInput struct {
	Name        string
	Description string
	Visibility  string // пусто — private
}

// UpdateProjectInput — частичное обновление проекта (PATCH): nil — поле
// не меняется, указатель на пустую строку — поле очищается (если разрешено).
type UpdateProjectInput struct {
	Name        *string
	Description *string
	Visibility  *string
}
