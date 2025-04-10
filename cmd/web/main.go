package main

import (
	"context"
	"database/sql"
	"flag"
	"html/template"
	"log/slog"
	"os"
	"time"

	_ "github.com/lib/pq"

	"github.com/amari03/habit-tracker/internal/data"
)

type application struct {
	logger        *slog.Logger
	addr          *string
	dsn           *string
	habits        *data.HabitModel // Change to concrete type
	templateCache map[string]*template.Template
}

func main() {
	addr := flag.String("addr", "", "HTTP network address")
	dsn := flag.String("dsn", "", "PostgreSQL DSN")
	flag.Parse()

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	db, err := openDB(*dsn)
	if err != nil {
		logger.Error("Unable to connect to DB", "error", err)
		os.Exit(1)
	}
	defer db.Close()
	logger.Info("Database connection established")

	templateCache, err := newTemplateCache()
	if err != nil {
		logger.Error("Template caching failed", "error", err)
		os.Exit(1)
	}

	app := &application{
		logger:        logger,
		addr:          addr,
		dsn:           dsn,
		habits:        &data.HabitModel{DB: db}, // Initialize with DB
		templateCache: templateCache,
	}

	err = app.serve()
	if err != nil {
		logger.Error("Server error", "error", err)
		os.Exit(1)
	}
}

func openDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err = db.PingContext(ctx); err != nil {
		return nil, err
	}
	return db, nil
}
