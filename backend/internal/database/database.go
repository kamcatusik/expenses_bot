package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"sort"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"expense-tracker/backend/migrations"
)

// Config содержит параметры для подключения и настройки пула БД.
type Config struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// DefaultConfig возвращает параметры по умолчанию для подключения к БД.
func DefaultConfig(url string) Config {
	return Config{
		URL:             url,
		MaxOpenConns:    25,
		MaxIdleConns:    10,
		ConnMaxLifetime: 5 * time.Minute,
		ConnMaxIdleTime: 1 * time.Minute,
	}
}

// Open открывает соединение с БД, настраивает пул соединений и дожидается готовности.
func Open(ctx context.Context, cfg Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		db.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	if err := waitForDB(ctx, db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

func waitForDB(ctx context.Context, db *sql.DB) error {
	for i := 0; i < 30; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		err := db.PingContext(pingCtx)
		cancel()
		if err == nil {
			return nil
		}

		log.Println("waiting for database...")
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
	return fmt.Errorf("database is unavailable after 30 attempts")
}

// Migrate применяет *.sql по порядку имён. Файлы написаны идемпотентно (IF NOT EXISTS).
func Migrate(ctx context.Context, db *sql.DB) error {
	files, err := fs.Glob(migrations.FS, "*.sql")
	if err != nil {
		return fmt.Errorf("glob migrations: %w", err)
	}
	sort.Strings(files)
	for _, f := range files {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		body, err := migrations.FS.ReadFile(f)
		if err != nil {
			return fmt.Errorf("read migration %s: %w", f, err)
		}
		if _, err := db.ExecContext(ctx, string(body)); err != nil {
			return fmt.Errorf("execute migration %s: %w", f, err)
		}
		log.Printf("applied migration %s", f)
	}
	return nil
}
