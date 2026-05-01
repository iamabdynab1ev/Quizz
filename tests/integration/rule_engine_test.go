//go:build integration

package integration

import (
	"context"
	"fmt"
	"os"
	"testing"

	"request-system/internal/repositories"
	"request-system/internal/services"

	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubRuleRepo struct {
	repositories.OrderRoutingRuleRepositoryInterface
}

func ptr[T any](v T) *T {
	return &v
}

func newRuleEngine() services.RuleEngineServiceInterface {
	return services.NewRuleEngineService(stubRuleRepo{}, nil, zap.NewNop())
}

func requireIntegrationTx(t *testing.T) (context.Context, *pgx.Conn, pgx.Tx) {
	t.Helper()

	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	ctx := context.Background()
	conn, err := pgx.Connect(ctx, databaseURL)
	require.NoError(t, err)

	tx, err := conn.Begin(ctx)
	require.NoError(t, err)

	t.Cleanup(func() {
		_ = tx.Rollback(ctx)
		_ = conn.Close(ctx)
	})

	return ctx, conn, tx
}

func setupRuleEngineTempTables(t *testing.T, ctx context.Context, tx pgx.Tx) {
	t.Helper()

	statements := []string{
		`CREATE TEMP TABLE statuses (
			id BIGINT PRIMARY KEY,
			code TEXT NOT NULL
		) ON COMMIT DROP`,
		`CREATE TEMP TABLE positions (
			id BIGINT PRIMARY KEY,
			type TEXT NOT NULL
		) ON COMMIT DROP`,
		`CREATE TEMP TABLE branches (
			id BIGINT PRIMARY KEY,
			name TEXT NOT NULL
		) ON COMMIT DROP`,
		`CREATE TEMP TABLE users (
			id BIGINT PRIMARY KEY,
			fio TEXT NOT NULL,
			email TEXT NOT NULL,
			status_id BIGINT NOT NULL,
			position_id BIGINT,
			department_id BIGINT,
			otdel_id BIGINT,
			branch_id BIGINT,
			office_id BIGINT,
			deleted_at TIMESTAMPTZ
		) ON COMMIT DROP`,
		`CREATE TEMP TABLE user_positions (
			user_id BIGINT NOT NULL,
			position_id BIGINT NOT NULL
		) ON COMMIT DROP`,
		`CREATE TEMP TABLE order_routing_rules (
			order_type_id BIGINT,
			department_id BIGINT,
			otdel_id BIGINT,
			branch_id BIGINT,
			office_id BIGINT,
			assign_to_position_id INTEGER,
			status_id INTEGER
		) ON COMMIT DROP`,
	}

	for _, statement := range statements {
		_, err := tx.Exec(ctx, statement)
		require.NoError(t, err)
	}
}

func insertActiveUserForPosition(t *testing.T, ctx context.Context, tx pgx.Tx, userID uint64, positionID uint64, departmentID uint64, otdelID *uint64, branchID *uint64) {
	t.Helper()

	_, err := tx.Exec(ctx,
		`INSERT INTO users (id, fio, email, status_id, position_id, department_id, otdel_id, branch_id)
		 VALUES ($1, $2, $3, 1, $4, $5, $6, $7)`,
		userID,
		fmt.Sprintf("User %d", userID),
		fmt.Sprintf("user%d@example.com", userID),
		positionID,
		departmentID,
		otdelID,
		branchID,
	)
	require.NoError(t, err)

	_, err = tx.Exec(ctx, `INSERT INTO user_positions (user_id, position_id) VALUES ($1, $2)`, userID, positionID)
	require.NoError(t, err)
}

func seedRuleEngineBaseData(t *testing.T, ctx context.Context, tx pgx.Tx) {
	t.Helper()

	_, err := tx.Exec(ctx, `INSERT INTO statuses (id, code) VALUES (1, 'ACTIVE')`)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, `INSERT INTO positions (id, type) VALUES (100, 'SPECIALIST'), (200, 'SPECIALIST'), (300, 'SPECIALIST')`)
	require.NoError(t, err)
	_, err = tx.Exec(ctx, `INSERT INTO branches (id, name) VALUES (20, 'Regional')`)
	require.NoError(t, err)
}

