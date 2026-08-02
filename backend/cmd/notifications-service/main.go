package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	jwtmiddleware "github.com/hondyman/uisce/libs/jwt-middleware"
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
	databaseURL := getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/alpha?sslmode=disable")
	kafkaBrokers := getEnv("KAFKA_BROKERS", "localhost:9092")

	logger.Info("Starting Notifications Service",
		zap.String("port", port),
		zap.String("database_url", maskURL(databaseURL)),
		zap.String("kafka_brokers", kafkaBrokers),
	)

	db, err := sqlx.Connect("postgres", databaseURL)
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		logger.Fatal("Database connection check failed", zap.Error(err))
	}
	logger.Info("Database connection established")

	// Start outbox relay consumer in background
	go startOutboxRelay(db, logger)

	// Start validation event consumer in background
	go consumeValidationEvents(kafkaBrokers, db, logger)

	router := chi.NewRouter()
	router.Use(middleware.RequestID)
	router.Use(middleware.RealIP)
	router.Use(middleware.Logger)
	router.Use(middleware.Recoverer)

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Warn("JWT_SECRET not set, using development secret")
		jwtSecret = "dev-secret"
	}
	jwtMw := jwtmiddleware.NewJWTMiddleware("/health", "/metrics")
	router.Use(jwtMw.Handler)

	router.Get("/health", healthHandler(logger))
	router.Get("/metrics", metricsHandler(logger))

	router.Route("/api/notifications", func(r chi.Router) {
		r.Post("/send", sendNotificationHandler(db, logger))
		r.Get("/{notificationID}", getNotificationStatusHandler(db, logger))
		r.Get("/", listNotificationsHandler(db, logger))
		r.Put("/{notificationID}/read", markAsReadHandler(db, logger))
		r.Get("/stats/delivery", getDeliveryStatsHandler(db, logger))
	})

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
// OUTBOX PATTERN - CDC via Debezium
// ============================================================================

func startOutboxRelay(db *sqlx.DB, logger *zap.Logger) {
	logger.Info("Starting outbox relay consumer")

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := processOutboxBatch(ctx, db, logger)
		cancel()

		if err != nil {
			logger.Error("Outbox relay error", zap.Error(err))
			time.Sleep(1 * time.Second)
		}
	}
}

func processOutboxBatch(ctx context.Context, db *sqlx.DB, logger *zap.Logger) error {
	var rows []struct {
		ID            string `db:"id"`
		AggregateType string `db:"aggregate_type"`
		EventType     string `db:"event_type"`
		TenantID      string `db:"tenant_id"`
		UserID        string `db:"user_id"`
		Payload       []byte `db:"payload"`
		CreatedAt     time.Time `db:"created_at"`
	}

	err := db.SelectContext(ctx, &rows, `
		SELECT id, aggregate_type, event_type, tenant_id, user_id, payload, created_at
		FROM notification_outbox
		WHERE processed_at IS NULL
		ORDER BY created_at ASC
		LIMIT 100
	`)
	if err != nil {
		return fmt.Errorf("failed to select outbox rows: %w", err)
	}

	if len(rows) == 0 {
		return nil
	}

	logger.Info("Processing outbox batch", zap.Int("count", len(rows)))

	for _, row := range rows {
		err := processOutboxRow(ctx, db, row)
		if err != nil {
			logger.Error("Failed to process outbox row",
				zap.String("id", row.ID),
				zap.Error(err))
			continue
		}

		_, err = db.ExecContext(ctx, `
			UPDATE notification_outbox
			SET processed_at = NOW()
			WHERE id = $1
		`, row.ID)
		if err != nil {
			logger.Error("Failed to mark outbox row as processed",
				zap.String("id", row.ID),
				zap.Error(err))
		}
	}

	return nil
}

