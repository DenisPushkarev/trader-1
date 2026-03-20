package postgres

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Client holds a PostgreSQL connection pool.
type Client struct {
	Pool *pgxpool.Pool
}

// NewClient creates a new PostgreSQL client and verifies connectivity.
func NewClient(ctx context.Context, dsn string) (*Client, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("pgxpool new: %w", err)
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("pgxpool ping: %w", err)
	}
	return &Client{Pool: pool}, nil
}

// Close releases all pool connections.
func (c *Client) Close() {
	c.Pool.Close()
}
