package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hondyman/semlayer/backend/models"
	"github.com/jmoiron/sqlx"
)

// DomainService manages hierarchical data domains (3-level).
type DomainService struct {
	db *sqlx.DB
}

// NewDomainService creates a new DomainService.
func NewDomainService(db *sqlx.DB) *DomainService {
	return &DomainService{db: db}
}

func (s *DomainService) List(ctx context.Context) ([]models.DataDomain, error) {
	rows := []models.DataDomain{}
	err := s.db.SelectContext(ctx, &rows, `SELECT id, name, slug, parent_id, level, description, created_by, created_at, updated_at FROM public.data_domain ORDER BY level, name`)
	return rows, err
}

func (s *DomainService) GetByID(ctx context.Context, id uuid.UUID) (*models.DataDomain, error) {
	var d models.DataDomain
	err := s.db.GetContext(ctx, &d, `SELECT id, name, slug, parent_id, level, description, created_by, created_at, updated_at FROM public.data_domain WHERE id = $1`, id)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (s *DomainService) Create(ctx context.Context, in *models.DataDomain) (*models.DataDomain, error) {
	if in.ID == uuid.Nil {
		in.ID = uuid.New()
	}
	now := time.Now().UTC()
	in.CreatedAt = now
	in.UpdatedAt = now
	slug := strings.ToLower(strings.ReplaceAll(in.Slug, " ", "-"))
	if slug == "" {
		slug = strings.ToLower(strings.ReplaceAll(in.Name, " ", "-"))
	}
	in.Slug = slug

	_, err := s.db.ExecContext(ctx, `INSERT INTO public.data_domain (id, name, slug, parent_id, level, description, created_by, created_at, updated_at) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`, in.ID, in.Name, in.Slug, nullableUUID(in.ParentID), in.Level, in.Description, in.CreatedBy, in.CreatedAt, in.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return in, nil
}

func (s *DomainService) Update(ctx context.Context, in *models.DataDomain) (*models.DataDomain, error) {
	in.UpdatedAt = time.Now().UTC()
	_, err := s.db.ExecContext(ctx, `UPDATE public.data_domain SET name=$1, slug=$2, parent_id=$3, level=$4, description=$5, updated_at=$6 WHERE id=$7`, in.Name, in.Slug, nullableUUID(in.ParentID), in.Level, in.Description, in.UpdatedAt, in.ID)
	if err != nil {
		return nil, err
	}
	return in, nil
}

func (s *DomainService) Delete(ctx context.Context, id uuid.UUID) error {
	// Soft behavior: allow delete; children keep parent_id -> NULL
	_, err := s.db.ExecContext(ctx, `DELETE FROM public.data_domain WHERE id=$1`, id)
	return err
}

func (s *DomainService) Search(ctx context.Context, q string, limit int) ([]models.DataDomain, error) {
	if limit <= 0 {
		limit = 10
	}
	like := fmt.Sprintf("%%%s%%", strings.ToLower(q))
	rows := []models.DataDomain{}
	err := s.db.SelectContext(ctx, &rows, `SELECT id, name, slug, parent_id, level, description, created_by, created_at, updated_at FROM public.data_domain WHERE lower(name) LIKE $1 OR lower(slug) LIKE $1 ORDER BY level, name LIMIT $2`, like, limit)
	return rows, err
}

// nullableUUID returns nil or the uuid depending on value for use in Exec parameters
func nullableUUID(u *uuid.UUID) interface{} {
	if u == nil || *u == uuid.Nil {
		return nil
	}
	return *u
}