func processOutboxRow(ctx context.Context, db *sqlx.DB, row struct {
	ID            string `db:"id"`
	AggregateType string `db:"aggregate_type"`
	EventType     string `db:"event_type"`
	TenantID      string `db:"tenant_id"`
	UserID        string `db:"user_id"`
	Payload       []byte `db:"payload"`
	CreatedAt     time.Time `db:"created_at"`
}) error {
	switch row.AggregateType {
	case "notification":
		return processNotificationOutbox(ctx, db, row)
	default:
		return nil
	}
}

func processNotificationOutbox(ctx context.Context, db *sqlx.DB, row struct {
	ID            string `db:"id"`
	AggregateType string `db:"aggregate_type"`
	EventType     string `db:"event_type"`
	TenantID      string `db:"tenant_id"`
	UserID        string `db:"user_id"`
	Payload       []byte `db:"payload"`
	CreatedAt     time.Time `db:"created_at"`
}) error {
	var payload map[string]interface{}
	if err := json.Unmarshal(row.Payload, &payload); err != nil {
		return fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	switch row.EventType {
	case "created":
		notificationID := row.ID
		subject, _ := payload["subject"].(string)
		message, _ := payload["message"].(string)
		notifType, _ := payload["type"].(string)
		status := "pending"

		_, err := db.ExecContext(ctx, `
			INSERT INTO notifications (id, tenant_id, user_id, type, subject, message, delivery_status, created_at, updated_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)
			ON CONFLICT (id) DO NOTHING
		`, notificationID, row.TenantID, row.UserID, notifType, subject, message, status, row.CreatedAt)
		return err

	case "read":
		notificationID := payload["notification_id"]
		_, err := db.ExecContext(ctx, `
			UPDATE notifications SET read_at = NOW() WHERE id = $1
		`, notificationID)
		return err

	case "delivered":
		notificationID := payload["notification_id"]
		_, err := db.ExecContext(ctx, `
			UPDATE notifications SET delivery_status = 'delivered', updated_at = NOW() WHERE id = $1
		`, notificationID)
		return err

	default:
		return nil
	}
}

// ============================================================================
// EVENT CONSUMER - writes to outbox
// ============================================================================

func consumeValidationEvents(brokers string, db *sqlx.DB, logger *zap.Logger) {
	topic := "semlayer.validations"
	groupID := "notifications-service-group"

	r := kafka.NewReader(kafka.ReaderConfig{
		Brokers:  strings.Split(brokers, ","),
		GroupID:  groupID,
		Topic:    topic,
		MinBytes: 10e3,
		MaxBytes: 10e6,
	})
	defer r.Close()

	logger.Info("Kafka validation consumer started",
		zap.String("brokers", brokers),
		zap.String("topic", topic),
		zap.String("group_id", groupID),
	)

	ctx := context.Background()
	for {
		m, err := r.FetchMessage(ctx)
		if err != nil {
			logger.Error("Failed to fetch message", zap.Error(err))
			time.Sleep(1 * time.Second)
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
			logger.Error("Failed to commit message", zap.Error(err))
		}
		return
	}

	logger.Info("Processing validation event", zap.Any("event", event))

	notificationType := "validation_complete"
	subject := fmt.Sprintf("Validation: %v", event["validation_id"])
	messageBytes, _ := json.Marshal(event)

	payload := map[string]interface{}{
		"type":            notificationType,
		"subject":         subject,
		"message":         string(messageBytes),
		"validation_id":    event["validation_id"],
	}

	payloadBytes, _ := json.Marshal(payload)

	outboxID := generateUUID()

	_, err := db.ExecContext(ctx, `
		INSERT INTO notification_outbox (id, aggregate_type, event_type, payload, tenant_id, user_id, created_at)
		VALUES ($1, 'notification', 'created', $2, $3, $4, NOW())
	`, outboxID, payloadBytes, event["tenant_id"], event["user_id"])

	if err != nil {
		logger.Error("Failed to write to outbox", zap.Error(err))
	}

	if err := r.CommitMessages(ctx, msg); err != nil {
		logger.Error("Failed to commit message", zap.Error(err))
	}
}

func generateUUID() string {
	return fmt.Sprintf("%s-%s-%s-%s-%s",
		generateRandomString(8),
		generateRandomString(4),
		generateRandomString(4),
		generateRandomString(4),
		generateRandomString(12))
}

func generateRandomString(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[time.Now().UnixNano()%int64(len(chars))]
		time.Sleep(time.Nanosecond)
	}
	return string(result)
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

		notificationID := generateUUID()
		payload := map[string]interface{}{
			"type":    req.Type,
			"subject": req.Subject,
			"message": req.Message,
		}
		payloadBytes, _ := json.Marshal(payload)

		_, err := db.ExecContext(r.Context(), `
			INSERT INTO notification_outbox (id, aggregate_type, event_type, payload, tenant_id, user_id, created_at)
			VALUES ($1, 'notification', 'created', $2, $3, $4, NOW())
		`, notificationID, payloadBytes, req.TenantID, req.UserID)

		if err != nil {
			logger.Error("Failed to send notification", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to send"})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"notification_id": notificationID,
			"status":         "queued",
		})
	}
}

