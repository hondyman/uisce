package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/hondyman/semlayer/backend/internal/parity"
)

type server struct {
	comparator  *parity.Comparator
	db          *sql.DB
	comparisons prometheus.Counter
	matches     prometheus.Counter
}

func newServer(comp *parity.Comparator, db *sql.DB) *server {
	comparisons := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cube_parity_comparisons_total",
		Help: "Total number of parity comparisons.",
	})
	matches := prometheus.NewCounter(prometheus.CounterOpts{
		Name: "cube_parity_matches_total",
		Help: "Total comparisons that matched within tolerance.",
	})
	prometheus.MustRegister(comparisons, matches)
	return &server{comparator: comp, db: db, comparisons: comparisons, matches: matches}
}

func (s *server) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/compare", s.handleCompare)
	mux.HandleFunc("/compare/batch", s.handleBatch)
	return mux
}

func (s *server) handleCompare(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var req parity.ComparisonRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 5<<20)).Decode(&req); err != nil {
		http.Error(w, "invalid payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	result, err := s.comparator.Compare(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	s.comparisons.Inc()
	if result.Status == parity.StatusMatch {
		s.matches.Inc()
	}
	if s.db != nil {
		if err := parity.StoreResult(r.Context(), s.db, result); err != nil {
			log.Printf("store result error: %v", err)
		}
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func (s *server) handleBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	var requests []parity.ComparisonRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 20<<20)).Decode(&requests); err != nil {
		http.Error(w, "invalid payload: "+err.Error(), http.StatusBadRequest)
		return
	}
	results := make([]parity.ComparisonResult, 0, len(requests))
	for _, req := range requests {
		result, err := s.comparator.Compare(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		s.comparisons.Inc()
		if result.Status == parity.StatusMatch {
			s.matches.Inc()
		}
		if s.db != nil {
			if err := parity.StoreResult(r.Context(), s.db, result); err != nil {
				log.Printf("store result error: %v", err)
			}
		}
		results = append(results, result)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(results)
}

func main() {
	var (
		addr      = flag.String("addr", ":8090", "HTTP listen address")
		tolerance = flag.Float64("tolerance", 1e-6, "numeric tolerance")
		dsn       = flag.String("dsn", os.Getenv("PARITY_DATABASE_URL"), "Postgres/StarRocks DSN")
	)
	flag.Parse()

	comparator := parity.NewComparator(*tolerance)

	var db *sql.DB
	var err error
	if *dsn != "" {
		db, err = sql.Open("postgres", *dsn)
		if err != nil {
			log.Fatalf("failed to open db: %v", err)
		}
		if err := db.Ping(); err != nil {
			log.Fatalf("failed to ping db: %v", err)
		}
		log.Printf("connected to db")
	}

	srv := &http.Server{
		Addr:         *addr,
		Handler:      newServer(comparator, db).routes(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("cube-parity listening on %s", *addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	if db != nil {
		_ = db.Close()
	}
	log.Printf("cube-parity stopped")
}
