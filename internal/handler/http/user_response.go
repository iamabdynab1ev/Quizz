package http

import (
	"time"

	"lms-arvand-backend/internal/domain"
)

type userResponse struct {
	ID           string     `json:"id"`
	Email        *string    `json:"email,omitempty"`
	GoogleID     *string    `json:"google_id,omitempty"`
	HasPassword  bool       `json:"has_password"`
	IsAdmin      bool       `json:"is_admin"`
	IsSuperAdmin *bool      `json:"is_super_admin,omitempty"`
	FirstName    string     `json:"first_name"`
	LastName     string     `json:"last_name"`
	Patronymic   string     `json:"patronymic,omitempty"`
	Phone        *string    `json:"phone,omitempty"`
	IsMale       *bool      `json:"is_male,omitempty"`
	Address      *string    `json:"address,omitempty"`
	City         *string    `json:"city,omitempty"`
	AvatarURL    *string    `json:"avatar_url,omitempty"`
	BirthDate    *string    `json:"birth_date,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

type loginResponse struct {
	Token     string       `json:"token"`
	ExpiresAt *time.Time   `json:"expires_at,omitempty"`
	User      userResponse `json:"user"`
}

func toLoginResponse(result domain.LoginResult) loginResponse {
	return loginResponse{
		Token:     result.Token,
		ExpiresAt: result.ExpiresAt,
		User:      toUserResponse(result.User),
	}
}

func toUserResponse(user domain.User) userResponse {
	return userResponse{
		ID:           user.ID,
		Email:        user.Email,
		GoogleID:     user.GoogleID,
		IsAdmin:      user.IsAdmin,
		IsSuperAdmin: superAdminResponseValue(user.IsSuperAdmin),
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Patronymic:   user.Patronymic,
		Phone:        user.Phone,
		HasPassword:  user.PasswordHash != nil,
		IsMale:       user.IsMale,
		Address:      user.Address,
		City:         user.City,
		AvatarURL:    user.AvatarURL,
		BirthDate:    user.BirthDate,
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
	}
}

func superAdminResponseValue(isSuperAdmin bool) *bool {
	if !isSuperAdmin {
		return nil
	}

	value := true
	return &value
}

func toUserResponses(users []domain.User) []userResponse {
	responses := make([]userResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, toUserResponse(user))
	}
	return responses
}
