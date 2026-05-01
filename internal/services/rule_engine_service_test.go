package services

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"reflect"
	"strings"
	"testing"

	"request-system/internal/entities"
	"request-system/internal/repositories"
	apperrors "request-system/pkg/errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var ruleEngineTestContext = context.Background()

func testPtr[T any](v T) *T {
	return &v
}

type stubRuleRepo struct {
	repositories.OrderRoutingRuleRepositoryInterface
}

type fakeUserRepo struct {
	repositories.UserRepositoryInterface
	findInTx func(ctx context.Context, tx pgx.Tx, id uint64) (*entities.User, error)
}

func (f *fakeUserRepo) FindUserByIDInTx(ctx context.Context, tx pgx.Tx, id uint64) (*entities.User, error) {
	if f.findInTx == nil {
		return nil, fmt.Errorf("unexpected FindUserByIDInTx call")
	}
	return f.findInTx(ctx, tx, id)
}

type queryExpectation struct {
	sqlContains []string
	args        []any
	matcher     func(t *testing.T, sql string, args []any)
	rowValues   []any
	rowErr      error
}

type fakeTx struct {
	t            *testing.T
	expectations []queryExpectation
	index        int
}

func newFakeTx(t *testing.T) *fakeTx {
	t.Helper()
	return &fakeTx{t: t}
}

func (tx *fakeTx) expectQueryRow(exp queryExpectation) {
	tx.expectations = append(tx.expectations, exp)
}

func (tx *fakeTx) ExpectationsMet() {
	tx.t.Helper()
	require.Equal(tx.t, len(tx.expectations), tx.index, "not all SQL expectations were consumed")
}

func (tx *fakeTx) nextExpectation() queryExpectation {
	tx.t.Helper()
	require.Less(tx.t, tx.index, len(tx.expectations), "unexpected QueryRow call")
	exp := tx.expectations[tx.index]
	tx.index++
	return exp
}

func (tx *fakeTx) Begin(context.Context) (pgx.Tx, error) {
	return nil, fmt.Errorf("Begin not implemented in fakeTx")
}

func (tx *fakeTx) Commit(context.Context) error {
	return nil
}

func (tx *fakeTx) Rollback(context.Context) error {
	return nil
}

func (tx *fakeTx) CopyFrom(context.Context, pgx.Identifier, []string, pgx.CopyFromSource) (int64, error) {
	return 0, fmt.Errorf("CopyFrom not implemented in fakeTx")
}

func (tx *fakeTx) SendBatch(context.Context, *pgx.Batch) pgx.BatchResults {
	return fakeBatchResults{}
}

func (tx *fakeTx) LargeObjects() pgx.LargeObjects {
	return pgx.LargeObjects{}
}

func (tx *fakeTx) Prepare(context.Context, string, string) (*pgconn.StatementDescription, error) {
	return nil, fmt.Errorf("Prepare not implemented in fakeTx")
}

func (tx *fakeTx) Exec(context.Context, string, ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag(""), fmt.Errorf("Exec not implemented in fakeTx")
}

func (tx *fakeTx) Query(context.Context, string, ...any) (pgx.Rows, error) {
	return fakeRows{}, fmt.Errorf("Query not implemented in fakeTx")
}

func (tx *fakeTx) QueryRow(_ context.Context, sql string, args ...any) pgx.Row {
	exp := tx.nextExpectation()
	for _, fragment := range exp.sqlContains {
		assert.Contains(tx.t, sql, fragment)
	}
	if exp.matcher != nil {
		exp.matcher(tx.t, sql, args)
	} else {
		assert.Equal(tx.t, exp.args, args)
	}
	return fakeRow{values: exp.rowValues, err: exp.rowErr}
}

func (tx *fakeTx) Conn() *pgx.Conn {
	return nil
}

type fakeBatchResults struct{}

func (fakeBatchResults) Exec() (pgconn.CommandTag, error) { return pgconn.NewCommandTag(""), nil }
func (fakeBatchResults) Query() (pgx.Rows, error)         { return fakeRows{}, nil }
func (fakeBatchResults) QueryRow() pgx.Row                { return fakeRow{err: pgx.ErrNoRows} }
func (fakeBatchResults) Close() error                     { return nil }

