package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	validator "github.com/go-playground/validator/v10"
	pgx "github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"

	"github.com/hondyman/semlayer/backend/internal/profiler/helpers"
)

type Config struct {
	AlphaDBURL string `env:"ALPHA_DB_URL" envDefault:"postgres://postgres:postgres@localhost/alpha?sslmode=disable"`
	ServerPort string `env:"SERVER_PORT" envDefault:"8080"`
}

type ProfileRequest struct {
	DataSource string   `json:"datasource" validate:"required,postgres_dsn"`
	Schema     string   `json:"schema" validate:"required"`
	Tables     []string `json:"tables" validate:"required,dive,required"`
	SampleSize int      `json:"sample_size,omitempty" validate:"min=1000,max=1000000"`
	FPRate     float64  `json:"fp_rate,omitempty" validate:"min=0.0001,max=0.1"`
	BatchSize  int      `json:"batch_size,omitempty" validate:"min=1,max=1000"`
}

type Job struct {
	ID        string    `json:"id"`
	Status    string    `json:"status"`
	Error     string    `json:"error,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Req       ProfileRequest
	mu        sync.Mutex `json:"-"`
}

type Application struct {
	Logger      *zap.Logger
	AlphaPool   *pgxpool.Pool
	Validate    *validator.Validate
	sourcePools sync.Map
	jobs        sync.Map
	wg          sync.WaitGroup
}

func loadConfig() (*Config, error) {
	return &Config{
		AlphaDBURL: getEnv("ALPHA_DB_URL", "postgres://postgres:postgres@localhost/alpha?sslmode=disable"),
		ServerPort: getEnv("SERVER_PORT", "8080"),
	}, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func postgresDSN(fl validator.FieldLevel) bool {
	re := regexp.MustCompile(`^postgres://[a-zA-Z0-9_]+:[a-zA-Z0-9_]+@[a-zA-Z0-9_.]+:\d+/[a-zA-Z0-9_]+(\?.*)?$`)
	return re.MatchString(fl.Field().String())
}

func (j *Job) setStatus(status, err string) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.Status = status
	j.Error = err
}

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		panic(fmt.Sprintf("cannot initialize zap logger: %v", err))
	}
	defer logger.Sync()

	cfg, err := loadConfig()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	alphaPool, err := pgxpool.New(context.Background(), cfg.AlphaDBURL)
	if err != nil {
		logger.Fatal("failed to connect to alpha DB", zap.Error(err))
	}
	defer alphaPool.Close()

	validate := validator.New(validator.WithRequiredStructEnabled())
	validate.RegisterValidation("postgres_dsn", postgresDSN)

	app := &Application{
		Logger:    logger,
		AlphaPool: alphaPool,
		Validate:  validate,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/profile", app.profileHandler)
	mux.HandleFunc("/profile/status/", app.profileStatusHandler)
	mux.HandleFunc("/results", app.resultsHandler)
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    ":" + cfg.ServerPort,
		Handler: mux,
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	go func() {
		app.Logger.Info("server starting", zap.String("addr", server.Addr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			app.Logger.Fatal("server failed", zap.Error(err))
		}
	}()

	<-stop
	app.Logger.Info("shutting down server, waiting for active jobs to finish")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		app.Logger.Fatal("server shutdown failed", zap.Error(err))
	}

	app.wg.Wait()

	app.AlphaPool.Close()
	app.Logger.Info("server stopped gracefully")
}

func (app *Application) profileHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		app.Logger.Warn("failed to decode request", zap.Error(err))
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}

	if err := app.Validate.Struct(&req); err != nil {
		app.Logger.Warn("validation failed", zap.Error(err))
		http.Error(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	jobID := generateJobID()
	job := &Job{
		ID:        jobID,
		Status:    "PENDING",
		CreatedAt: time.Now(),
		Req:       req,
	}
	app.jobs.Store(jobID, job)

	go app.processJob(jobID)

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"jobId": jobID, "message": "Profiling job enqueued"})
}

