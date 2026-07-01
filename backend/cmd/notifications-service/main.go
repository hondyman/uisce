package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	jwtmiddleware "github.com/hondyman/semlayer/libs/jwt-middleware"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	kafka "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	port := getEnv("PORT", "8084")
	databaseURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@100.84.50.65:5432/alpha?sslmode=disable")
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")

	logger.Info("Starting Notifications Service",
		zap.String("port", port),
		zap.String("database_url", maskURL(databaseURL)),
		zap.String("kafka_brokers", kafkaBrokers),
	)

	// Connect to database
	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Fatal("Database connection check failed", zap.Error(err))
	}
	logger.Info("Database connection established")

	// Start event consumer in background
	go consumeValidationEvents(kafkaBrokers, db, logger)

	// HTTP router
	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	// Initialize JWT middleware
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Warn("JWT_SECRET not set, using development secret")
		jwtSecret = "dev-secret"
	}
	jwtMw := jwtmiddleware.NewJWTMiddleware("/health", "/metrics")
	router.Use(jwtMw.Handler)

	// Health check
	router.Get("/health", healthHandler(logger))

	// Metrics
	router.Get("/metrics", metricsHandler(logger))

	// Notification API
	router.Route("/api/notifications", func(r chi.Router) {
		// Send notification
		r.Post("/send", sendNotificationHandler(db, logger))

		// Get notification status
		r.Get("/{notificationID}", getNotificationStatusHandler(db, logger))

		// List recent notifications
		r.Get("/", listNotificationsHandler(db, logger))

		// Mark as read
		r.Put("/{notificationID}/read", markAsReadHandler(db, logger))

		// Get delivery stats
		r.Get("/stats/delivery", getDeliveryStatsHandler(db, logger))
	})

	// Start server
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("Notifications Service listening", zap.String("address", srv.Addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Server error", zap.Error(err))
		}
	}()

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	logger.Info("Shutting down Notifications Service")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("Server shutdown error", zap.Error(err))
	}
	logger.Info("Notifications Service stopped")
}

// ============================================================================
// EVENT CONSUMER
// ============================================================================

func consumeValidationEvents(brokers string, db *sqlx.DB, logger *zap.Logger) {
	topic := "semlayer.validations"
	groupID := "notifications-service-group"

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  strings.Split(brokers, ","),
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3, // 10KB
		MaxBytes: 10e6, // 10MB
	})
	defer r.Close()

	logger.Info("Kafka consumer started",
		zap.String("brokers", brokers),
		zap.String("topic", topic),
		zap.String("group_id", groupID),
	)

	ctx := context.Background()
	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			logger.Error("Failed to fetch message", zap.Error(err))
			time.Sleep(1 * time.Second) // prevent busy loop on error
			continue
		}

		handleValidationEvent(ctx, r, m, db, logger)
	}
}

func handleValidationEvent(ctx context.Context, r *kafka.Reader, msg kafka.Message, db *sqlx.DB, logger *zap.Logger) {
	var event map[string]interface{}
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		logger.Error("Failed to unmarshal event", zap.Error(err))
		if err := r.CommitMessages(ctx, msg); err != nil {
			logger.Error("Failed to commit message after unmarshal error", zap.Error(err))
		}
		return
	}

	logger.Info("Processing validation event", zap.Any("event", event))

	notificationType := "validation_complete"
	status := "sent"
	subject := fmt.Sprintf("Validation: %v", event["validation_id"])
	messageBytes, _ := json.Marshal(event)
	messageStr := string(messageBytes)

	notifID := uuid.New().String()
	_, err := db.ExecContext(ctx, `
		INSERT INTO notifications (id, tenant_id, user_id, type, subject, message, delivery_status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW())
		ON CONFLICT DO NOTHING
	`, notifID, event["tenant_id"], event["user_id"], notificationType, subject, messageStr, status)

	if err != nil {
		logger.Error("Failed to store notification in database", zap.Error(err))
		// Log and commit to avoid infinite loop
	}

	if err := r.CommitMessages(ctx, msg); err != nil {
		logger.Error("Failed to commit message", zap.Error(err))
	}
}

// ============================================================================
// HTTP HANDLERS
// ============================================================================

func healthHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"service":   "notifications-service",
			"timestamp": time.Now().UTC(),
		})
	}
}

func metricsHandler(logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "# HELP notifications_sent_total Total notifications sent\n")
		fmt.Fprintf(w, "# TYPE notifications_sent_total counter\n")
		fmt.Fprintf(w, "notifications_sent_total 0\n")
	}
}

