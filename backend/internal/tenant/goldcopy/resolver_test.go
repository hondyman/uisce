package goldcopy

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

type mockRedis struct {
	getFn func(ctx context.Context, key string) (string, error)
	setFn func(ctx context.Context, key string, val interface{}, exp time.Duration) error
	delFn func(ctx context.Context, keys ...string) error
}

func (m *mockRedis) Get(ctx context.Context, key string) (string, error) {
	if m.getFn != nil {
		return m.getFn(ctx, key)
	}
	return "", redis.Nil
}

func (m *mockRedis) Set(ctx context.Context, key string, val interface{}, exp time.Duration) error {
	if m.setFn != nil {
		return m.setFn(ctx, key, val, exp)
	}
	return nil
}

func (m *mockRedis) Del(ctx context.Context, keys ...string) error {
	if m.delFn != nil {
		return m.delFn(ctx, keys...)
	}
	return nil
}

func (m *mockRedis) Ping(ctx context.Context) error {
	return nil
}

func TestResolver_Resolve_FromDB(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	goldID := uuid.New().String()
	rows := sqlmock.NewRows([]string{"id"}).AddRow(goldID)
	mock.ExpectQuery(`SELECT id FROM public.tenants WHERE gold_copy = true`).
		WillReturnRows(rows)

	r := NewResolver(nil, nil, nil)
	r.db = sqlx.NewDb(db, "postgres")

	id, err := r.resolveFromDB(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id.String() != goldID {
		t.Fatalf("expected %s, got %s", goldID, id.String())
	}
}

func TestResolver_Resolve_DBError(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	mock.ExpectQuery(`SELECT id FROM public.tenants WHERE gold_copy = true`).
		WillReturnError(errors.New("db error"))

	r := NewResolver(nil, nil, nil)
	r.db = sqlx.NewDb(db, "postgres")

	_, err := r.resolveFromDB(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestResolver_Resolve_NoRows(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	rows := sqlmock.NewRows([]string{"id"})
	mock.ExpectQuery(`SELECT id FROM public.tenants WHERE gold_copy = true`).
		WillReturnRows(rows)

	r := NewResolver(nil, nil, nil)
	r.db = sqlx.NewDb(db, "postgres")

	_, err := r.resolveFromDB(context.Background())
	if !errors.Is(err, ErrGoldCopyNotFound) {
		t.Fatalf("expected ErrGoldCopyNotFound, got %v", err)
	}
}

func TestResolver_Resolve_RedisCacheHit(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	cachedID := uuid.New().String()
	redisClient := &mockRedis{
		getFn: func(ctx context.Context, key string) (string, error) {
			if key == CacheKey {
				return cachedID, nil
			}
			return "", redis.Nil
		},
		setFn: func(ctx context.Context, key string, val interface{}, exp time.Duration) error {
			return nil
		},
	}

	r := NewResolver(nil, redisClient, nil)
	r.db = sqlx.NewDb(db, "postgres")

	id, err := r.Resolve(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id.String() != cachedID {
		t.Fatalf("expected cached %s, got %s", cachedID, id.String())
	}
}

func TestResolver_Resolve_RedisMissFallsBackToDB(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	goldID := uuid.New().String()
	rows := sqlmock.NewRows([]string{"id"}).AddRow(goldID)
	mock.ExpectQuery(`SELECT id FROM public.tenants WHERE gold_copy = true`).
		WillReturnRows(rows)

	redisClient := &mockRedis{
		getFn: func(ctx context.Context, key string) (string, error) {
			return "", redis.Nil
		},
		setFn: func(ctx context.Context, key string, val interface{}, exp time.Duration) error {
			return nil
		},
	}

	r := NewResolver(nil, redisClient, nil)
	r.db = sqlx.NewDb(db, "postgres")

	id, err := r.Resolve(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if id.String() != goldID {
		t.Fatalf("expected %s, got %s", goldID, id.String())
	}
}

func TestResolver_Invalidate(t *testing.T) {
	delCalled := false
	redisClient := &mockRedis{
		delFn: func(ctx context.Context, keys ...string) error {
			delCalled = true
			return nil
		},
	}

	r := NewResolver(nil, redisClient, nil)
	err := r.Invalidate(context.Background())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !delCalled {
		t.Fatal("expected Del to be called on redis")
	}
}

func TestResolver_IsGoldCopy(t *testing.T) {
	db, mock, _ := sqlmock.New()
	defer db.Close()

	goldID := uuid.New().String()
	rows := sqlmock.NewRows([]string{"id"}).AddRow(goldID)
	mock.ExpectQuery(`SELECT id FROM public.tenants WHERE gold_copy = true`).
		WillReturnRows(rows)

	r := NewResolver(nil, nil, nil)
	r.db = sqlx.NewDb(db, "postgres")

	isGold, err := r.IsGoldCopy(uuid.Nil)
	if err != nil {
		t.Fatalf("expected no error for uuid.Nil, got %v", err)
	}
	if isGold {
		t.Fatal("expected false for uuid.Nil")
	}

	isGold, err = r.IsGoldCopy(uuid.New())
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if isGold {
		t.Fatal("expected false for non-gold-copy uuid")
	}

	isGold, err = r.IsGoldCopy(uuid.MustParse(goldID))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !isGold {
		t.Fatal("expected true for gold copy uuid")
	}
}
