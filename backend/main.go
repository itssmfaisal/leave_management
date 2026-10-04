package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "time/tzdata" // lets TZ (e.g. Asia/Dhaka) work in minimal containers
)

func main() {
	dbPath := env("DB_PATH", "leave.db")
	staticDir := env("STATIC_DIR", "")
	addr := env("ADDR", ":8080")

	store, err := OpenStore(dbPath)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer store.Close()

	mux := http.NewServeMux()
	(&API{store: store}).Register(mux)

	// In production the Go server also serves the built React app.
	if staticDir != "" {
		mux.Handle("/", http.FileServer(http.Dir(staticDir)))
	}

	srv := &http.Server{Addr: addr, Handler: logRequests(mux)}
	go func() {
		log.Printf("listening on %s (db=%s, static=%q)", addr, dbPath, staticDir)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatal(err)
		}
	}()

	// On "docker stop", finish in-flight requests and close the database
	// cleanly so all data is in leave.db (important for backups).
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	<-ctx.Done()
	log.Print("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx)
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func logRequests(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s %s", r.Method, r.URL.Path, time.Since(start).Round(time.Millisecond))
	})
}
