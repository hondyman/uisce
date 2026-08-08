package activities

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/exec"

	"github.com/google/uuid"
	"github.com/hondyman/uisce/backend/internal/db"
	"github.com/hondyman/uisce/backend/internal/iceberg"
	"github.com/hondyman/uisce/backend/internal/provisioning"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

type TenantProvisioningActivities struct {
	DB                   *sqlx.DB
	ControlDB            *sqlx.DB
	LakekeeperProvisioner *iceberg.LakekeeperProvisioner
	Logger               *zap.SugaredLogger
	KafkaBrokers         []string
}

func NewTenantProvisioningActivities(db *sql.DB, controlDB *sql.DB, logger *zap.SugaredLogger) *TenantProvisioningActivities {
	lp := iceberg.NewLakekeeperProvisioner(
		os.Getenv("LAKEKEEPER_URL"),
		os.Getenv("S3_BUCKET"),
		os.Getenv("S3_ENDPOINT"),
	)
	return &TenantProvisioningActivities{
		DB:                   sqlx.NewDb(db, "pgx"),
		ControlDB:            sqlx.NewDb(controlDB, "pgx"),
		LakekeeperProvisioner: lp,
		Logger:               logger,
	}
}

func (a *TenantProvisioningActivities) RegisterTenant(ctx context.Context, input provisioning.RegisterTenantInput) (string, error) {
	a.Logger.Infof("Registering tenant: %s (%s)", input.TenantName, input.TenantCode)

	tenantID := uuid.New().String()
	query := `
		INSERT INTO public.tenants (id, name, code, display_name, is_active, gold_copy, status, created_at, updated_at)
		VALUES ($1, $2, $3, $2, true, false, 'provisioning', NOW(), NOW())
		ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, display_name = EXCLUDED.display_name, updated_at = NOW()
		RETURNING id
	`
	var returnedID string
	err := a.ControlDB.GetContext(ctx, &returnedID, query, tenantID, input.TenantName, input.TenantCode)
	if err != nil {
		return "", fmt.Errorf("failed to insert tenant: %w", err)
	}
	if returnedID != tenantID {
		tenantID = returnedID
	}

	a.Logger.Infof("Registered tenant with ID: %s", tenantID)
	return tenantID, nil
}

func (a *TenantProvisioningActivities) RollbackRegisterTenant(ctx context.Context, tenantID string) error {
	a.Logger.Infof("Rolling back tenant: %s", tenantID)

	query := `DELETE FROM public.tenants WHERE id = $1`
	_, err := a.ControlDB.ExecContext(ctx, query, tenantID)
	if err != nil {
		a.Logger.Errorf("Failed to rollback tenant %s: %v", tenantID, err)
		return err
	}
	return nil
}

func (a *TenantProvisioningActivities) RegisterInstance(ctx context.Context, input provisioning.RegisterInstanceInput) (string, error) {
	a.Logger.Infof("Registering instance: %s for tenant %s", input.InstanceName, input.TenantID)

	instanceID := uuid.New().String()
	query := `
		INSERT INTO public.tenant_instance (id, tenant_id, instance_name, display_name, config, is_active, status, created_at, updated_at)
		VALUES ($1, $2, $3, $3, '{}', true, 'provisioning', NOW(), NOW())
		ON CONFLICT (tenant_id, instance_name) DO UPDATE SET display_name = EXCLUDED.display_name, updated_at = NOW()
		RETURNING id
	`
	var returnedID string
	err := a.ControlDB.GetContext(ctx, &returnedID, query, instanceID, input.TenantID, input.InstanceName)
	if err != nil {
		return "", fmt.Errorf("failed to insert instance: %w", err)
	}
	if returnedID != instanceID {
		instanceID = returnedID
	}

	a.Logger.Infof("Registered instance with ID: %s", instanceID)
	return instanceID, nil
}

func (a *TenantProvisioningActivities) RollbackRegisterInstance(ctx context.Context, instanceID string) error {
	a.Logger.Infof("Rolling back instance: %s", instanceID)

	query := `DELETE FROM public.tenant_instance WHERE id = $1`
	_, err := a.ControlDB.ExecContext(ctx, query, instanceID)
	if err != nil {
		a.Logger.Errorf("Failed to rollback instance %s: %v", instanceID, err)
		return err
	}
	return nil
}

