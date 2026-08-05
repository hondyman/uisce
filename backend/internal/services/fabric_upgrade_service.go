package services

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/models"
	"github.com/jmoiron/sqlx"
)

type patchChangeKind string

const (
	patchAdd       patchChangeKind = "add"
	patchModify    patchChangeKind = "modify"
	patchDeprecate patchChangeKind = "deprecate"
)

type patchChange struct {
	Kind   patchChangeKind
	Path   []string
	Before any
	After  any
	Reason string
}

type patchDiff struct {
	Changes []patchChange
}

type RemovalPolicy int

const (
	DeprecateOnly RemovalPolicy = iota
	HardDelete
)

type UpgradeService struct {
	DB *sqlx.DB
}

func NewUpgradeService(db *sqlx.DB) *UpgradeService {
	return &UpgradeService{DB: db}
}

func (s *UpgradeService) UpgradeCoreModel(ctx context.Context, dsn, driver string, schemas []string, currentDef models.FabricDefn, policy RemovalPolicy, dryRun bool, actorID uuid.UUID) (*patchDiff, *models.FabricDefn, error) {
	return nil, nil, fmt.Errorf("UpgradeCoreModel not implemented: cube semantic layer removed")
}

func (s *UpgradeService) UpgradeExtensionModel(ctx context.Context, currentDef models.FabricDefn, baseDef models.FabricDefn, policy RemovalPolicy, dryRun bool, actorID uuid.UUID) (*patchDiff, *models.FabricDefn, error) {
	return nil, nil, fmt.Errorf("UpgradeExtensionModel not implemented: cube semantic layer removed")
}

func (s *UpgradeService) ApplyUpgrade(ctx context.Context, defID, actorID uuid.UUID, dryRun bool, policy RemovalPolicy) (*models.FabricDefn, error) {
	return nil, fmt.Errorf("ApplyUpgrade not implemented: cube semantic layer removed")
}

func (s *UpgradeService) SimulateUpgrade(ctx context.Context, defID uuid.UUID) (*UpgradeSimulation, error) {
	return nil, fmt.Errorf("SimulateUpgrade not implemented: cube semantic layer removed")
}

type UpgradeSimulation struct {
	FromVersion string
	ToVersion   string
	Diff        patchDiff
	Warnings    []string
}

func (s *UpgradeService) GenerateModelFromDatabase(ctx context.Context, db *sql.DB, schemas []string, modelKey, title, description string, tags []string) (*models.FabricDefn, error) {
	return nil, fmt.Errorf("GenerateModelFromDatabase not implemented: cube semantic layer removed")
}

func (s *UpgradeService) DiscoverCatalog(ctx context.Context, db *sql.DB, schemas []string) (interface{}, error) {
	return nil, fmt.Errorf("DiscoverCatalog not implemented: cube semantic layer removed")
}

func (s *UpgradeService) GetChangesForSimulation(ctx context.Context, a, b, c string) ([]interface{}, error) {
	return nil, nil
}
