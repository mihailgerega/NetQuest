package projects

import "time"

const (
	VisibilityPrivate  = "private"
	VisibilityPublic   = "public"
	VisibilityUnlisted = "unlisted"
)

type Project struct {
	ID          string     `json:"id"`
	OwnerID     string     `json:"ownerId"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	Visibility  string     `json:"visibility"`
	CreatedAt   time.Time  `json:"createdAt"`
	UpdatedAt   time.Time  `json:"updatedAt"`
	DeletedAt   *time.Time `json:"deletedAt,omitempty"`
}

type CreateRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Visibility  string `json:"visibility"`
}

type UpdateRequest struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
	Visibility  *string `json:"visibility"`
}