func (a *TenantProvisioningActivities) CreateTenantDatabase(ctx context.Context, databaseName string) error {
	a.Logger.Infof("Creating database: %s", databaseName)

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbHost := os.Getenv("DB_HOST")
		if dbHost == "" {
			dbHost = "localhost"
		}
		dbPort := os.Getenv("DB_PORT")
		if dbPort == "" {
			dbPort = "5432"
		}
		dbUser := os.Getenv("DB_USER")
		if dbUser == "" {
			dbUser = "postgres"
		}
		dbPass := os.Getenv("DB_PASS")
		if dbPass == "" {
			dbPass = "postgres"
		}
		dbName := os.Getenv("DB_NAME")
		if dbName == "" {
			dbName = "postgres"
		}
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	_, err = db.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %s", databaseName))
	if err != nil {
		if err.Error() == "pq: database \""+databaseName+"\" already exists" {
			a.Logger.Infof("Database %s already exists, continuing", databaseName)
			return nil
		}
		return fmt.Errorf("failed to create database: %w", err)
	}

	a.Logger.Infof("Created database: %s", databaseName)
	return nil
}

func (a *TenantProvisioningActivities) RollbackCreateTenantDatabase(ctx context.Context, databaseName string) error {
	a.Logger.Infof("Rolling back database: %s", databaseName)

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbHost := os.Getenv("DB_HOST")
		if dbHost == "" {
			dbHost = "localhost"
		}
		dbPort := os.Getenv("DB_PORT")
		if dbPort == "" {
			dbPort = "5432"
		}
		dbUser := os.Getenv("DB_USER")
		if dbUser == "" {
			dbUser = "postgres"
		}
		dbPass := os.Getenv("DB_PASS")
		if dbPass == "" {
			dbPass = "postgres"
		}
		dbName := os.Getenv("DB_NAME")
		if dbName == "" {
			dbName = "postgres"
		}
		dbURL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbUser, dbPass, dbHost, dbPort, dbName)
	}

	dbConn, err := sql.Open("postgres", dbURL)
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer dbConn.Close()

	_, err = dbConn.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %s", databaseName))
	if err != nil {
		a.Logger.Errorf("Failed to rollback database %s: %v", databaseName, err)
		return err
	}
	return nil
}

func (a *TenantProvisioningActivities) CloneSchemaFromGoldCopy(ctx context.Context, input provisioning.CloneSchemaInput) error {
	a.Logger.Infof("Cloning schema from %s to %s", input.SourceDatabase, input.TargetDatabase)

	dbHost := os.Getenv("DB_HOST")
	if dbHost == "" {
		dbHost = "localhost"
	}
	dbPort := os.Getenv("DB_PORT")
	if dbPort == "" {
		dbPort = "5432"
	}
	dbUser := os.Getenv("DB_USER")
	if dbUser == "" {
		dbUser = "postgres"
	}
	dbPass := os.Getenv("DB_PASS")
	if dbPass == "" {
		dbPass = "postgres"
	}

	cmd := exec.CommandContext(ctx, "pg_dump",
		"-h", dbHost,
		"-p", dbPort,
		"-U", dbUser,
		"-d", input.SourceDatabase,
		"--schema-only",
		"--no-owner",
		"--no-acl",
	)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PGPASSWORD=%s", dbPass),
	)

	pipeCmd := exec.CommandContext(ctx, "psql",
		"-h", dbHost,
		"-p", dbPort,
		"-U", dbUser,
		"-d", input.TargetDatabase,
		"--quiet",
	)
	pipeCmd.Env = append(os.Environ(),
		fmt.Sprintf("PGPASSWORD=%s", dbPass),
	)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("failed to get stdout pipe: %w", err)
	}
	pipeCmd.Stdin = stdoutPipe

	stderrPipe, err := pipeCmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start pg_dump: %w", err)
	}

	if err := pipeCmd.Start(); err != nil {
		return fmt.Errorf("failed to start psql: %w", err)
	}

	err = pipeCmd.Wait()
	if err != nil {
		stderrBytes := make([]byte, 4096)
		stderrPipe.Read(stderrBytes)
		return fmt.Errorf("schema clone failed: %w (stderr: %s)", err, string(stderrBytes))
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("pg_dump failed: %w", err)
	}

	a.Logger.Infof("Successfully cloned schema from %s to %s", input.SourceDatabase, input.TargetDatabase)
	return nil
}

func (a *TenantProvisioningActivities) CreateLakekeeperNamespace(ctx context.Context, namespace string) error {
	a.Logger.Infof("Creating Lakekeeper namespace: %s", namespace)

	err := a.LakekeeperProvisioner.CreateNamespace(ctx, namespace)
	if err != nil {
		return fmt.Errorf("failed to create Lakekeeper namespace: %w", err)
	}

	a.Logger.Infof("Created Lakekeeper namespace: %s", namespace)
	return nil
}