type fakeRows struct{}

func (fakeRows) Close()                        {}
func (fakeRows) Err() error                    { return nil }
func (fakeRows) CommandTag() pgconn.CommandTag { return pgconn.NewCommandTag("") }
func (fakeRows) FieldDescriptions() []pgconn.FieldDescription {
	return nil
}
func (fakeRows) Next() bool             { return false }
func (fakeRows) Scan(...any) error      { return pgx.ErrNoRows }
func (fakeRows) Values() ([]any, error) { return nil, nil }
func (fakeRows) RawValues() [][]byte    { return nil }
func (fakeRows) Conn() *pgx.Conn        { return nil }

type fakeRow struct {
	values []any
	err    error
}

func (r fakeRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return fmt.Errorf("scan destination count mismatch: got %d want %d", len(dest), len(r.values))
	}
	for i, target := range dest {
		if err := assignScanValue(target, r.values[i]); err != nil {
			return fmt.Errorf("scan column %d: %w", i, err)
		}
	}
	return nil
}

func assignScanValue(target any, value any) error {
	rv := reflect.ValueOf(target)
	if rv.Kind() != reflect.Ptr || rv.IsNil() {
		return fmt.Errorf("target must be a non-nil pointer")
	}
	return assignReflectValue(rv.Elem(), value)
}

func assignReflectValue(dest reflect.Value, value any) error {
	if dest.Kind() == reflect.Ptr {
		if value == nil {
			dest.Set(reflect.Zero(dest.Type()))
			return nil
		}
		valueRV := reflect.ValueOf(value)
		if valueRV.Kind() == reflect.Ptr {
			if valueRV.IsNil() {
				dest.Set(reflect.Zero(dest.Type()))
				return nil
			}
			if valueRV.Type().AssignableTo(dest.Type()) {
				dest.Set(valueRV)
				return nil
			}
			valueRV = valueRV.Elem()
		}
		ptr := reflect.New(dest.Type().Elem())
		if err := assignReflectValue(ptr.Elem(), valueRV.Interface()); err != nil {
			return err
		}
		dest.Set(ptr)
		return nil
	}

	if value == nil {
		dest.Set(reflect.Zero(dest.Type()))
		return nil
	}

	valueRV := reflect.ValueOf(value)
	if valueRV.Kind() == reflect.Ptr {
		if valueRV.IsNil() {
			dest.Set(reflect.Zero(dest.Type()))
			return nil
		}
		valueRV = valueRV.Elem()
	}

	if valueRV.Type().AssignableTo(dest.Type()) {
		dest.Set(valueRV)
		return nil
	}
	if valueRV.Type().ConvertibleTo(dest.Type()) {
		dest.Set(valueRV.Convert(dest.Type()))
		return nil
	}

	return fmt.Errorf("cannot assign %s to %s", valueRV.Type(), dest.Type())
}

func newRuleEngineService(userRepo repositories.UserRepositoryInterface) *RuleEngineService {
	return &RuleEngineService{
		repo:     stubRuleRepo{},
		userRepo: userRepo,
		logger:   zap.NewNop(),
	}
}

func requireHTTPError(t *testing.T, err error, code int, contains string) *apperrors.HttpError {
	t.Helper()

	var httpErr *apperrors.HttpError
	require.Error(t, err)
	require.ErrorAs(t, err, &httpErr)
	assert.Equal(t, code, httpErr.Code)
	if contains != "" {
		assert.Contains(t, strings.ToLower(httpErr.Message), strings.ToLower(contains))
	}
	return httpErr
}

func activeUser(id uint64, modifiers ...func(*entities.User)) *entities.User {
	user := &entities.User{
		ID:         id,
		Fio:        fmt.Sprintf("User %d", id),
		Email:      fmt.Sprintf("user%d@example.com", id),
		StatusCode: "ACTIVE",
	}
	for _, modifier := range modifiers {
		modifier(user)
	}
	return user
}

