package goldcopy

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

const (
	CacheKey    = "tenant:goldcopy:id"
	PositiveTTL = 24 * time.Hour
	NegativeTTL = 5 * time.Minute
)

var ErrGoldCopyNotFound = errors.New("gold copy tenant not found in database")

type RedisClient interface {
	Get(ctx context.Context, key string) (string, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Del(ctx context.Context, keys ...string) error
	Ping(ctx context.Context) error
}

type Resolver struct {
	db    *sqlx.DB
	redis RedisClient
	log   *slog.Logger

	once sync.Once
}

func NewResolver(db *sqlx.DB, redisClient RedisClient, log *slog.Logger) *Resolver {
	return &Resolver{
		db:    db,
		redis: redisClient,
		log:   log,
	}
}

func (r *Resolver) Resolve(ctx context.Context) (uuid.UUID, error) {
	if r.redis != nil {
		val, err := r.redis.Get(ctx, CacheKey)
		if err == nil && val != "" {
			id, err := uuid.Parse(val)
			if err == nil {
				return id, nil
			}
		}
		if err != nil && !errors.Is(err, redis.Nil) {
			r.log.Warn("goldcopy: redis get failed, falling back to DB", "err", err)
		}
	}

	id, err := r.resolveFromDB(ctx)
	if err != nil {
		if r.redis != nil {
			r.setNegativeCache(ctx)
		}
		return uuid.Nil, err
	}

	if r.redis != nil {
		if err := r.redis.Set(ctx, CacheKey, id.String(), PositiveTTL); err != nil {
			r.log.Warn("goldcopy: failed to cache positive result", "err", err)
		}
	}

	return id, nil
}

func (r *Resolver) Warmup(ctx context.Context) (uuid.UUID, error) {
	id, err := r.Resolve(ctx)
	if err != nil {
		r.log.Warn("goldcopy: warmup failed", "err", err)
		return uuid.Nil, err
	}
	r.once.Do(func() {})
	r.log.Info("goldcopy: warmup complete", "gold_copy_tenant_id", id.String())
	return id, nil
}

func (r *Resolver) Invalidate(ctx context.Context) error {
	if r.redis == nil {
		return nil
	}
	if err := r.redis.Del(ctx, CacheKey); err != nil {
		r.log.Warn("goldcopy: failed to invalidate cache", "err", err)
		return err
	}
	r.log.Info("goldcopy: cache invalidated")
	return nil
}

func (r *Resolver) IsGoldCopy(id uuid.UUID) (bool, error) {
	if id == uuid.Nil {
		return false, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	goldID, err := r.Resolve(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to resolve gold copy tenant: %w", err)
	}
	return id == goldID, nil
}

func (r *Resolver) resolveFromDB(ctx context.Context) (uuid.UUID, error) {
	var id string
	query := `SELECT id FROM public.tenants WHERE gold_copy = true ORDER BY created_at LIMIT 1`
	err := r.db.GetContext(ctx, &id, query)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return uuid.Nil, ErrGoldCopyNotFound
		}
		return uuid.Nil, fmt.Errorf("goldcopy: DB query failed: %w", err)
	}
	uid, err := uuid.Parse(id)
	if err != nil {
		return uuid.Nil, fmt.Errorf("goldcopy: invalid UUID from DB: %w", err)
	}
	return uid, nil
}

func (r *Resolver) setNegativeCache(ctx context.Context) {
	if r.redis == nil {
		return
	}
	if err := r.redis.Set(ctx, CacheKey+":negative", "1", NegativeTTL); err != nil {
		r.log.Warn("goldcopy: failed to set negative cache", "err", err)
	}
}

func (r *Resolver) isNegativeCached(ctx context.Context) bool {
	if r.redis == nil {
		return false
	}
	val, err := r.redis.Get(ctx, CacheKey+":negative")
	return err == nil && val == "1"
}
