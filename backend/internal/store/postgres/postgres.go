package postgres

import (
	"context"
	"log/slog"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewConnection(url string, ctx context.Context) (*pgxpool.Pool, error) {
	slog.Info("connecting to postgres")
	conn, err := pgxpool.New(ctx, url)
	if err != nil {
		slog.Error("postgres connect failed", "error", err)
		return nil, err
	}
	return conn, nil
}

type Store struct {
	conn *pgxpool.Pool
}

func NewStore(conn *pgxpool.Pool) *Store {
	return &Store{conn: conn}
}