func TestIntegration_ExactRuleBeatsWildcard(t *testing.T) {
	ctx, _, tx := requireIntegrationTx(t)
	setupRuleEngineTempTables(t, ctx, tx)
	seedRuleEngineBaseData(t, ctx, tx)

	insertActiveUserForPosition(t, ctx, tx, 1, 100, 10, nil, ptr(uint64(20)))
	insertActiveUserForPosition(t, ctx, tx, 2, 200, 10, nil, ptr(uint64(20)))

	_, err := tx.Exec(ctx, `
		INSERT INTO order_routing_rules (order_type_id, department_id, branch_id, assign_to_position_id, status_id)
		VALUES
			(NULL, NULL, NULL, 100, 1),
			(5, 10, 20, 200, 9)
	`)
	require.NoError(t, err)

	result, err := newRuleEngine().ResolveExecutor(ctx, tx, services.OrderContext{
		OrderTypeID:  5,
		DepartmentID: 10,
		BranchID:     ptr(uint64(20)),
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.RuleFound)
	assert.Equal(t, 9, result.StatusID)
	assert.Equal(t, uint64(2), result.Executor.ID)
}

func TestIntegration_MoreSpecificPartialRuleWins(t *testing.T) {
	ctx, _, tx := requireIntegrationTx(t)
	setupRuleEngineTempTables(t, ctx, tx)
	seedRuleEngineBaseData(t, ctx, tx)

	otdelID := uint64(30)
	branchID := uint64(20)
	insertActiveUserForPosition(t, ctx, tx, 1, 100, 10, nil, &branchID)
	insertActiveUserForPosition(t, ctx, tx, 2, 200, 10, &otdelID, &branchID)

	_, err := tx.Exec(ctx, `
		INSERT INTO order_routing_rules (order_type_id, department_id, otdel_id, branch_id, assign_to_position_id, status_id)
		VALUES
			(5, 10, NULL, 20, 100, 1),
			(5, 10, 30, 20, 200, 2)
	`)
	require.NoError(t, err)

	result, err := newRuleEngine().ResolveExecutor(ctx, tx, services.OrderContext{
		OrderTypeID:  5,
		DepartmentID: 10,
		OtdelID:      &otdelID,
		BranchID:     &branchID,
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.RuleFound)
	assert.Equal(t, 2, result.StatusID)
	assert.Equal(t, uint64(2), result.Executor.ID)
}

func TestIntegration_NullsLastOrdering(t *testing.T) {
	ctx, _, tx := requireIntegrationTx(t)
	setupRuleEngineTempTables(t, ctx, tx)
	seedRuleEngineBaseData(t, ctx, tx)

	branchID := uint64(20)
	insertActiveUserForPosition(t, ctx, tx, 1, 100, 10, nil, nil)
	insertActiveUserForPosition(t, ctx, tx, 2, 200, 10, nil, &branchID)

	_, err := tx.Exec(ctx, `
		INSERT INTO order_routing_rules (order_type_id, department_id, branch_id, assign_to_position_id, status_id)
		VALUES
			(5, 10, NULL, 100, 11),
			(NULL, 10, 20, 200, 22)
	`)
	require.NoError(t, err)

	result, err := newRuleEngine().ResolveExecutor(ctx, tx, services.OrderContext{
		OrderTypeID:  5,
		DepartmentID: 10,
		BranchID:     &branchID,
	}, nil)

	require.NoError(t, err)
	require.NotNil(t, result)
	assert.True(t, result.RuleFound)
	assert.Equal(t, 11, result.StatusID)
	assert.Equal(t, uint64(1), result.Executor.ID)
}