func TestMatchesExecutorToStructure(t *testing.T) {
	departmentID := uint64(10)
	otherDepartmentID := uint64(11)
	otdelID := uint64(20)
	branchID := uint64(30)
	otherBranchID := uint64(31)
	officeID := uint64(40)

	tests := []struct {
		name     string
		user     *entities.User
		orderCtx OrderContext
		want     bool
	}{
		{
			name: "department match",
			user: activeUser(1, func(u *entities.User) { u.DepartmentID = testPtr(departmentID) }),
			orderCtx: OrderContext{
				DepartmentID: departmentID,
			},
			want: true,
		},
		{
			name: "department mismatch",
			user: activeUser(2, func(u *entities.User) { u.DepartmentID = testPtr(otherDepartmentID) }),
			orderCtx: OrderContext{
				DepartmentID: departmentID,
			},
			want: false,
		},
		{
			name: "otdel and branch match",
			user: activeUser(3, func(u *entities.User) {
				u.OtdelID = testPtr(otdelID)
				u.BranchID = testPtr(branchID)
			}),
			orderCtx: OrderContext{
				OtdelID:  testPtr(otdelID),
				BranchID: testPtr(branchID),
			},
			want: true,
		},
		{
			name: "otdel matches but branch mismatches",
			user: activeUser(4, func(u *entities.User) {
				u.OtdelID = testPtr(otdelID)
				u.BranchID = testPtr(otherBranchID)
			}),
			orderCtx: OrderContext{
				OtdelID:  testPtr(otdelID),
				BranchID: testPtr(branchID),
			},
			want: false,
		},
		{
			name: "branch only",
			user: activeUser(5, func(u *entities.User) { u.BranchID = testPtr(branchID) }),
			orderCtx: OrderContext{
				BranchID: testPtr(branchID),
			},
			want: true,
		},
		{
			name: "office only",
			user: activeUser(6, func(u *entities.User) { u.OfficeID = testPtr(officeID) }),
			orderCtx: OrderContext{
				OfficeID: testPtr(officeID),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, matchesExecutorToStructure(tt.user, tt.orderCtx))
		})
	}
}

func TestBuildExecutorStructureError(t *testing.T) {
	tests := []struct {
		name     string
		orderCtx OrderContext
		want     string
	}{
		{
			name: "department",
			orderCtx: OrderContext{
				DepartmentID: 10,
			},
			want: "Выбранный исполнитель не относится к выбранному департаменту.",
		},
		{
			name: "otdel",
			orderCtx: OrderContext{
				OtdelID: testPtr(uint64(20)),
			},
			want: "Выбранный исполнитель не относится к выбранному отделу.",
		},
		{
			name: "branch",
			orderCtx: OrderContext{
				BranchID: testPtr(uint64(30)),
			},
			want: "Выбранный исполнитель не относится к выбранному филиалу.",
		},
		{
			name: "office",
			orderCtx: OrderContext{
				OfficeID: testPtr(uint64(40)),
			},
			want: "Выбранный исполнитель не относится к выбранному офису.",
		},
		{
			name:     "default",
			orderCtx: OrderContext{},
			want:     "Выбранный исполнитель не соответствует структуре заявки.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, buildExecutorStructureError(tt.orderCtx))
		})
	}
}

func TestResolveExecutor_ExplicitExecutor_HappyPath(t *testing.T) {
	departmentID := uint64(10)
	executorID := uint64(55)
	userRepo := &fakeUserRepo{
		findInTx: func(context.Context, pgx.Tx, uint64) (*entities.User, error) {
			return activeUser(executorID, func(u *entities.User) {
				u.DepartmentID = testPtr(departmentID)
			}), nil
		},
	}

	result, err := newRuleEngineService(userRepo).ResolveExecutor(ruleEngineTestContext, nil, OrderContext{
		DepartmentID: departmentID,
	}, &executorID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, executorID, result.Executor.ID)
	assert.False(t, result.RuleFound)
	assert.Zero(t, result.StatusID)
}

