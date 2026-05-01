package domain

import "time"

type UserRole string

const (
	UserRoleAdmin    UserRole = "admin"
	UserRoleEmployee UserRole = "employee"
	UserRoleStudent  UserRole = "student"
	UserRoleGuest    UserRole = "guest"
)

func (r UserRole) IsValid() bool {
	switch r {
	case UserRoleAdmin, UserRoleEmployee, UserRoleStudent, UserRoleGuest:
		return true
	default:
		return false
	}
}

type Gender string

const (
	GenderMale        Gender = "male"
	GenderFemale      Gender = "female"
	GenderOther       Gender = "other"
	GenderUnspecified Gender = "unspecified"
)

func (g Gender) IsValid() bool {
	switch g {
	case GenderMale, GenderFemale, GenderOther, GenderUnspecified:
		return true
	default:
		return false
	}
}

type User struct {
	ID           string        `json:"id"`
	Username     string        `json:"username"`
	Email        *string       `json:"email,omitempty"`
	GoogleID     *string       `json:"google_id,omitempty"`
	Role         UserRole      `json:"role"`
	FirstName    string        `json:"first_name"`
	LastName     string        `json:"last_name"`
	Patronymic   string        `json:"patronymic,omitempty"`
	Phone        *string       `json:"phone,omitempty"`
	Gender       Gender        `json:"gender"`
	Address      *string       `json:"address,omitempty"`
	City         *string       `json:"city,omitempty"`
	AvatarURL    *string       `json:"avatar_url,omitempty"`
	IsActive     bool          `json:"is_active"`
	CreatedAt    time.Time     `json:"created_at"`
	UpdatedAt    time.Time     `json:"updated_at"`
	EmployeeInfo *EmployeeInfo `json:"employee_info,omitempty"`
	AdminInfo    *AdminInfo    `json:"admin_info,omitempty"`
	StudentInfo  *StudentInfo  `json:"student_info,omitempty"`
	GuestInfo    *GuestInfo    `json:"guest_info,omitempty"`
	PasswordHash *string       `json:"-"`
}

type EmployeeInfo struct {
	Branch     string `json:"branch,omitempty"`
	Office     string `json:"office,omitempty"`
	Position   string `json:"position,omitempty"`
	Department string `json:"department,omitempty"`
	EmployeeID string `json:"employee_id,omitempty"`
	HireDate   string `json:"hire_date,omitempty"`
	Notes      string `json:"notes,omitempty"`
}

type AdminInfo struct {
	IsSuperAdmin bool       `json:"is_super_admin"`
	Permissions  []string   `json:"permissions,omitempty"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
}

type StudentInfo struct {
	StudentID      string `json:"student_id,omitempty"`
	GroupName      string `json:"group_name,omitempty"`
	EducationLevel string `json:"education_level,omitempty"`
	BirthDate      string `json:"birth_date,omitempty"`
}

type GuestInfo struct {
	Source    string     `json:"source,omitempty"`
	InvitedBy *string    `json:"invited_by,omitempty"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type CreateUserParams struct {
	Username     string        `json:"username"`
	Email        *string       `json:"email,omitempty"`
	GoogleID     *string       `json:"google_id,omitempty"`
	Password     *string       `json:"password,omitempty"`
	PasswordHash *string       `json:"-"`
	Role         UserRole      `json:"role"`
	FirstName    string        `json:"first_name"`
	LastName     string        `json:"last_name"`
	Patronymic   string        `json:"patronymic,omitempty"`
	Phone        *string       `json:"phone,omitempty"`
	Gender       Gender        `json:"gender"`
	Address      *string       `json:"address,omitempty"`
	City         *string       `json:"city,omitempty"`
	AvatarURL    *string       `json:"avatar_url,omitempty"`
	EmployeeInfo *EmployeeInfo `json:"employee_info,omitempty"`
	AdminInfo    *AdminInfo    `json:"admin_info,omitempty"`
	StudentInfo  *StudentInfo  `json:"student_info,omitempty"`
	GuestInfo    *GuestInfo    `json:"guest_info,omitempty"`
}

type UpdateUserParams struct {
	ID           string        `json:"id"`
	Username     string        `json:"username"`
	Email        *string       `json:"email,omitempty"`
	GoogleID     *string       `json:"google_id,omitempty"`
	Password     *string       `json:"password,omitempty"`
	PasswordHash *string       `json:"-"`
	Role         UserRole      `json:"role"`
	FirstName    string        `json:"first_name"`
	LastName     string        `json:"last_name"`
	Patronymic   string        `json:"patronymic,omitempty"`
	Phone        *string       `json:"phone,omitempty"`
	Gender       Gender        `json:"gender"`
	Address      *string       `json:"address,omitempty"`
	City         *string       `json:"city,omitempty"`
	AvatarURL    *string       `json:"avatar_url,omitempty"`
	IsActive     bool          `json:"is_active"`
	EmployeeInfo *EmployeeInfo `json:"employee_info,omitempty"`
	AdminInfo    *AdminInfo    `json:"admin_info,omitempty"`
	StudentInfo  *StudentInfo  `json:"student_info,omitempty"`
	GuestInfo    *GuestInfo    `json:"guest_info,omitempty"`
}

type UserListFilter struct {
	Search   string
	Role     *UserRole
	IsActive *bool
	Limit    int
	Offset   int
}
