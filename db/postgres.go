package db

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/RehanAthallahAzhar/tokohobby-orders/internal/models"
	_ "github.com/lib/pq"
)

func Connect(ctx context.Context, credential *models.Credential) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%d sslmode=disable TimeZone=Asia/Jakarta",
		credential.Host,
		credential.Username,
		credential.Password,
		credential.DatabaseName,
		credential.Port,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open connection: %w", err)
	}

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db.SetMaxIdleConns(10)
	db.SetMaxOpenConns(100)
	db.SetConnMaxLifetime(time.Hour)

	return db, nil
}