func TestResolveExecutor_ExplicitExecutor_IgnoresRoutingRules(t *testing.T) {
	departmentID := uint64(10)
	executorID := uint64(77)
	tx := newFakeTx(t)
	userRepo := &fakeUserRepo{
		findInTx: func(context.Context, pgx.Tx, uint64) (*entities.User, error) {
			return activeUser(executorID, func(u *entities.User) {
				u.DepartmentID = testPtr(departmentID)
			}), nil
		},
	}

	result, err := newRuleEngineService(userRepo).ResolveExecutor(ruleEngineTestContext, tx, OrderContext{
		DepartmentID: departmentID,
	}, &executorID)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, executorID, result.Executor.ID)
	tx.ExpectationsMet()
}

func TestResolveExecutor_ExplicitExecutor_UserNotFound(t *testing.T) {
	executorID := uint64(99)
	userRepo := &fakeUserRepo{
		findInTx: func(context.Context, pgx.Tx, uint64) (*entities.User, error) {
			return nil, pgx.ErrNoRows
		},
	}

	_, err := newRuleEngineService(userRepo).ResolveExecutor(ruleEngineTestContext, nil, OrderContext{}, &executorID)

	requireHTTPError(t, err, http.StatusBadRequest, "не найден")
}

func TestResolveExecutor_ExplicitExecutor_UserInactive(t *testing.T) {
	executorID := uint64(99)
	userRepo := &fakeUserRepo{
		findInTx: func(context.Context, pgx.Tx, uint64) (*entities.User, error) {
			return activeUser(executorID, func(u *entities.User) {
				u.StatusCode = "DISABLED"
			}), nil
		},
	}

	_, err := newRuleEngineService(userRepo).ResolveExecutor(ruleEngineTestContext, nil, OrderContext{}, &executorID)

	requireHTTPError(t, err, http.StatusBadRequest, "неактивен")
}

func TestResolveExecutor_ExplicitExecutor_WrongDepartment(t *testing.T) {
	executorID := uint64(99)
	userRepo := &fakeUserRepo{
		findInTx: func(context.Context, pgx.Tx, uint64) (*entities.User, error) {
			return activeUser(executorID, func(u *entities.User) {
				u.DepartmentID = testPtr(uint64(11))
			}), nil
		},
	}

	_, err := newRuleEngineService(userRepo).ResolveExecutor(ruleEngineTestContext, nil, OrderContext{
		DepartmentID: 10,
	}, &executorID)

	requireHTTPError(t, err, http.StatusBadRequest, "департамент")
}

func TestResolveExecutor_ExplicitExecutor_WrongOtdel(t *testing.T) {
	executorID := uint64(99)
	userRepo := &fakeUserRepo{
		findInTx: func(context.Context, pgx.Tx, uint64) (*entities.User, error) {
			return activeUser(executorID, func(u *entities.User) {
				u.OtdelID = testPtr(uint64(21))
			}), nil
		},
	}

	_, err := newRuleEngineService(userRepo).ResolveExecutor(ruleEngineTestContext, nil, OrderContext{
		OtdelID: testPtr(uint64(20)),
	}, &executorID)

	requireHTTPError(t, err, http.StatusBadRequest, "отдел")
}

func TestResolveExecutor_ExplicitExecutor_WrongBranch(t *testing.T) {
	executorID := uint64(99)
	userRepo := &fakeUserRepo{
		findInTx: func(context.Context, pgx.Tx, uint64) (*entities.User, error) {
			return activeUser(executorID, func(u *entities.User) {
				u.BranchID = testPtr(uint64(31))
			}), nil
		},
	}

	_, err := newRuleEngineService(userRepo).ResolveExecutor(ruleEngineTestContext, nil, OrderContext{
		BranchID: testPtr(uint64(30)),
	}, &executorID)

	requireHTTPError(t, err, http.StatusBadRequest, "филиал")
}

func TestResolveExecutor_ExplicitExecutor_WrongOffice(t *testing.T) {
	executorID := uint64(99)
	userRepo := &fakeUserRepo{
		findInTx: func(context.Context, pgx.Tx, uint64) (*entities.User, error) {
			return activeUser(executorID, func(u *entities.User) {
				u.OfficeID = testPtr(uint64(41))
			}), nil
		},
	}

	_, err := newRuleEngineService(userRepo).ResolveExecutor(ruleEngineTestContext, nil, OrderContext{
		OfficeID: testPtr(uint64(40)),
	}, &executorID)

	requireHTTPError(t, err, http.StatusBadRequest, "офис")
}