func (app *Application) profileStatusHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	jobID := strings.TrimPrefix(r.URL.Path, "/profile/status/")
	if jobID == "" {
		http.Error(w, "job ID required", http.StatusBadRequest)
		return
	}

	jobIface, ok := app.jobs.Load(jobID)
	if !ok {
		http.Error(w, "job not found", http.StatusNotFound)
		return
	}

	job := jobIface.(*Job)
	job.mu.Lock()
	defer job.mu.Unlock()

	response := map[string]interface{}{
		"id":         job.ID,
		"status":     job.Status,
		"error":      job.Error,
		"created_at": job.CreatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		app.Logger.Error("failed to encode job status", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (app *Application) processJob(jobID string) {
	app.wg.Add(1)
	defer app.wg.Done()

	jobIface, ok := app.jobs.Load(jobID)
	if !ok {
		app.Logger.Error("job not found for processing", zap.String("jobId", jobID))
		return
	}
	j := jobIface.(*Job)

	j.setStatus("RUNNING", "")

	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)
	numWorkers := runtime.NumCPU()
	g.SetLimit(numWorkers)

	// Group requested tables by schema. Tables may be qualified as 'schema.table'
	// when originating from the HTTP API's node ID resolution. This mirrors the
	// server behavior and prevents profiling unintended schemas.
	schemaTableMap := make(map[string][]string)
	if j.Req.Schema != "" {
		if len(j.Req.Tables) == 0 {
			schemaTableMap[j.Req.Schema] = []string{}
		} else {
			schemaTableMap[j.Req.Schema] = j.Req.Tables
		}
	} else if len(j.Req.Tables) > 0 {
		for _, t := range j.Req.Tables {
			if strings.Contains(t, ".") {
				parts := strings.SplitN(t, ".", 2)
				sch := parts[0]
				tbl := parts[1]
				schemaTableMap[sch] = append(schemaTableMap[sch], tbl)
			} else {
				schemaTableMap["public"] = append(schemaTableMap["public"], t)
			}
		}
	} else {
		schemaTableMap["public"] = []string{}
	}

	for schema, tables := range schemaTableMap {
		schema := schema
		tables := tables
		for _, table := range tables {
			table := table
			g.Go(func() error {
				return app.profileTable(ctx, j.Req.DataSource, schema, table, j.Req.SampleSize, j.Req.FPRate)
			})
		}
	}

	var finalStatus, errMsg string
	if err := g.Wait(); err != nil {
		finalStatus = "FAILED"
		errMsg = err.Error()
	} else {
		finalStatus = "COMPLETED"
	}
	j.setStatus(finalStatus, errMsg)
}

func generateJobID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func (app *Application) getSourcePool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	if poolIface, ok := app.sourcePools.Load(dsn); ok {
		return poolIface.(*pgxpool.Pool), nil
	}

	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create new source pool: %w", err)
	}

	app.sourcePools.Store(dsn, pool)
	return pool, nil
}

func (app *Application) profileTable(ctx context.Context, ds, schema, table string, sampleSize int, fpRate float64) error {
	pool, err := app.getSourcePool(ctx, ds)
	if err != nil {
		return err
	}

	rows, err := pool.Query(ctx, "SELECT column_name FROM information_schema.columns WHERE table_schema = $1 AND table_name = $2", schema, table)
	if err != nil {
		return fmt.Errorf("failed to fetch columns for %s.%s: %w", schema, table, err)
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var col string
		if err := rows.Scan(&col); err != nil {
			return fmt.Errorf("failed to scan column name: %w", err)
		}
		columns = append(columns, col)
	}

	g, ctx := errgroup.WithContext(ctx)
	for _, col := range columns {
		col := col
		g.Go(func() error {
			return app.profileColumn(ctx, pool, ds, schema, table, col, sampleSize, fpRate)
		})
	}

	return g.Wait()
}