func (a *TenantProvisioningActivities) RollbackCreateLakekeeperNamespace(ctx context.Context, namespace string) error {
	a.Logger.Infof("Rolling back Lakekeeper namespace: %s", namespace)

	err := a.LakekeeperProvisioner.DeleteNamespace(ctx, namespace)
	if err != nil {
		a.Logger.Errorf("Failed to rollback Lakekeeper namespace %s: %v", namespace, err)
		return err
	}
	return nil
}

func (a *TenantProvisioningActivities) CloneGoldCopyProducts(ctx context.Context, input provisioning.CloneProductsInput) error {
	a.Logger.Infof("Cloning gold copy products from tenant %s instance %s to tenant %s instance %s",
		input.GoldCopyTenantID, input.GoldCopyInstanceID, input.TargetTenantID, input.TargetInstanceID)

	targetTenantID, err := uuid.Parse(input.TargetTenantID)
	if err != nil {
		return fmt.Errorf("invalid target tenant ID: %w", err)
	}
	targetInstanceID, err := uuid.Parse(input.TargetInstanceID)
	if err != nil {
		return fmt.Errorf("invalid target instance ID: %w", err)
	}

	_, err = db.CloneGoldCopyInstance(ctx, a.ControlDB, targetTenantID, targetInstanceID)
	if err != nil {
		return fmt.Errorf("failed to clone gold copy products: %w", err)
	}

	a.Logger.Infof("Successfully cloned gold copy products")
	return nil
}

func (a *TenantProvisioningActivities) RollbackCloneGoldCopyProducts(ctx context.Context, input provisioning.CloneProductsInput) error {
	a.Logger.Infof("Rolling back cloned products for tenant %s", input.TargetTenantID)

	err := db.DeleteClonedProducts(ctx, a.ControlDB, input.TargetTenantID, input.TargetInstanceID)
	if err != nil {
		a.Logger.Errorf("Failed to rollback cloned products: %v", err)
		return err
	}
	return nil
}

func (a *TenantProvisioningActivities) EmitProvisioningEvent(ctx context.Context, input provisioning.EmitEventInput) error {
	a.Logger.Infof("Emitting provisioning event for tenant %s (status: %s)", input.TenantID, input.Status)

	if input.Status == "failed" {
		a.Logger.Warnf("Provisioning failed for tenant %s: %s", input.TenantID, input.Error)
	}

	return nil
}

func (a *TenantProvisioningActivities) UpdateTenantStatus(ctx context.Context, tenantID, status string) error {
	a.Logger.Infof("Updating tenant %s status to %s", tenantID, status)

	query := `UPDATE public.tenants SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := a.ControlDB.ExecContext(ctx, query, status, tenantID)
	if err != nil {
		return fmt.Errorf("failed to update tenant status: %w", err)
	}
	return nil
}

func (a *TenantProvisioningActivities) UpdateInstanceStatus(ctx context.Context, instanceID, status string) error {
	a.Logger.Infof("Updating instance %s status to %s", instanceID, status)

	query := `UPDATE public.tenant_instance SET status = $1, updated_at = NOW() WHERE id = $2`
	_, err := a.ControlDB.ExecContext(ctx, query, status, instanceID)
	if err != nil {
		return fmt.Errorf("failed to update instance status: %w", err)
	}
	return nil
}

func (a *TenantProvisioningActivities) GetGoldCopyInfo(ctx context.Context) (string, string, string, error) {
	a.Logger.Info("Resolving gold copy tenant and instance")

	var result struct {
		TenantID   string `db:"tenant_id"`
		InstanceID string `db:"instance_id"`
		Database   string `db:"database_name"`
	}

	query := `
		SELECT t.id as tenant_id, ti.id as instance_id, COALESCE(t.database_name, 'alpha') as database_name
		FROM public.tenants t
		JOIN public.tenant_instance ti ON ti.tenant_id = t.id
		WHERE t.gold_copy = true
		LIMIT 1
	`
	err := a.ControlDB.GetContext(ctx, &result, query)
	if err != nil {
		if err == sql.ErrNoRows {
			return "", "", "", fmt.Errorf("no gold copy tenant found")
		}
		return "", "", "", fmt.Errorf("failed to query gold copy: %w", err)
	}

	a.Logger.Infof("Gold copy resolved: tenant=%s, instance=%s, database=%s",
		result.TenantID, result.InstanceID, result.Database)
	return result.TenantID, result.InstanceID, result.Database, nil
}

func (a *TenantProvisioningActivities) HealthCheck(ctx context.Context) error {
	if a.LakekeeperProvisioner == nil {
		return nil
	}
	return a.LakekeeperProvisioner.HealthCheck(ctx)
}