func getNotificationStatusHandler(db *sqlx.DB, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notificationID := chi.URLParam(r, "notificationID")

		var notification struct {
			ID             string     `db:"id"`
			TenantID       string     `db:"tenant_id"`
			Type           string     `db:"type"`
			Subject        string     `db:"subject"`
			Message        string     `db:"message"`
			DeliveryStatus string     `db:"delivery_status"`
			ReadAt         *time.Time `db:"read_at"`
			CreatedAt      time.Time  `db:"created_at"`
			UpdatedAt      time.Time  `db:"updated_at"`
		}

		err := db.GetContext(r.Context(), &notification, `
			SELECT id, tenant_id, type, subject, message, delivery_status, read_at, created_at, updated_at
			FROM notifications WHERE id = $1
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

		var notifications []struct {
			ID             string    `db:"id"`
			Type           string    `db:"type"`
			Subject        string    `db:"subject"`
			DeliveryStatus string    `db:"delivery_status"`
			CreatedAt      time.Time `db:"created_at"`
		}

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

		notificationIDOutbox := generateUUID()
		payload := map[string]interface{}{
			"notification_id": notificationID,
		}
		payloadBytes, _ := json.Marshal(payload)

		_, err := db.ExecContext(r.Context(), `
			INSERT INTO notification_outbox (id, aggregate_type, event_type, payload, tenant_id, created_at)
			SELECT $1, 'notification', 'read', $2, tenant_id, NOW()
			FROM notifications WHERE id = $3
		`, notificationIDOutbox, payloadBytes, notificationID)

		if err != nil {
			logger.Error("Failed to mark as read", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to update"})
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
			Total       int64
			Sent        int64
			Failed      int64
			Pending     int64
			SuccessRate float64
		}

		err := db.QueryRowContext(r.Context(), `
			SELECT
				COUNT(*) as total,
				COUNT(*) FILTER (WHERE delivery_status = 'sent') as sent,
				COUNT(*) FILTER (WHERE delivery_status = 'failed') as failed,
				COUNT(*) FILTER (WHERE delivery_status = 'pending') as pending
			FROM notifications WHERE tenant_id = $1
		`, tenantID).Scan(&stats.Total, &stats.Sent, &stats.Failed, &stats.Pending)

		if err != nil {
			logger.Error("Failed to get stats", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to get stats"})
			return
		}

		if stats.Total > 0 {
			stats.SuccessRate = (float64(stats.Sent) * 100.0) / float64(stats.Total)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"total":        stats.Total,
			"sent":         stats.Sent,
			"failed":       stats.Failed,
			"pending":      stats.Pending,
			"success_rate": stats.SuccessRate,
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
