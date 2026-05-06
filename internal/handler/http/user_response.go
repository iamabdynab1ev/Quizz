package http

import (
	"time"

	"lms-arvand-backend/internal/domain"
)

type userResponse struct {
	ID           string               `json:"id"`
	Email        *string              `json:"email,omitempty"`
	GoogleID     *string              `json:"google_id,omitempty"`
	IsAdmin      bool                 `json:"is_admin"`
	FirstName    string               `json:"first_name"`
	LastName     string               `json:"last_name"`
	Patronymic   string               `json:"patronymic,omitempty"`
	Phone        *string              `json:"phone,omitempty"`
	IsMale       bool                 `json:"is_male"`
	Address      *string              `json:"address,omitempty"`
	City         *string              `json:"city,omitempty"`
	AvatarURL    *string              `json:"avatar_url,omitempty"`
	BirthDate    *string              `json:"birth_date,omitempty"`
	CreatedAt    time.Time            `json:"created_at"`
	UpdatedAt    time.Time            `json:"updated_at"`
	EmployeeInfo *domain.EmployeeInfo `json:"employee_info,omitempty"`
	StudentInfo  *domain.StudentInfo  `json:"student_info,omitempty"`
	GuestInfo    *domain.GuestInfo    `json:"guest_info,omitempty"`
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
		IsAdmin:      user.Role == domain.UserRoleAdmin,
		FirstName:    user.FirstName,
		LastName:     user.LastName,
		Patronymic:   user.Patronymic,
		Phone:        user.Phone,
		IsMale:       user.Gender == domain.GenderMale,
		Address:      user.Address,
		City:         user.City,
		AvatarURL:    user.AvatarURL,
		BirthDate:    userBirthDate(user),
		CreatedAt:    user.CreatedAt,
		UpdatedAt:    user.UpdatedAt,
		EmployeeInfo: user.EmployeeInfo,
		StudentInfo:  user.StudentInfo,
		GuestInfo:    user.GuestInfo,
	}
}

func toUserResponses(users []domain.User) []userResponse {
	responses := make([]userResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, toUserResponse(user))
	}
	return responses
}

func userBirthDate(user domain.User) *string {
	if user.BirthDate != nil && *user.BirthDate != "" {
		return user.BirthDate
	}

	if user.StudentInfo != nil && user.StudentInfo.BirthDate != "" {
		return &user.StudentInfo.BirthDate
	}

	return nil
}
