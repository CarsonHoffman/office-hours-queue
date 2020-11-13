package db

import (
	"context"
	"fmt"

	"github.com/dlmiddlecote/sqlstats"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/prometheus/client_golang/prometheus"
)

type Server struct {
	DB *sqlx.DB
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