func (app *Application) profileColumn(ctx context.Context, pool *pgxpool.Pool, ds, schema, table, col string, sampleSize int, fpRate float64) error {
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	quotedTable := pgx.Identifier{table}.Sanitize()
	quotedCol := pgx.Identifier{col}.Sanitize()

	query := fmt.Sprintf("SELECT %s FROM %s.%s ORDER BY random() LIMIT $1", quotedCol, quotedSchema, quotedTable)
	rows, err := pool.Query(ctx, query, sampleSize)
	if err != nil {
		return fmt.Errorf("query failed for %s.%s.%s: %w", schema, table, col, err)
	}
	defer rows.Close()

	var values []interface{}
	for rows.Next() {
		var val interface{}
		if err := rows.Scan(&val); err != nil {
			return fmt.Errorf("failed to scan value: %w", err)
		}
		values = append(values, val)
	}

	if len(values) == 0 {
		return nil
	}

	prof := app.computeProfile(values)
	prof.DataSource = ds
	prof.Schema = schema
	prof.TableName = table
	prof.ColumnName = col
	prof.CreatedAt = time.Now()

	bloomBytes, err := helpers.CreateBloomFilter(values, fpRate)
	if err != nil {
		return fmt.Errorf("failed to create bloom filter: %w", err)
	}
	prof.BloomFilter = bloomBytes

	tx, err := app.AlphaPool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO column_profiles (datasource, schema, table_name, column_name, data_type, cardinality, min_length, max_length, avg_length,
			min_value, max_value, avg_value, std_dev, frequent_values, inferred_patterns, bloom_filter, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (datasource, schema, table_name, column_name) DO UPDATE SET
			cardinality = EXCLUDED.cardinality, min_length = EXCLUDED.min_length, max_length = EXCLUDED.max_length,
			avg_length = EXCLUDED.avg_length, min_value = EXCLUDED.min_value, max_value = EXCLUDED.max_value,
			avg_value = EXCLUDED.avg_value, std_dev = EXCLUDED.std_dev, frequent_values = EXCLUDED.frequent_values,
			inferred_patterns = EXCLUDED.inferred_patterns, bloom_filter = EXCLUDED.bloom_filter, created_at = EXCLUDED.created_at`,
		prof.DataSource, prof.Schema, prof.TableName, prof.ColumnName, prof.DataType, prof.Cardinality, prof.MinLength,
		prof.MaxLength, prof.AvgLength, prof.MinValue, prof.MaxValue, prof.AvgValue, prof.StdDev, prof.FrequentValues,
		prof.InferredPatterns, prof.BloomFilter, prof.CreatedAt)
	if err != nil {
		return fmt.Errorf("failed to upsert profile for %s: %w", col, err)
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	app.Logger.Info("profile stored successfully", zap.String("column", col))
	return nil
}

func (app *Application) computeProfile(values []interface{}) *helpers.ColumnProfile {
	return helpers.ComputeProfile(values)
}

func (app *Application) resultsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	rows, err := app.AlphaPool.Query(r.Context(), "SELECT datasource, schema, table_name, column_name, data_type, cardinality FROM column_profiles ORDER BY created_at DESC LIMIT 100")
	if err != nil {
		app.Logger.Error("query failed", zap.Error(err))
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var profiles []helpers.ColumnProfile
	for rows.Next() {
		var p helpers.ColumnProfile
		if err := rows.Scan(&p.DataSource, &p.Schema, &p.TableName, &p.ColumnName, &p.DataType, &p.Cardinality); err != nil {
			app.Logger.Error("scan failed", zap.Error(err))
			continue
		}
		profiles = append(profiles, p)
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string][]helpers.ColumnProfile{"profiles": profiles}); err != nil {
		app.Logger.Error("failed to encode response", zap.Error(err))
	}
}
