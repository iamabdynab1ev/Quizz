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
	u.email,
	u.google_id,
	u.password_hash,
	u.is_admin,
	u.is_super_admin,
	u.first_name,
	u.last_name,
	u.patronymic,
	u.phone,
	u.is_male,
	u.birth_date,
	u.address,
	u.city,
	u.avatar_url,
	u.is_active,
	u.created_at,
	u.updated_at
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
	var userID string
	if err := r.pool.QueryRow(ctx, `
		INSERT INTO users (
			email,
			google_id,
			password_hash,
			is_admin,
			is_super_admin,
			first_name,
			last_name,
			patronymic,
			phone,
			is_male,
			birth_date,
			address,
			city,
			avatar_url
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NULLIF($11, '')::date, $12, $13, $14
		)
		RETURNING id
	`,
		nullableStringPointerForWrite(params.Email),
		nullableStringPointerForWrite(params.GoogleID),
		nullableStringPointerForWrite(params.PasswordHash),
		params.IsAdmin,
		params.IsSuperAdmin,
		nullableStringForWrite(params.FirstName),
		nullableStringForWrite(params.LastName),
		nullableStringForWrite(params.Patronymic),
		nullableStringPointerForWrite(params.Phone),
		nullableBoolPointerForWrite(params.IsMale),
		stringPointerForWrite(params.BirthDate),
		nullableStringPointerForWrite(params.Address),
		nullableStringPointerForWrite(params.City),
		nullableStringPointerForWrite(params.AvatarURL),
	).Scan(&userID); err != nil {
		return domain.User{}, wrapPGError("repository postgres users create insert", err)
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

func (r *UserRepository) GetByEmail(ctx context.Context, email string) (domain.User, error) {
	user, err := scanUser(
		r.pool.QueryRow(ctx, `
			SELECT `+userSelectColumns+`
			FROM users u
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
			query.WriteString(`SELECT ` + userSelectColumns + ` FROM users u WHERE 1 = 1`)
		} else {
			query.WriteString(`SELECT COUNT(*) FROM users u WHERE 1 = 1`)
		}

		args := make([]any, 0, 6)
		position := 1

		if filter.Search != "" {
			query.WriteString(fmt.Sprintf(`
				AND (
					COALESCE(u.email, '') ILIKE $%d OR
					COALESCE(u.first_name, '') ILIKE $%d OR
					COALESCE(u.last_name, '') ILIKE $%d OR
					COALESCE(u.patronymic, '') ILIKE $%d
				)
			`, position, position, position, position))
			args = append(args, "%"+filter.Search+"%")
			position++
		}

		if filter.IsAdmin != nil {
			query.WriteString(fmt.Sprintf(" AND u.is_admin = $%d", position))
			args = append(args, *filter.IsAdmin)
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
	var userID string
	if err := r.pool.QueryRow(ctx, `
		UPDATE users
		SET
			email = $2,
			google_id = $3,
			password_hash = COALESCE($4, password_hash),
			is_admin = CASE WHEN is_super_admin THEN true ELSE $5 END,
			is_super_admin = CASE WHEN is_super_admin THEN true ELSE COALESCE($6, is_super_admin) END,
			first_name = $7,
			last_name = $8,
			patronymic = $9,
			phone = $10,
			is_male = $11,
			birth_date = NULLIF($12, '')::date,
			address = $13,
			city = $14,
			avatar_url = $15,
			is_active = $16,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id
	`,
		params.ID,
		nullableStringPointerForWrite(params.Email),
		nullableStringPointerForWrite(params.GoogleID),
		nullableStringPointerForWrite(params.PasswordHash),
		params.IsAdmin,
		nullableBoolPointerForWrite(params.IsSuperAdmin),
		nullableStringForWrite(params.FirstName),
		nullableStringForWrite(params.LastName),
		nullableStringForWrite(params.Patronymic),
		nullableStringPointerForWrite(params.Phone),
		nullableBoolPointerForWrite(params.IsMale),
		stringPointerForWrite(params.BirthDate),
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

func scanUser(scanner userRowScanner) (domain.User, error) {
	var user domain.User
	var email sql.NullString
	var googleID sql.NullString
	var passwordHash sql.NullString
	var firstName sql.NullString
	var lastName sql.NullString
	var patronymic sql.NullString
	var phone sql.NullString
	var isMale sql.NullBool
	var birthDate sql.NullTime
	var address sql.NullString
	var city sql.NullString
	var avatarURL sql.NullString

	if err := scanner.Scan(
		&user.ID,
		&email,
		&googleID,
		&passwordHash,
		&user.IsAdmin,
		&user.IsSuperAdmin,
		&firstName,
		&lastName,
		&patronymic,
		&phone,
		&isMale,
		&birthDate,
		&address,
		&city,
		&avatarURL,
		&user.IsActive,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return domain.User{}, err
	}

	user.Email = optionalString(email)
	user.GoogleID = optionalString(googleID)
	user.PasswordHash = optionalString(passwordHash)
	user.FirstName = firstName.String
	user.LastName = lastName.String
	user.Patronymic = patronymic.String
	user.Phone = optionalString(phone)
	if isMale.Valid {
		user.IsMale = &isMale.Bool
	}
	user.BirthDate = optionalDateString(birthDate)
	user.Address = optionalString(address)
	user.City = optionalString(city)
	user.AvatarURL = optionalString(avatarURL)

	return user, nil
}
