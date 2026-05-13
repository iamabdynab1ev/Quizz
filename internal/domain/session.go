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

type RegisterParams struct {
	Email          string  `json:"email"`
	Password       string  `json:"password"`
	FirstName      string  `json:"first_name"`
	LastName       string  `json:"last_name"`
	Patronymic     string  `json:"patronymic,omitempty"`
	Phone          *string `json:"phone,omitempty"`
	IsMale         *bool   `json:"is_male,omitempty"`
	Address        *string `json:"address,omitempty"`
	City           *string `json:"city,omitempty"`
	AvatarURL      *string `json:"avatar_url,omitempty"`
	BirthDate      *string `json:"birth_date,omitempty"`
	RecaptchaToken string  `json:"recaptcha_token,omitempty"`
	IPAddress      *string `json:"-"`
	UserAgent      *string `json:"-"`
}

type UpdateProfileParams struct {
	UserID       string  `json:"-"`
	SessionToken string  `json:"-"`
	Email        *string `json:"email,omitempty"`
	Password     *string `json:"password,omitempty"`
	FirstName    string  `json:"first_name"`
	LastName     string  `json:"last_name"`
	Patronymic   string  `json:"patronymic,omitempty"`
	Phone        *string `json:"phone,omitempty"`
	IsMale       *bool   `json:"is_male,omitempty"`
	Address      *string `json:"address,omitempty"`
	City         *string `json:"city,omitempty"`
	AvatarURL    *string `json:"avatar_url,omitempty"`
	BirthDate    *string `json:"birth_date,omitempty"`
}

type ChangePasswordParams struct {
	UserID          string  `json:"-"`
	SessionToken    string  `json:"-"`
	CurrentPassword *string `json:"current_password,omitempty"`
	NewPassword     string  `json:"new_password"`
}

type ForgotPasswordParams struct {
	Email string `json:"email"`
}

type ForgotPasswordResult struct {
	ResetToken *string    `json:"reset_token,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}

type ResetPasswordParams struct {
	Token       string `json:"token"`
	NewPassword string `json:"new_password"`
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
