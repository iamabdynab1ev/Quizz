package domain

import "time"

type User struct {
	ID                 string    `json:"id"`
	Email              *string   `json:"email,omitempty"`
	GoogleID           *string   `json:"google_id,omitempty"`
	IsAdmin            bool      `json:"is_admin"`
	IsSuperAdmin       bool      `json:"is_super_admin"`
	MustChangePassword bool      `json:"must_change_password,omitempty"`
	FirstName          string    `json:"first_name"`
	LastName           string    `json:"last_name"`
	Patronymic         string    `json:"patronymic,omitempty"`
	Phone              *string   `json:"phone,omitempty"`
	IsMale             *bool     `json:"is_male,omitempty"`
	BirthDate          *string   `json:"birth_date,omitempty"`
	Address            *string   `json:"address,omitempty"`
	City               *string   `json:"city,omitempty"`
	AvatarURL          *string   `json:"avatar_url,omitempty"`
	IsActive           bool      `json:"is_active"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
	PasswordHash       *string   `json:"-"`
}

type CreateUserParams struct {
	Email              *string `json:"email,omitempty"`
	GoogleID           *string `json:"google_id,omitempty"`
	Password           *string `json:"password,omitempty"`
	PasswordHash       *string `json:"-"`
	IsAdmin            bool    `json:"is_admin"`
	IsSuperAdmin       bool    `json:"is_super_admin"`
	MustChangePassword bool    `json:"must_change_password,omitempty"`
	FirstName          string  `json:"first_name"`
	LastName           string  `json:"last_name"`
	Patronymic         string  `json:"patronymic,omitempty"`
	Phone              *string `json:"phone,omitempty"`
	IsMale             *bool   `json:"is_male,omitempty"`
	BirthDate          *string `json:"birth_date,omitempty"`
	Address            *string `json:"address,omitempty"`
	City               *string `json:"city,omitempty"`
	AvatarURL          *string `json:"avatar_url,omitempty"`
}

type UpdateUserParams struct {
	ID                 string  `json:"id"`
	Email              *string `json:"email,omitempty"`
	GoogleID           *string `json:"google_id,omitempty"`
	Password           *string `json:"password,omitempty"`
	PasswordHash       *string `json:"-"`
	IsAdmin            bool    `json:"is_admin"`
	IsSuperAdmin       *bool   `json:"is_super_admin,omitempty"`
	MustChangePassword *bool   `json:"must_change_password,omitempty"`
	FirstName          string  `json:"first_name"`
	LastName           string  `json:"last_name"`
	Patronymic         string  `json:"patronymic,omitempty"`
	Phone              *string `json:"phone,omitempty"`
	IsMale             *bool   `json:"is_male,omitempty"`
	BirthDate          *string `json:"birth_date,omitempty"`
	Address            *string `json:"address,omitempty"`
	City               *string `json:"city,omitempty"`
	AvatarURL          *string `json:"avatar_url,omitempty"`
	IsActive           bool    `json:"is_active"`
}

type UserListFilter struct {
	Search   string
	IsAdmin  *bool
	IsActive *bool
	Limit    int
	Offset   int
}
