package users

import "time"

type User struct {
	ID           string     `json:"id"`
	Email        string     `json:"email"`
	DisplayName  string     `json:"displayName"`
	AvatarURL    string     `json:"avatarUrl,omitempty"`
	Role         string     `json:"role"`
	CreatedAt    time.Time  `json:"createdAt"`
	UpdatedAt    time.Time  `json:"updatedAt"`
	DeletedAt    *time.Time `json:"deletedAt,omitempty"`
	PasswordHash string     `json:"-"`
}

type CreateUser struct {
	ID           string
	Email        string
	PasswordHash string
	DisplayName  string
	Role         string
}
