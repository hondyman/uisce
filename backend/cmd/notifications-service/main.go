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
	hasuraclient "github.com/hondyman/semlayer/libs/hasura-client"
	jwtmiddleware "github.com/hondyman/semlayer/libs/jwt-middleware"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	kafka "github.com/segmentio/kafka-go"
	"go.uber.org/zap"
)

// HasuraClient interface for GraphQL operations
type HasuraClient interface {
	Query(query string, variables map[string]interface{}) (map[string]interface{}, error)
	Mutate(mutation string, variables map[string]interface{}) (map[string]interface{}, error)
}

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

	// Initialize Hasura client
	hasuraEndpoint := getEnv("HASURA_ENDPOINT", "http://localhost:8080/v1/graphql")
	hasuraAdmin := getEnv("HASURA_ADMIN_SECRET", "newadminsecretkey")
	hasuraClient := hasuraclient.NewHasuraClient(&hasuraclient.HasuraConfig{
		Endpoint:    hasuraEndpoint,
		AdminSecret: hasuraAdmin,
	})

	// Start event consumer in background
	go consumeValidationEvents(kafkaBrokers, hasuraClient, logger)

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
		r.Post("/send", sendNotificationHandler(hasuraClient, logger))

		// Get notification status
		r.Get("/{notificationID}", getNotificationStatusHandler(hasuraClient, logger))

		// List recent notifications
		r.Get("/", listNotificationsHandler(hasuraClient, logger))

		// Mark as read
		r.Put("/{notificationID}/read", markAsReadHandler(hasuraClient, logger))

		// Get delivery stats
		r.Get("/stats/delivery", getDeliveryStatsHandler(hasuraClient, logger))
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

func consumeValidationEvents(brokers string, hc HasuraClient, logger *zap.Logger) {
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

		handleValidationEvent(ctx, r, m, hc, logger)
	}
}

func handleValidationEvent(ctx context.Context, r *kafka.Reader, msg kafka.Message, hc HasuraClient, logger *zap.Logger) {
	var event map[string]interface{}
	if err := json.Unmarshal(msg.Value, &event); err != nil {
		logger.Error("Failed to unmarshal event", zap.Error(err))
		// Continue to commit so we don't get stuck on bad message
		if err := r.CommitMessages(ctx, msg); err != nil {
			logger.Error("Failed to commit message after unmarshal error", zap.Error(err))
		}
		return
	}

	logger.Info("Processing validation event", zap.Any("event", event))

	// Build data for the notification
	notificationType := "validation_complete"
	status := "sent"
	subject := fmt.Sprintf("Validation: %v", event["validation_id"])
	messageBytes, _ := json.Marshal(event)
	messageStr := string(messageBytes)

	// GraphQL mutation to insert a notification
	mutation := `
		mutation InsertNotification($tenant_id: uuid!, $user_id: uuid, $type: String!, $subject: String!, $message: String!, $delivery_status: String!) {
			insert_notifications_one(object: { tenant_id: $tenant_id, user_id: $user_id, type: $type, subject: $subject, message: $message, delivery_status: $delivery_status }) {
				id
			}
		}
	`

	vars := map[string]interface{}{
		"tenant_id":       event["tenant_id"],
		"user_id":         event["user_id"],
		"type":            notificationType,
		"subject":         subject,
		"message":         messageStr,
		"delivery_status": status,
	}

	if _, err := hc.Mutate(mutation, vars); err != nil {
		logger.Error("Failed to store notification in Hasura", zap.Error(err))
		// For retry-able errors, we might NOT want to commit here, causing a re-read.
		// However, for simplicity/safety against infinite loops in this migration,
		// we will log error and commit. Ideally, use a dead-letter queue.
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

func sendNotificationHandler(hc HasuraClient, logger *zap.Logger) http.HandlerFunc {
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

		mutation := `
			mutation SendNotification($tenant_id: uuid!, $user_id: uuid, $type: String!, $subject: String!, $message: String!) {
				insert_notifications_one(object: { tenant_id: $tenant_id, user_id: $user_id, type: $type, subject: $subject, message: $message, delivery_status: "pending" }) {
					id
				}
			}
		`

		vars := map[string]interface{}{
			"tenant_id": req.TenantID,
			"user_id":   req.UserID,
			"type":      req.Type,
			"subject":   req.Subject,
			"message":   req.Message,
		}

		result, err := hc.Mutate(mutation, vars)
		if err != nil {
			logger.Error("Failed to send notification", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to send"})
			return
		}

		// extract id
		var notificationID string
		if ins, ok := result["insert_notifications_one"].(map[string]interface{}); ok {
			if idv, ok := ins["id"].(string); ok {
				notificationID = idv
			}
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(map[string]string{
			"notification_id": notificationID,
			"status":          "queued",
		})
	}
}

func getNotificationStatusHandler(hc HasuraClient, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notificationID := chi.URLParam(r, "notificationID")

		var notification map[string]interface{}
		query := `
			query GetNotification($id: uuid!) {
				notifications_by_pk(id: $id) {
					id
					tenant_id
					type
					subject
					message
					delivery_status
					read_at
					created_at
					updated_at
				}
			}
		`
		result, err := hc.Query(query, map[string]interface{}{"id": notificationID})
		// result already handled above; notification is available

		if nbpk, ok := result["notifications_by_pk"].(map[string]interface{}); ok {
			notification = nbpk
		} else {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "notification not found"})
			return
		}

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

func listNotificationsHandler(hc HasuraClient, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		tenantID := claims.TenantID
		userID := r.URL.Query().Get("user_id")

		var notifications []map[string]interface{}

		if userID != "" {
			query := `
				query ListNotifications($tenantId: uuid!, $userId: uuid!) {
					notifications(where: { tenant_id: { _eq: $tenantId }, user_id: { _eq: $userId } }, order_by: { created_at: desc }, limit: 50) {
						id
						type
						subject
						delivery_status
						created_at
					}
				}
			`
			res, err := hc.Query(query, map[string]interface{}{"tenantId": tenantID, "userId": userID})
			if err != nil {
				logger.Error("Failed to list notifications", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "failed to list"})
				return
			}
			if arr, ok := res["notifications"].([]interface{}); ok {
				for _, item := range arr {
					if m, ok := item.(map[string]interface{}); ok {
						notifications = append(notifications, m)
					}
				}
			}
		} else {
			query := `
				query ListNotifications($tenantId: uuid!) {
					notifications(where: { tenant_id: { _eq: $tenantId } }, order_by: { created_at: desc }, limit: 50) {
						id
						type
						subject
						delivery_status
						created_at
					}
				}
			`
			res, err := hc.Query(query, map[string]interface{}{"tenantId": tenantID})
			if err != nil {
				logger.Error("Failed to list notifications", zap.Error(err))
				w.WriteHeader(http.StatusInternalServerError)
				json.NewEncoder(w).Encode(map[string]string{"error": "failed to list"})
				return
			}
			if arr, ok := res["notifications"].([]interface{}); ok {
				for _, item := range arr {
					if m, ok := item.(map[string]interface{}); ok {
						notifications = append(notifications, m)
					}
				}
			}
		}
		// response constructed above

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"count":         len(notifications),
			"notifications": notifications,
		})
	}
}

