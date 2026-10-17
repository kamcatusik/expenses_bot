package main

import (
	"context"
	"database/sql"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os/signal"
	"sort"
	"syscall"
	"time"
	_ "time/tzdata"

	_ "github.com/jackc/pgx/v5/stdlib"

	"expense-tracker/backend/internal/config"
	"expense-tracker/backend/internal/repository"
	apihttp "expense-tracker/backend/internal/transport/http"
	"expense-tracker/backend/migrations"
)

func main() {
	cfg := config.Load()

	loc, err := time.LoadLocation(cfg.TZ)
	if err != nil {
		log.Fatalf("bad TZ %q: %v", cfg.TZ, err)
	}

	db, err := sql.Open("pgx", cfg.DatabaseURL)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	waitForDB(db)
	if err := migrate(db); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           apihttp.NewRouter(repository.NewExpenses(db), repository.NewUsers(db), loc),
		ReadHeaderTimeout: 5 * time.Second,
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

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

func waitForDB(db *sql.DB) {
	for i := 0; i < 30; i++ {
		if err := db.Ping(); err == nil {
			return
		}
		log.Println("waiting for database...")
		time.Sleep(time.Second)
	}
	log.Fatal("database is unavailable")
}

// migrate применяет *.sql по порядку имён. Файлы написаны идемпотентно (IF NOT EXISTS).
func migrate(db *sql.DB) error {
	files, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return err
	}
	sort.Strings(files)
	for _, f := range files {
		body, err := migrations.FS.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := db.Exec(string(body)); err != nil {
			return err
		}
		log.Printf("applied migration %s", f)
	}
	return nil
}
