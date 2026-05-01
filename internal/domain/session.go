package domain

import "time"

type Session struct {
	Token     string     `json:"token,omitempty"`
	UserID    string     `json:"user_id"`
	IPAddress *string    `json:"ip_address,omitempty"`
	UserAgent *string    `json:"user_agent,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type CreateSessionParams struct {
	Token     string
	UserID    string
	IPAddress *string
	UserAgent *string
	ExpiresAt *time.Time
}

type LoginParams struct {
	Identifier string
	Password   string
	IPAddress  *string
	UserAgent  *string
}

type GoogleLoginParams struct {
	IDToken   string
	IPAddress *string
	UserAgent *string
}

type LoginResult struct {
	Token     string     `json:"token"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	User      User       `json:"user"`
}

type AuthIdentity struct {
	User    User    `json:"user"`
	Session Session `json:"session"`
}
