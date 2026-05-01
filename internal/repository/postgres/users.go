package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"lms-arvand-backend/internal/domain"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

const userSelectColumns = `
	u.id,
	u.username,
	u.email,
	u.google_id,
	u.password_hash,
	u.role,
	u.first_name,
	u.last_name,
	u.patronymic,
	u.phone,
	u.gender,
	u.address,
	u.city,
	u.avatar_url,
	u.is_active,
	u.created_at,
	u.updated_at,
	emp.user_id IS NOT NULL AS has_employee_info,
	emp.branch,
	emp.office,
	emp.position,
	emp.department,
	emp.employee_id,
	emp.hire_date,
	emp.notes,
	adm.user_id IS NOT NULL AS has_admin_info,
	adm.is_super_admin,
	adm.permissions,
	adm.last_login_at,
	stu.user_id IS NOT NULL AS has_student_info,
	stu.student_id,
	stu.group_name,
	stu.education_level,
	stu.birth_date,
	gst.user_id IS NOT NULL AS has_guest_info,
	gst.source,
	gst.invited_by,
	gst.expires_at
`

type userRowScanner interface {
	Scan(dest ...any) error
}

type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Create(ctx context.Context, params domain.CreateUserParams) (domain.User, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.User{}, fmt.Errorf("repository postgres users create begin tx: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var userID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO users (
			username,
			email,
			google_id,
			password_hash,
			role,
			first_name,
			last_name,
			patronymic,
			phone,
			gender,
			address,
			city,
			avatar_url
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)
		RETURNING id
	`,
		params.Username,
		nullableStringPointerForWrite(params.Email),
		nullableStringPointerForWrite(params.GoogleID),
		nullableStringPointerForWrite(params.PasswordHash),
		string(params.Role),
		nullableStringForWrite(params.FirstName),
		nullableStringForWrite(params.LastName),
		nullableStringForWrite(params.Patronymic),
		nullableStringPointerForWrite(params.Phone),
		string(params.Gender),
		nullableStringPointerForWrite(params.Address),
		nullableStringPointerForWrite(params.City),
		nullableStringPointerForWrite(params.AvatarURL),
	).Scan(&userID); err != nil {
		return domain.User{}, wrapPGError("repository postgres users create insert", err)
	}

	if err := r.replaceRoleDetails(ctx, tx, userID, params.Role, params.EmployeeInfo, params.AdminInfo, params.StudentInfo, params.GuestInfo); err != nil {
		return domain.User{}, fmt.Errorf("repository postgres users create role details: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, fmt.Errorf("repository postgres users create commit: %w", err)
	}

	user, err := r.GetByID(ctx, userID)
	if err != nil {
		return domain.User{}, fmt.Errorf("repository postgres users create fetch by id: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByID(ctx context.Context, userID string) (domain.User, error) {
	user, err := scanUser(
		r.pool.QueryRow(ctx, `
			SELECT `+userSelectColumns+`
			FROM users u
			LEFT JOIN user_employee_info emp ON emp.user_id = u.id
			LEFT JOIN user_admin_info adm ON adm.user_id = u.id
			LEFT JOIN user_student_info stu ON stu.user_id = u.id
			LEFT JOIN user_guest_info gst ON gst.user_id = u.id
			WHERE u.id = $1
		`, userID),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, fmt.Errorf("repository postgres users get by id: %w", domain.ErrNotFound)
		}

		return domain.User{}, fmt.Errorf("repository postgres users get by id: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByLogin(ctx context.Context, identifier string) (domain.User, error) {
	user, err := scanUser(
		r.pool.QueryRow(ctx, `
			SELECT `+userSelectColumns+`
			FROM users u
			LEFT JOIN user_employee_info emp ON emp.user_id = u.id
			LEFT JOIN user_admin_info adm ON adm.user_id = u.id
			LEFT JOIN user_student_info stu ON stu.user_id = u.id
			LEFT JOIN user_guest_info gst ON gst.user_id = u.id
			WHERE u.is_active = true AND (u.username = $1 OR u.email = $1)
			LIMIT 1
		`, identifier),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, fmt.Errorf("repository postgres users get by login: %w", domain.ErrNotFound)
		}

		return domain.User{}, fmt.Errorf("repository postgres users get by login: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	user, err := scanUser(
		r.pool.QueryRow(ctx, `
			SELECT `+userSelectColumns+`
			FROM users u
			LEFT JOIN user_employee_info emp ON emp.user_id = u.id
			LEFT JOIN user_admin_info adm ON adm.user_id = u.id
			LEFT JOIN user_student_info stu ON stu.user_id = u.id
			LEFT JOIN user_guest_info gst ON gst.user_id = u.id
			WHERE LOWER(u.email) = LOWER($1)
			LIMIT 1
		`, email),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, fmt.Errorf("repository postgres users get by email: %w", domain.ErrNotFound)
		}

		return domain.User{}, fmt.Errorf("repository postgres users get by email: %w", err)
	}

	return user, nil
}

func (r *UserRepository) GetByGoogleID(ctx context.Context, googleID string) (domain.User, error) {
	user, err := scanUser(
		r.pool.QueryRow(ctx, `
			SELECT `+userSelectColumns+`
			FROM users u
			LEFT JOIN user_employee_info emp ON emp.user_id = u.id
			LEFT JOIN user_admin_info adm ON adm.user_id = u.id
			LEFT JOIN user_student_info stu ON stu.user_id = u.id
			LEFT JOIN user_guest_info gst ON gst.user_id = u.id
			WHERE u.google_id = $1
			LIMIT 1
		`, googleID),
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, fmt.Errorf("repository postgres users get by google id: %w", domain.ErrNotFound)
		}

		return domain.User{}, fmt.Errorf("repository postgres users get by google id: %w", err)
	}

	return user, nil
}

func (r *UserRepository) List(ctx context.Context, filter domain.UserListFilter) ([]domain.User, int, error) {
	buildQuery := func(includePagination bool) (string, []any) {
		query := strings.Builder{}
		if includePagination {
			query.WriteString(`
				SELECT ` + userSelectColumns + `
				FROM users u
				LEFT JOIN user_employee_info emp ON emp.user_id = u.id
				LEFT JOIN user_admin_info adm ON adm.user_id = u.id
				LEFT JOIN user_student_info stu ON stu.user_id = u.id
				LEFT JOIN user_guest_info gst ON gst.user_id = u.id
				WHERE 1 = 1
			`)
		} else {
			query.WriteString(`
				SELECT COUNT(*)
				FROM users u
				LEFT JOIN user_employee_info emp ON emp.user_id = u.id
				LEFT JOIN user_admin_info adm ON adm.user_id = u.id
				LEFT JOIN user_student_info stu ON stu.user_id = u.id
				LEFT JOIN user_guest_info gst ON gst.user_id = u.id
				WHERE 1 = 1
			`)
		}

		args := make([]any, 0, 6)
		position := 1

		if filter.Search != "" {
			query.WriteString(fmt.Sprintf(`
				AND (
					u.username ILIKE $%d OR
					COALESCE(u.email, '') ILIKE $%d OR
					COALESCE(u.first_name, '') ILIKE $%d OR
					COALESCE(u.last_name, '') ILIKE $%d OR
					COALESCE(u.patronymic, '') ILIKE $%d
				)
			`, position, position, position, position, position))
			args = append(args, "%"+filter.Search+"%")
			position++
		}

		if filter.Role != nil {
			query.WriteString(fmt.Sprintf(" AND u.role = $%d", position))
			args = append(args, string(*filter.Role))
			position++
		}

		if filter.IsActive != nil {
			query.WriteString(fmt.Sprintf(" AND u.is_active = $%d", position))
			args = append(args, *filter.IsActive)
			position++
		}

		if includePagination {
			query.WriteString(fmt.Sprintf(" ORDER BY u.created_at DESC LIMIT $%d OFFSET $%d", position, position+1))
			args = append(args, filter.Limit, filter.Offset)
		}

		return query.String(), args
	}

	countQuery, countArgs := buildQuery(false)
	var total int
	if err := r.pool.QueryRow(ctx, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("repository postgres users list count: %w", err)
	}

	query, args := buildQuery(true)
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("repository postgres users list query: %w", err)
	}
	defer rows.Close()

	users := make([]domain.User, 0, filter.Limit)
	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			return nil, 0, fmt.Errorf("repository postgres users list scan: %w", err)
		}

		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("repository postgres users list rows: %w", err)
	}

	return users, total, nil
}

func (r *UserRepository) Update(ctx context.Context, params domain.UpdateUserParams) (domain.User, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return domain.User{}, fmt.Errorf("repository postgres users update begin tx: %w", err)
	}

	defer func() {
		_ = tx.Rollback(ctx)
	}()

	var userID string
	if err := tx.QueryRow(ctx, `
		UPDATE users
		SET
			username = $2,
			email = $3,
			google_id = $4,
			password_hash = $5,
			role = $6,
			first_name = $7,
			last_name = $8,
			patronymic = $9,
			phone = $10,
			gender = $11,
			address = $12,
			city = $13,
			avatar_url = $14,
			is_active = $15,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`,
		params.ID,
		params.Username,
		nullableStringPointerForWrite(params.Email),
		nullableStringPointerForWrite(params.GoogleID),
		nullableStringPointerForWrite(params.PasswordHash),
		string(params.Role),
		nullableStringForWrite(params.FirstName),
		nullableStringForWrite(params.LastName),
		nullableStringForWrite(params.Patronymic),
		nullableStringPointerForWrite(params.Phone),
		string(params.Gender),
		nullableStringPointerForWrite(params.Address),
		nullableStringPointerForWrite(params.City),
		nullableStringPointerForWrite(params.AvatarURL),
		params.IsActive,
	).Scan(&userID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, fmt.Errorf("repository postgres users update: %w", domain.ErrNotFound)
		}

		return domain.User{}, wrapPGError("repository postgres users update", err)
	}

	if err := r.replaceRoleDetails(ctx, tx, userID, params.Role, params.EmployeeInfo, params.AdminInfo, params.StudentInfo, params.GuestInfo); err != nil {
		return domain.User{}, fmt.Errorf("repository postgres users update role details: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.User{}, fmt.Errorf("repository postgres users update commit: %w", err)
	}

	user, err := r.GetByID(ctx, userID)
	if err != nil {
		return domain.User{}, fmt.Errorf("repository postgres users update fetch by id: %w", err)
	}

	return user, nil
}

func (r *UserRepository) Deactivate(ctx context.Context, userID string) error {
	var returnedID string
	if err := r.pool.QueryRow(ctx, `
		UPDATE users
		SET
			is_active = false,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, userID).Scan(&returnedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fmt.Errorf("repository postgres users deactivate: %w", domain.ErrNotFound)
		}

		return fmt.Errorf("repository postgres users deactivate: %w", err)
	}

	return nil
}