func TestResolveExecutor_RuleFound_UserFound(t *testing.T) {
	orderCtx := OrderContext{
		OrderTypeID:  5,
		DepartmentID: 10,
	}
	tx := newFakeTx(t)
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM order_routing_rules", "ORDER BY order_type_id NULLS LAST"},
		args:        []any{orderCtx.OrderTypeID, orderCtx.DepartmentID, orderCtx.OtdelID, orderCtx.BranchID, orderCtx.OfficeID},
		rowValues:   []any{101, 7, nil, nil, nil, nil},
	})
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "JOIN user_positions up", "WHERE up.position_id = $1"},
		args:        []any{uint64(101), uint64(10)},
		rowValues:   []any{501, "Rule User", "rule@example.com", uint64(101), uint64(10), nil},
	})

	result, err := newRuleEngineService(nil).ResolveExecutor(ruleEngineTestContext, tx, orderCtx, nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.RuleFound)
	assert.Equal(t, 7, result.StatusID)
	assert.Equal(t, uint64(501), result.Executor.ID)
	tx.ExpectationsMet()
}

func TestResolveExecutor_RuleFound_UserNotFound_LosesRuleStatusAndReturnsFallback(t *testing.T) {
	orderCtx := OrderContext{
		OrderTypeID:  5,
		DepartmentID: 10,
	}
	tx := newFakeTx(t)
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM order_routing_rules", "LIMIT 1"},
		args:        []any{orderCtx.OrderTypeID, orderCtx.DepartmentID, orderCtx.OtdelID, orderCtx.BranchID, orderCtx.OfficeID},
		rowValues:   []any{101, 7, nil, nil, nil, nil},
	})
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "JOIN user_positions up", "WHERE up.position_id = $1"},
		args:        []any{uint64(101), uint64(10)},
		rowErr:      pgx.ErrNoRows,
	})
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "JOIN user_positions up", "p.type = $1"},
		args:        []any{"HEAD_OF_DEPARTMENT", uint64(10)},
		rowErr:      pgx.ErrNoRows,
	})
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "JOIN user_positions up", "p.type = $1"},
		args:        []any{"DEPUTY_HEAD_OF_DEPARTMENT", uint64(10)},
		rowValues:   []any{601, "Deputy Head", "deputy@example.com", uint64(901), uint64(10), nil},
	})

	result, err := newRuleEngineService(nil).ResolveExecutor(ruleEngineTestContext, tx, orderCtx, nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, uint64(601), result.Executor.ID)
	// Фиксируем текущее поведение: при fallback статус из правила теряется.
	// Это может быть скрытый баг — при рефакторинге менять только осознанно.
	assert.False(t, result.RuleFound)
	assert.Zero(t, result.StatusID)
	tx.ExpectationsMet()
}

func TestResolveExecutor_NoRule_FallsBackToHierarchy(t *testing.T) {
	orderCtx := OrderContext{
		OrderTypeID:  5,
		DepartmentID: 10,
	}
	tx := newFakeTx(t)
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM order_routing_rules", "LIMIT 1"},
		args:        []any{orderCtx.OrderTypeID, orderCtx.DepartmentID, orderCtx.OtdelID, orderCtx.BranchID, orderCtx.OfficeID},
		rowErr:      pgx.ErrNoRows,
	})
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "JOIN user_positions up", "p.type = $1"},
		args:        []any{"HEAD_OF_DEPARTMENT", uint64(10)},
		rowValues:   []any{701, "Department Head", "head@example.com", uint64(901), uint64(10), nil},
	})

	result, err := newRuleEngineService(nil).ResolveExecutor(ruleEngineTestContext, tx, orderCtx, nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.False(t, result.RuleFound)
	assert.Zero(t, result.StatusID)
	assert.Equal(t, uint64(701), result.Executor.ID)
	tx.ExpectationsMet()
}

