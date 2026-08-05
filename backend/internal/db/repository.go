package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type QueryOption func(*queryBuilder)

type Filter struct {
	Column string
	Op    string
	Value interface{}
}

type queryBuilder struct {
	limit   int
	offset  int
	filters []Filter
	orderBy string
}

func WithLimit(n int) QueryOption {
	return func(q *queryBuilder) { q.limit = n }
}

func WithOffset(n int) QueryOption {
	return func(q *queryBuilder) { q.offset = n }
}

func WithFilter(col string, arg interface{}) QueryOption {
	return func(q *queryBuilder) {
		q.filters = append(q.filters, Filter{Column: col, Op: "=", Value: arg})
	}
}

func WithRangeFilter(col, op string, arg interface{}) QueryOption {
	return func(q *queryBuilder) {
		q.filters = append(q.filters, Filter{Column: col, Op: op, Value: arg})
	}
}

func WithOrderBy(col, dir string) QueryOption {
	return func(q *queryBuilder) { q.orderBy = fmt.Sprintf("ORDER BY %s %s", col, dir) }
}

type BaseRepository[T any] struct {
	db       *sqlx.DB
	table    string
	idColumn string
}

func NewBaseRepository[T any](db *sqlx.DB, table, idColumn string) *BaseRepository[T] {
	return &BaseRepository[T]{db: db, table: table, idColumn: idColumn}
}