func markAsReadHandler(hc HasuraClient, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		notificationID := chi.URLParam(r, "notificationID")

		// Set read_at to current timestamp
		ts := time.Now().UTC().Format(time.RFC3339)
		mutation := `
			mutation MarkRead($id: uuid!, $read_at: timestamptz!) {
				update_notifications_by_pk(pk_columns: { id: $id }, _set: { read_at: $read_at }) {
					id
				}
			}
		`
		vars := map[string]interface{}{"id": notificationID, "read_at": ts}
		res, err := hc.Mutate(mutation, vars)

		if err != nil {
			logger.Error("Failed to mark as read", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to update"})
			return
		}

		// Hasura returns the updated row in update_notifications_by_pk; if nil, it wasn't found
		if updated, ok := res["update_notifications_by_pk"].(map[string]interface{}); !ok || updated == nil {
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]string{"error": "notification not found"})
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

func getDeliveryStatsHandler(hc HasuraClient, logger *zap.Logger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		claims := jwtmiddleware.GetClaimsFromContext(r)
		if claims == nil {
			http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			return
		}
		tenantID := claims.TenantID

		var stats struct {
			Total       int64   `db:"total"`
			Sent        int64   `db:"sent"`
			Failed      int64   `db:"failed"`
			Pending     int64   `db:"pending"`
			SuccessRate float64 `db:"success_rate"`
		}

		// Use Hasura aggregates to fetch counts
		totalQuery := `
			query TotalCount($tenantId: uuid!) {
				notifications_aggregate(where: { tenant_id: { _eq: $tenantId } }) {
					aggregate { count }
				}
			}
		`
		sentQuery := `
			query SentCount($tenantId: uuid!) {
				notifications_aggregate(where: { tenant_id: { _eq: $tenantId }, delivery_status: { _eq: "sent" } }) {
					aggregate { count }
				}
			}
		`
		failedQuery := `
			query FailedCount($tenantId: uuid!) {
				notifications_aggregate(where: { tenant_id: { _eq: $tenantId }, delivery_status: { _eq: "failed" } }) {
					aggregate { count }
				}
			}
		`
		pendingQuery := `
			query PendingCount($tenantId: uuid!) {
				notifications_aggregate(where: { tenant_id: { _eq: $tenantId }, delivery_status: { _eq: "pending" } }) {
					aggregate { count }
				}
			}
		`

		// total
		resTotal, err := hc.Query(totalQuery, map[string]interface{}{"tenantId": tenantID})
		if err != nil {
			logger.Error("Failed to get stats", zap.Error(err))
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": "failed to get stats"})
			return
		}
		getCount := func(val map[string]interface{}) int64 {
			if agg, ok := val["notifications_aggregate"].(map[string]interface{}); ok {
				if inner, ok := agg["aggregate"].(map[string]interface{}); ok {
					if cnt, ok := inner["count"].(float64); ok {
						return int64(cnt)
					}
				}
			}
			return 0
		}
		total := getCount(resTotal)

		resSent, _ := hc.Query(sentQuery, map[string]interface{}{"tenantId": tenantID})
		resFailed, _ := hc.Query(failedQuery, map[string]interface{}{"tenantId": tenantID})
		resPending, _ := hc.Query(pendingQuery, map[string]interface{}{"tenantId": tenantID})
		sent := getCount(resSent)
		failed := getCount(resFailed)
		pending := getCount(resPending)

		var successRate float64 = 0.0
		if total > 0 {
			successRate = (float64(sent) * 100.0) / float64(total)
		}
		// Populate and return
		stats.Total = total
		stats.Sent = sent
		stats.Failed = failed
		stats.Pending = pending
		stats.SuccessRate = successRate

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