func sendNotificationHandler(db *sqlx.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TenantID string `json:"tenant_id"`
			UserID   string `json:"user_id"`
			Type     string `json:"type"`
			Subject  string `json:"subject"`
			Message  string `json:"message"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(w).Encode(map[string]string{"error": "invalid request"})
			return
		}

		notifID := uuid.New().String()
		_, err := db.ExecContext(r.Context(), `
			INSERT INTO notifications (id, tenant_id, user_id, type, subject, message, delivery_status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, 'pending', NOW(), NOW())
		`, notifID, req.TenantID, req.UserID, req.Type, req.Subject, req.Message)

		if err != nil {
			logger.Error("Failed to send notification", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to send"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"notification_id": notifID,
			"status":          "queued",
		})
	}
}

func getNotificationStatusHandler(db *sqlx.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notificationID := chi.URLParam(r, "notificationID")

		var notification struct {
			ID             string     `db:"id" json:"id"`
			TenantID       string     `db:"tenant_id" json:"tenant_id"`
			Type           string     `db:"type" json:"type"`
			Subject        string     `db:"subject" json:"subject"`
			Message        string     `db:"message" json:"message"`
			DeliveryStatus string     `db:"delivery_status" json:"delivery_status"`
			ReadAt         *time.Time `db:"read_at" json:"read_at"`
			CreatedAt      time.Time  `db:"created_at" json:"created_at"`
			UpdatedAt      time.Time  `db:"updated_at" json:"updated_at"`
		}

		err := db.GetContext(r.Context(), &notification, `
			SELECT id, tenant_id, type, subject, message, delivery_status, read_at, created_at, updated_at
			FROM notifications
			WHERE id = $1
		`, notificationID)

		if err != nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "notification not found"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(notification)
	}
}

func listNotificationsHandler(db *sqlx.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		tenantID := claims.TenantID
		userID := r.URL.Query().Get("user_id")

		type NotifRow struct {
			ID             string    `db:"id" json:"id"`
			Type           string    `db:"type" json:"type"`
			Subject        string    `db:"subject" json:"subject"`
			DeliveryStatus string    `db:"delivery_status" json:"delivery_status"`
			CreatedAt      time.Time `db:"created_at" json:"created_at"`
		}

		var notifications []NotifRow
		var err error

		if userID != "" {
			err = db.SelectContext(r.Context(), &notifications, `
				SELECT id, type, subject, delivery_status, created_at
				FROM notifications
				WHERE tenant_id = $1 AND user_id = $2
				ORDER BY created_at DESC
				LIMIT 50
			`, tenantID, userID)
		} else {
			err = db.SelectContext(r.Context(), &notifications, `
				SELECT id, type, subject, delivery_status, created_at
				FROM notifications
				WHERE tenant_id = $1
				ORDER BY created_at DESC
				LIMIT 50
			`, tenantID)
		}

		if err != nil {
			logger.Error("Failed to list notifications", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to list"})
			return
		}

		if notifications == nil {
			notifications = []NotifRow{}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":         len(notifications),
			"notifications": notifications,
		})
	}
}

func markAsReadHandler(db *sqlx.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notificationID := chi.URLParam(r, "notificationID")

		result, err := db.ExecContext(r.Context(), `
			UPDATE notifications SET read_at = NOW(), updated_at = NOW()
			WHERE id = $1
		`, notificationID)

		if err != nil {
			logger.Error("Failed to mark as read", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to update"})
			return
		}

		rows, _ := result.RowsAffected()
		if rows == 0 {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "notification not found"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func getDeliveryStatsHandler(db *sqlx.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		tenantID := claims.TenantID

		var stats struct {
			Total   int64 `db:"total"`
			Sent    int64 `db:"sent"`
			Failed  int64 `db:"failed"`
			Pending int64 `db:"pending"`
		}

		err := db.GetContext(r.Context(), &stats, `
			SELECT
				COUNT(*) as total,
				COUNT(*) FILTER (WHERE delivery_status = 'sent') as sent,
				COUNT(*) FILTER (WHERE delivery_status = 'failed') as failed,
				COUNT(*) FILTER (WHERE delivery_status = 'pending') as pending
			FROM notifications
			WHERE tenant_id = $1
		`, tenantID)

		if err != nil {
			logger.Error("Failed to get stats", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to get stats"})
			return
		}

		var successRate float64
		if stats.Total > 0 {
			successRate = (float64(stats.Sent) * 100.0) / float64(stats.Total)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total":        stats.Total,
			"sent":         stats.Sent,
			"failed":       stats.Failed,
			"pending":      stats.Pending,
			"success_rate": successRate,
			"timestamp":    time.Now().UTC(),
		})
	}
}

// ============================================================================
// UTILITY FUNCTIONS
// ============================================================================

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func maskURL(url string) string {
	if len(url) > 30 {
		return url[:15] + "..." + url[len(url)-10:]
	}
	return url
}
