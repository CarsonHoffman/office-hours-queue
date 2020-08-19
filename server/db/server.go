package db

import (
	"context"
	"fmt"

	"github.com/CarsonHoffman/office-hours-queue/server/api"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"go.uber.org/zap"
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

func (s *Server) MetricsReport(logger *zap.SugaredLogger) error {
	type metricsQueue struct {
		Queue    string        `json:"queue_id"`
		Course   string        `json:"course_id"`
		Type     api.QueueType `json:"queue_type"`
		Students int           `json:"num_students"`
	}

	var queues []metricsQueue

	rows, err := s.DB.Query(`SELECT q.id, c.id, COUNT(e.id) FROM queues q LEFT JOIN queue_entries e ON e.queue=q.id AND NOT e.removed
							 LEFT JOIN courses c ON c.id=q.course WHERE q.active AND q.type='ordered' GROUP BY q.id, c.id`)

	if err != nil {
		return fmt.Errorf("failed to fetch ordered queues: %w", err)
	}

	for rows.Next() {
		var q metricsQueue
		q.Type = api.Ordered
		err = rows.Scan(&q.Queue, &q.Course, &q.Students)
		if err != nil {
			return fmt.Errorf("failed to scan into metrics queue: %w", err)
		}

		queues = append(queues, q)
	}

	rows, err = s.DB.Query(`SELECT q.id, c.id, COUNT(a.id) FROM queues q LEFT JOIN appointment_slots a ON a.queue=q.id
							AND a.student_email IS NOT NULL AND a.scheduled_time >= NOW()
							LEFT JOIN courses c ON c.id=q.course WHERE q.active AND q.type='appointments' GROUP BY q.id, c.id`)

	for rows.Next() {
		var q metricsQueue
		q.Type = api.Appointments
		err = rows.Scan(&q.Queue, &q.Course, &q.Students)
		if err != nil {
			return fmt.Errorf("failed to scan into metrics queue: %w", err)
		}

		queues = append(queues, q)
	}

	if len(queues) > 0 {
		logger.Infow("queue students report", "queues", queues)
	}

	return nil
}
