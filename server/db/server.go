package db

import (
	"context"
	"fmt"

	"github.com/CarsonHoffman/office-hours-queue/server/api"
	"github.com/dlmiddlecote/sqlstats"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
)

type Server struct {
	DB *sqlx.DB
}

func (s *Server) BeginTx() (*sqlx.Tx, error) {
	return s.DB.Beginx()
}

// Does this make calling database functions annoying? You bet!
// It helps the API handlers be less coupled to this database
// code, though. Am I starting to question certain
// architectural decisions? You bet!
func getTransaction(ctx context.Context) *sqlx.Tx {
	return ctx.Value(api.TransactionContextKey).(*sqlx.Tx)
}

func New(url, database, username, password string) (*Server, error) {
	connect := fmt.Sprintf("postgres://%s:%s@%s/%s?sslmode=disable", username, password, url, database)
	db, err := sqlx.Connect("postgres", connect)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	var s Server
	s.DB = db

	prometheus.MustRegister(sqlstats.NewStatsCollector("queue", db))

	return &s, nil
}

func (s *Server) SiteAdmin(ctx context.Context, email string) (bool, error) {
	var n int
	err := s.DB.GetContext(ctx, &n,
		"SELECT COUNT(*) FROM site_admins WHERE email=$1",
		email,
	)
	return n > 0, err
}