func (r *UserRepository) LinkGoogleID(ctx context.Context, userID, googleID string) (domain.User, error) {
	var returnedID string
	if err := r.pool.QueryRow(ctx, `
		UPDATE users
		SET
			google_id = $2,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`, userID, googleID).Scan(&returnedID); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.User{}, fmt.Errorf("repository postgres users link google id: %w", domain.ErrNotFound)
		}

		return domain.User{}, wrapPGError("repository postgres users link google id", err)
	}

	user, err := r.GetByID(ctx, returnedID)
	if err != nil {
		return domain.User{}, fmt.Errorf("repository postgres users link google id fetch by id: %w", err)
	}

	return user, nil
}

func (r *UserRepository) replaceRoleDetails(
	ctx context.Context,
	tx pgx.Tx,
	userID string,
	role domain.UserRole,
	employeeInfo *domain.EmployeeInfo,
	adminInfo *domain.AdminInfo,
	studentInfo *domain.StudentInfo,
	guestInfo *domain.GuestInfo,
) error {
	if err := r.clearRoleDetails(ctx, tx, userID); err != nil {
		return fmt.Errorf("repository postgres users replace role details clear: %w", err)
	}

	switch role {
	case domain.UserRoleEmployee:
		if employeeInfo == nil {
			employeeInfo = &domain.EmployeeInfo{}
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO user_employee_info (
				user_id,
				branch,
				office,
				position,
				department,
				employee_id,
				hire_date,
				notes
			) VALUES (
				$1, $2, $3, $4, $5, $6, NULLIF($7, '')::date, $8
			)
		`,
			userID,
			nullableStringForWrite(employeeInfo.Branch),
			nullableStringForWrite(employeeInfo.Office),
			nullableStringForWrite(employeeInfo.Position),
			nullableStringForWrite(employeeInfo.Department),
			nullableStringForWrite(employeeInfo.EmployeeID),
			employeeInfo.HireDate,
			nullableStringForWrite(employeeInfo.Notes),
		); err != nil {
			return wrapPGError("repository postgres users insert employee info", err)
		}
	case domain.UserRoleAdmin:
		if adminInfo == nil {
			adminInfo = &domain.AdminInfo{}
		}

		permissions, err := permissionsValue(adminInfo.Permissions)
		if err != nil {
			return fmt.Errorf("repository postgres users admin permissions value: %w", err)
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO user_admin_info (
				user_id,
				is_super_admin,
				permissions,
				last_login_at
			) VALUES (
				$1, $2, $3::jsonb, $4
			)
		`,
			userID,
			adminInfo.IsSuperAdmin,
			permissions,
			nullableTimePointerForWrite(adminInfo.LastLoginAt),
		); err != nil {
			return wrapPGError("repository postgres users insert admin info", err)
		}
	case domain.UserRoleStudent:
		if studentInfo == nil {
			studentInfo = &domain.StudentInfo{}
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO user_student_info (
				user_id,
				student_id,
				group_name,
				education_level,
				birth_date
			) VALUES (
				$1, $2, $3, $4, NULLIF($5, '')::date
			)
		`,
			userID,
			nullableStringForWrite(studentInfo.StudentID),
			nullableStringForWrite(studentInfo.GroupName),
			nullableStringForWrite(studentInfo.EducationLevel),
			studentInfo.BirthDate,
		); err != nil {
			return wrapPGError("repository postgres users insert student info", err)
		}
	case domain.UserRoleGuest:
		if guestInfo == nil {
			guestInfo = &domain.GuestInfo{}
		}

		if _, err := tx.Exec(ctx, `
			INSERT INTO user_guest_info (
				user_id,
				source,
				invited_by,
				expires_at
			) VALUES (
				$1, $2, $3::uuid, $4
			)
		`,
			userID,
			nullableStringForWrite(guestInfo.Source),
			nullableStringPointerForWrite(guestInfo.InvitedBy),
			nullableTimePointerForWrite(guestInfo.ExpiresAt),
		); err != nil {
			return wrapPGError("repository postgres users insert guest info", err)
		}
	}

	return nil
}

func (r *UserRepository) clearRoleDetails(ctx context.Context, tx pgx.Tx, userID string) error {
	if _, err := tx.Exec(ctx, `DELETE FROM user_employee_info WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("repository postgres users clear employee info: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_admin_info WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("repository postgres users clear admin info: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_student_info WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("repository postgres users clear student info: %w", err)
	}

	if _, err := tx.Exec(ctx, `DELETE FROM user_guest_info WHERE user_id = $1`, userID); err != nil {
		return fmt.Errorf("repository postgres users clear guest info: %w", err)
	}

	return nil
}

func scanUser(scanner userRowScanner) (domain.User, error) {
	var user domain.User
	var email sql.NullString
	var googleID sql.NullString
	var passwordHash sql.NullString
	var firstName sql.NullString
	var lastName sql.NullString
	var patronymic sql.NullString
	var phone sql.NullString
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
	var hasAdminInfo bool
	var adminIsSuperAdmin sql.NullBool
	var adminPermissions []byte
	var adminLastLogin sql.NullTime
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
		&user.ID,
		&user.Username,
		&email,
		&googleID,
		&passwordHash,
		&role,
		&firstName,
		&lastName,
		&patronymic,
		&phone,
		&gender,
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
		&hasAdminInfo,
		&adminIsSuperAdmin,
		&adminPermissions,
		&adminLastLogin,
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
		return domain.User{}, err
	}

	permissions, err := parsePermissions(adminPermissions)
	if err != nil {
		return domain.User{}, fmt.Errorf("repository postgres scan user permissions: %w", err)
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

	if hasAdminInfo {
		user.AdminInfo = &domain.AdminInfo{
			IsSuperAdmin: adminIsSuperAdmin.Bool,
			Permissions:  permissions,
			LastLoginAt:  optionalTime(adminLastLogin),
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

	return user, nil
}