func TestResolveByHierarchy_Department_HeadFound(t *testing.T) {
	tx := newFakeTx(t)
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "p.type = $1"},
		args:        []any{"HEAD_OF_DEPARTMENT", uint64(10)},
		rowValues:   []any{801, "Department Head", "head@example.com", uint64(900), uint64(10), nil},
	})

	result, err := newRuleEngineService(nil).resolveByHierarchy(ruleEngineTestContext, tx, OrderContext{
		DepartmentID: 10,
	})

	require.NoError(t, err)
	assert.Equal(t, uint64(801), result.Executor.ID)
	tx.ExpectationsMet()
}

func TestResolveByHierarchy_Department_FallsToDeputy(t *testing.T) {
	tx := newFakeTx(t)
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "p.type = $1"},
		args:        []any{"HEAD_OF_DEPARTMENT", uint64(10)},
		rowErr:      pgx.ErrNoRows,
	})
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "p.type = $1"},
		args:        []any{"DEPUTY_HEAD_OF_DEPARTMENT", uint64(10)},
		rowValues:   []any{802, "Department Deputy", "deputy@example.com", uint64(901), uint64(10), nil},
	})

	result, err := newRuleEngineService(nil).resolveByHierarchy(ruleEngineTestContext, tx, OrderContext{
		DepartmentID: 10,
	})

	require.NoError(t, err)
	assert.Equal(t, uint64(802), result.Executor.ID)
	tx.ExpectationsMet()
}

func TestResolveByHierarchy_HeadBranchNameFromEnv(t *testing.T) {
	branchID := uint64(20)
	officeID := uint64(30)
	previousValue, hadValue := os.LookupEnv("HEAD_BRANCH_NAMES")
	require.NoError(t, os.Setenv("HEAD_BRANCH_NAMES", "HQ"))
	t.Cleanup(func() {
		if hadValue {
			_ = os.Setenv("HEAD_BRANCH_NAMES", previousValue)
			return
		}
		_ = os.Unsetenv("HEAD_BRANCH_NAMES")
	})

	tx := newFakeTx(t)
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"SELECT name FROM branches WHERE id = $1"},
		args:        []any{branchID},
		rowValues:   []any{"HQ"},
	})
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "p.type = $1", "u.office_id = $2"},
		args:        []any{"HEAD_OF_DEPARTMENT", officeID},
		rowValues:   []any{803, "Head Branch Department", "head-branch@example.com", uint64(902), nil, nil},
	})

	result, err := newRuleEngineService(nil).resolveByHierarchy(ruleEngineTestContext, tx, OrderContext{
		BranchID: &branchID,
		OfficeID: &officeID,
	})

	require.NoError(t, err)
	assert.Equal(t, uint64(803), result.Executor.ID)
	tx.ExpectationsMet()
}

func TestResolveByHierarchy_BranchNameQuerySkippedWhenBranchNil(t *testing.T) {
	officeID := uint64(40)
	tx := newFakeTx(t)
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "p.type = $1", "u.office_id = $2"},
		args:        []any{"HEAD_OF_OFFICE", officeID},
		rowValues:   []any{804, "Office Head", "office@example.com", uint64(903), nil, nil},
	})

	result, err := newRuleEngineService(nil).resolveByHierarchy(ruleEngineTestContext, tx, OrderContext{
		OfficeID: &officeID,
	})

	require.NoError(t, err)
	assert.Equal(t, uint64(804), result.Executor.ID)
	tx.ExpectationsMet()
}