func (r *BaseRepository[T]) GetByID(ctx context.Context, id uuid.UUID) (*T, error) {
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s = $1", r.table, r.idColumn)
	var entity T
	err := r.db.GetContext(ctx, &entity, query, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &entity, err
}

func (r *BaseRepository[T]) List(ctx context.Context, tenantID uuid.UUID, opts ...QueryOption) ([]*T, error) {
	q := &queryBuilder{limit: 100}
	for _, o := range opts {
		o(q)
	}

	where := "WHERE tenant_id = $1"
	args := []interface{}{tenantID}
	argIdx := 2
	for _, f := range q.filters {
		where += fmt.Sprintf(" AND %s %s $%d", f.Column, f.Op, argIdx)
		args = append(args, f.Value)
		argIdx++
	}

	query := fmt.Sprintf("SELECT * FROM %s %s", r.table, where)
	if q.orderBy != "" {
		query += " " + q.orderBy
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", q.limit, q.offset)

	var entities []*T
	return entities, r.db.SelectContext(ctx, &entities, query, args...)
}

func (r *BaseRepository[T]) Create(ctx context.Context, entity *T) (*T, error) {
	query := fmt.Sprintf("INSERT INTO %s RETURNING *", r.table)
	rows, err := r.db.NamedQueryContext(ctx, query, entity)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if rows.Next() {
		var created T
		return &created, rows.StructScan(&created)
	}
	return nil, fmt.Errorf("insert failed for %s", r.table)
}

func (r *BaseRepository[T]) Update(ctx context.Context, entity *T) (*T, error) {
	query := fmt.Sprintf("UPDATE %s RETURNING *", r.table)
	_, err := r.db.NamedExecContext(ctx, query, entity)
	return entity, err
}

func (r *BaseRepository[T]) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s WHERE %s = $1", r.table, r.idColumn), id)
	return err
}

type BitemporalRepository[T any] struct {
	*BaseRepository[T]
}

func NewBitemporalRepository[T any](db *sqlx.DB, table, idColumn string) *BitemporalRepository[T] {
	return &BitemporalRepository[T]{
		BaseRepository: NewBaseRepository[T](db, table, idColumn),
	}
}

func (r *BitemporalRepository[T]) GetCurrent(ctx context.Context, id, tenantID uuid.UUID) (*T, error) {
	query := fmt.Sprintf(
		"SELECT * FROM %s WHERE %s = $1 AND tenant_id = $2 AND valid_to = 'infinity'",
		r.table, r.idColumn)
	var entity T
	err := r.db.GetContext(ctx, &entity, query, id, tenantID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &entity, err
}

func (r *BitemporalRepository[T]) ListCurrent(ctx context.Context, tenantID uuid.UUID, opts ...QueryOption) ([]*T, error) {
	q := &queryBuilder{limit: 100}
	for _, o := range opts {
		o(q)
	}

	where := fmt.Sprintf("tenant_id = $1 AND valid_to = 'infinity'")
	args := []interface{}{tenantID}
	argIdx := 2
	for _, f := range q.filters {
		where += fmt.Sprintf(" AND %s %s $%d", f.Column, f.Op, argIdx)
		args = append(args, f.Value)
		argIdx++
	}

	query := fmt.Sprintf("SELECT * FROM %s WHERE %s", r.table, where)
	if q.orderBy != "" {
		query += " " + q.orderBy
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", q.limit, q.offset)

	var entities []*T
	return entities, r.db.SelectContext(ctx, &entities, query, args...)
}

func (r *BitemporalRepository[T]) Invalidate(ctx context.Context, id, tenantID uuid.UUID) error {
	_, err := r.db.ExecContext(ctx,
		fmt.Sprintf("UPDATE %s SET valid_to = NOW() WHERE %s = $1 AND tenant_id = $2 AND valid_to = 'infinity'",
			r.table, r.idColumn), id, tenantID)
	return err
}

type StatusUpdater struct {
	db       *sqlx.DB
	table    string
	idColumn string
}

func NewStatusUpdater(db *sqlx.DB, table, idColumn string) *StatusUpdater {
	return &StatusUpdater{db: db, table: table, idColumn: idColumn}
}

func (s *StatusUpdater) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	query := fmt.Sprintf("UPDATE %s SET status = $1 WHERE %s = $2", s.table, s.idColumn)
	_, err := s.db.ExecContext(ctx, query, status, id)
	return err
}

func (s *StatusUpdater) UpdateStatusWithContext(ctx context.Context, id, tenantID uuid.UUID, status string, extraCols map[string]interface{}) error {
	query := fmt.Sprintf("UPDATE %s SET status = $1", s.table)
	args := []interface{}{status}
	argIdx := 2

	if tenantID != uuid.Nil {
		query += fmt.Sprintf(", tenant_id = $%d", argIdx)
		args = append(args, tenantID)
		argIdx++
	}

	for col, val := range extraCols {
		query += fmt.Sprintf(", %s = $%d", col, argIdx)
		args = append(args, val)
		argIdx++
	}

	query += fmt.Sprintf(" WHERE %s = $%d", s.idColumn, argIdx)
	args = append(args, id)

	_, err := s.db.ExecContext(ctx, query, args...)
	return err
}

func (s *StatusUpdater) TransitionStatus(ctx context.Context, id, tenantID uuid.UUID, fromStatus, toStatus string) error {
	query := fmt.Sprintf(
		"UPDATE %s SET status = $1 WHERE %s = $2 AND tenant_id = $3 AND status = $4",
		s.table, s.idColumn)
	_, err := s.db.ExecContext(ctx, query, toStatus, id, tenantID, fromStatus)
	return err
}

func (s *StatusUpdater) TransitionWithActor(ctx context.Context, id uuid.UUID, toStatus string, actor string, decidedAt time.Time) error {
	query := fmt.Sprintf(
		"UPDATE %s SET status = $1, decided_at = $2, decided_by = $3 WHERE %s = $4",
		s.table, s.idColumn)
	_, err := s.db.ExecContext(ctx, query, toStatus, decidedAt, actor, id)
	return err
}

func (s *StatusUpdater) TransitionWithActorAndTenant(ctx context.Context, id, tenantID uuid.UUID, toStatus string, actor string, decidedAt time.Time) error {
	query := fmt.Sprintf(
		"UPDATE %s SET status = $1, decided_at = $2, decided_by = $3 WHERE %s = $4 AND tenant_id = $5",
		s.table, s.idColumn)
	_, err := s.db.ExecContext(ctx, query, toStatus, decidedAt, actor, id, tenantID)
	return err
}
