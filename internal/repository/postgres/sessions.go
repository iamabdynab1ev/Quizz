package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"lms-arvand-backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type sessionRowScanner interface {
	Scan(dest ...any) error
}

type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

func (r *SessionRepository) Create(ctx context.Context, params domain.CreateSessionParams) (domain.Session, error) {
	session, err := scanSessionRow(r.pool.QueryRow(ctx, `
		INSERT INTO sessions (
			token,
			user_id,
			ip_address,
			user_agent,
			expires_at
		) VALUES (
			$1, $2, $3, $4, $5
		)
		RETURNING token, user_id, ip_address, user_agent, created_at, expires_at
	`,
		params.Token,
		params.UserID,
		nullableStringPointerForWrite(params.IPAddress),
		nullableStringPointerForWrite(params.UserAgent),
		nullableTimePointerForWrite(params.ExpiresAt),
	))
	if err != nil {
		return domain.Session{}, wrapPGError("repository postgres sessions create", err)
	}

	return session, nil
}

func (r *SessionRepository) GetByToken(ctx context.Context, token string) (domain.Session, error) {
	session, err := scanSessionRow(r.pool.QueryRow(ctx, `
		SELECT token, user_id, ip_address, user_agent, created_at, expires_at
		FROM sessions
		WHERE token = $1
	`, token))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Session{}, fmt.Errorf("repository postgres sessions get by token: %w", domain.ErrNotFound)
		}

		return domain.Session{}, fmt.Errorf("repository postgres sessions get by token: %w", err)
	}

	return session, nil
}

func (r *SessionRepository) GetByTokenWithUser(ctx context.Context, token string) (domain.AuthIdentity, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT
			s.token,
			s.user_id,
			s.ip_address,
			s.user_agent,
			s.created_at,
			s.expires_at,
			`+userSelectColumns+`
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		LEFT JOIN user_employee_info emp ON emp.user_id = u.id
		LEFT JOIN user_student_info stu ON stu.user_id = u.id
		LEFT JOIN user_guest_info gst ON gst.user_id = u.id
		WHERE s.token = $1
	`, token)

	identity, err := scanAuthIdentityRow(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AuthIdentity{}, fmt.Errorf("repository postgres sessions get by token with user: %w", domain.ErrNotFound)
		}

		return domain.AuthIdentity{}, fmt.Errorf("repository postgres sessions get by token with user: %w", err)
	}

	return identity, nil
}

func (r *SessionRepository) DeleteByToken(ctx context.Context, token string) error {
	commandTag, err := r.pool.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	if err != nil {
		return fmt.Errorf("repository postgres sessions delete by token: %w", err)
	}

	if commandTag.RowsAffected() == 0 {
		return fmt.Errorf("repository postgres sessions delete by token: %w", domain.ErrNotFound)
	}

	return nil
}

func scanSessionRow(scanner sessionRowScanner) (domain.Session, error) {
	var session domain.Session
	var ipAddress sql.NullString
	var userAgent sql.NullString
	var expiresAt sql.NullTime

	if err := scanner.Scan(
		&session.Token,
		&session.UserID,
		&ipAddress,
		&userAgent,
		&session.CreatedAt,
		&expiresAt,
	); err != nil {
		return domain.Session{}, err
	}

	session.IPAddress = optionalString(ipAddress)
	session.UserAgent = optionalString(userAgent)
	session.ExpiresAt = optionalTime(expiresAt)

	return session, nil
}

func scanAuthIdentityRow(scanner sessionRowScanner) (domain.AuthIdentity, error) {
	var session domain.Session
	var ipAddress sql.NullString
	var userAgent sql.NullString
	var expiresAt sql.NullTime
	var user domain.User
	var email sql.NullString
	var googleID sql.NullString
	var passwordHash sql.NullString
	var firstName sql.NullString
	var lastName sql.NullString
	var patronymic sql.NullString
	var phone sql.NullString
	var birthDate sql.NullTime
	var address sql.NullString
	var city sql.NullString
	var avatarURL sql.NullString
	var role string
	var gender string
	var hasEmployeeInfo bool
	var employeeBranch sql.NullString
	var employeeOffice sql.NullString
	var employeePosition sql.NullString
	var employeeDepartment sql.NullString
	var employeeID sql.NullString
	var employeeHireDate sql.NullTime
	var employeeNotes sql.NullString
	var hasStudentInfo bool
	var studentID sql.NullString
	var studentGroupName sql.NullString
	var studentEducationLevel sql.NullString
	var studentBirthDate sql.NullTime
	var hasGuestInfo bool
	var guestSource sql.NullString
	var guestInvitedBy sql.NullString
	var guestExpiresAt sql.NullTime

	if err := scanner.Scan(
		&session.Token,
		&session.UserID,
		&ipAddress,
		&userAgent,
		&session.CreatedAt,
		&expiresAt,
		&user.ID,
		&email,
		&googleID,
		&passwordHash,
		&role,
		&firstName,
		&lastName,
		&patronymic,
		&phone,
		&gender,
		&birthDate,
		&address,
		&city,
		&avatarURL,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
		&hasEmployeeInfo,
		&employeeBranch,
		&employeeOffice,
		&employeePosition,
		&employeeDepartment,
		&employeeID,
		&employeeHireDate,
		&employeeNotes,
		&hasStudentInfo,
		&studentID,
		&studentGroupName,
		&studentEducationLevel,
		&studentBirthDate,
		&hasGuestInfo,
		&guestSource,
		&guestInvitedBy,
		&guestExpiresAt,
	); err != nil {
		return domain.AuthIdentity{}, err
	}

	user.Email = optionalString(email)
	user.GoogleID = optionalString(googleID)
	user.PasswordHash = optionalString(passwordHash)
	user.Role = domain.UserRole(role)
	user.FirstName = firstName.String
	user.LastName = lastName.String
	user.Patronymic = patronymic.String
	user.Phone = optionalString(phone)
	user.Gender = domain.Gender(gender)
	user.BirthDate = optionalDateString(birthDate)
	user.Address = optionalString(address)
	user.City = optionalString(city)
	user.AvatarURL = optionalString(avatarURL)

	if hasEmployeeInfo {
		user.EmployeeInfo = &domain.EmployeeInfo{
			Branch:     employeeBranch.String,
			Office:     employeeOffice.String,
			Position:   employeePosition.String,
			Department: employeeDepartment.String,
			EmployeeID: employeeID.String,
			HireDate:   dateString(employeeHireDate),
			Notes:      employeeNotes.String,
		}
	}

	if hasStudentInfo {
		user.StudentInfo = &domain.StudentInfo{
			StudentID:      studentID.String,
			GroupName:      studentGroupName.String,
			EducationLevel: studentEducationLevel.String,
			BirthDate:      dateString(studentBirthDate),
		}
	}

	if hasGuestInfo {
		user.GuestInfo = &domain.GuestInfo{
			Source:    guestSource.String,
			InvitedBy: optionalString(guestInvitedBy),
			ExpiresAt: optionalTime(guestExpiresAt),
		}
	}

	session.IPAddress = optionalString(ipAddress)
	session.UserAgent = optionalString(userAgent)
	session.ExpiresAt = optionalTime(expiresAt)

	return domain.AuthIdentity{
		User:    user,
		Session: session,
	}, nil
}
