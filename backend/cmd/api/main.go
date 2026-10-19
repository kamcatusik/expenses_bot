package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
	_ "time/tzdata"

	"expense-tracker/backend/internal/config"
	"expense-tracker/backend/internal/database"
	"expense-tracker/backend/internal/repository"
	apihttp "expense-tracker/backend/internal/transport/http"
)

func main() {
	cfg := config.Load()

	loc, err := time.LoadLocation(cfg.TZ)
	if err != nil {
		log.Fatalf("bad TZ %q: %v", cfg.TZ, err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dbCfg := database.DefaultConfig(cfg.DatabaseURL)
	db, err := database.Open(ctx, dbCfg)
	if err != nil {
		log.Fatalf("database init: %v", err)
	}
	defer db.Close()

	if err := database.Migrate(ctx, db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           apihttp.NewRouter(repository.NewExpenses(db), repository.NewUsers(db), loc),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("backend listening on %s (tz=%s)", cfg.HTTPAddr, cfg.TZ)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
	}()

	select {
	case err := <-errCh: // сервер упал сам, например порт занят
		log.Printf("server: %v", err)
		return
	case <-ctx.Done(): // пришёл SIGINT/SIGTERM
	}
	stop() // вернуть сигналам обычное поведение: повторный Ctrl+C завершит процесс сразу

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}