func TestResolveByHierarchy_Otdel_FallsToManager(t *testing.T) {
	branchID := uint64(20)
	otdelID := uint64(30)
	tx := newFakeTx(t)
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"SELECT name FROM branches WHERE id = $1"},
		args:        []any{branchID},
		rowValues:   []any{"Regional"},
	})
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "p.type = $1", "u.otdel_id = $2", "u.branch_id = $3"},
		args:        []any{"HEAD_OF_OTDEL", otdelID, branchID},
		rowErr:      pgx.ErrNoRows,
	})
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "p.type = $1", "u.otdel_id = $2", "u.branch_id = $3"},
		args:        []any{"DEPUTY_HEAD_OF_OTDEL", otdelID, branchID},
		rowErr:      pgx.ErrNoRows,
	})
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "p.type = $1", "u.otdel_id = $2", "u.branch_id = $3"},
		args:        []any{"MANAGER", otdelID, branchID},
		rowValues:   []any{805, "Otdel Manager", "manager@example.com", uint64(904), nil, &branchID},
	})

	result, err := newRuleEngineService(nil).resolveByHierarchy(ruleEngineTestContext, tx, OrderContext{
		OtdelID:  &otdelID,
		BranchID: &branchID,
	})

	require.NoError(t, err)
	assert.Equal(t, uint64(805), result.Executor.ID)
	tx.ExpectationsMet()
}

func TestResolveByHierarchy_Branch(t *testing.T) {
	branchID := uint64(20)
	tx := newFakeTx(t)
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"SELECT name FROM branches WHERE id = $1"},
		args:        []any{branchID},
		rowValues:   []any{"Regional"},
	})
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "p.type = $1", "u.branch_id = $2"},
		args:        []any{"BRANCH_DIRECTOR", branchID},
		rowValues:   []any{806, "Branch Director", "director@example.com", uint64(905), nil, &branchID},
	})

	result, err := newRuleEngineService(nil).resolveByHierarchy(ruleEngineTestContext, tx, OrderContext{
		BranchID: &branchID,
	})

	require.NoError(t, err)
	assert.Equal(t, uint64(806), result.Executor.ID)
	tx.ExpectationsMet()
}

func TestResolveByHierarchy_Office(t *testing.T) {
	officeID := uint64(40)
	tx := newFakeTx(t)
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "p.type = $1", "u.office_id = $2"},
		args:        []any{"HEAD_OF_OFFICE", officeID},
		rowValues:   []any{807, "Office Head", "office-head@example.com", uint64(906), nil, nil},
	})

	result, err := newRuleEngineService(nil).resolveByHierarchy(ruleEngineTestContext, tx, OrderContext{
		OfficeID: &officeID,
	})

	require.NoError(t, err)
	assert.Equal(t, uint64(807), result.Executor.ID)
	tx.ExpectationsMet()
}

func TestResolveByHierarchy_NobodyFound(t *testing.T) {
	tx := newFakeTx(t)
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "p.type = $1"},
		args:        []any{"HEAD_OF_DEPARTMENT", uint64(10)},
		rowErr:      pgx.ErrNoRows,
	})
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "p.type = $1"},
		args:        []any{"DEPUTY_HEAD_OF_DEPARTMENT", uint64(10)},
		rowErr:      pgx.ErrNoRows,
	})

	_, err := newRuleEngineService(nil).resolveByHierarchy(ruleEngineTestContext, tx, OrderContext{
		DepartmentID: 10,
	})

	requireHTTPError(t, err, http.StatusBadRequest, "не найден")
	tx.ExpectationsMet()
}

func TestResolveByHierarchy_NoStructure(t *testing.T) {
	_, err := newRuleEngineService(nil).resolveByHierarchy(ruleEngineTestContext, nil, OrderContext{})

	requireHTTPError(t, err, http.StatusBadRequest, "структур")
}

func TestResolveByHierarchy_BranchNameQueryWarningDoesNotBreakFallback(t *testing.T) {
	branchID := uint64(22)
	tx := newFakeTx(t)
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"SELECT name FROM branches WHERE id = $1"},
		args:        []any{branchID},
		rowErr:      errors.New("branch lookup failed"),
	})
	tx.expectQueryRow(queryExpectation{
		sqlContains: []string{"FROM users u", "p.type = $1", "u.branch_id = $2"},
		args:        []any{"BRANCH_DIRECTOR", branchID},
		rowValues:   []any{808, "Branch Director", "director2@example.com", uint64(907), nil, &branchID},
	})

	result, err := newRuleEngineService(nil).resolveByHierarchy(ruleEngineTestContext, tx, OrderContext{
		BranchID: &branchID,
	})

	require.NoError(t, err)
	assert.Equal(t, uint64(808), result.Executor.ID)
	tx.ExpectationsMet()
}
