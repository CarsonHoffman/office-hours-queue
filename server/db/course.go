package db

import (
	"context"
	"fmt"

	"github.com/CarsonHoffman/office-hours-queue/server/api"
	"github.com/segmentio/ksuid"
)

func (s *Server) GetCourses(ctx context.Context) ([]*api.Course, error) {
	courses := make([]*api.Course, 0)
	err := s.DB.SelectContext(ctx, &courses,
		"SELECT id, short_name, full_name FROM courses",
	)
	return courses, err
}

func (s *Server) GetCourse(ctx context.Context, id ksuid.KSUID) (*api.Course, error) {
	var course api.Course
	err := s.DB.GetContext(ctx, &course,
		"SELECT id, short_name, full_name FROM courses WHERE id=$1",
		id,
	)
	return &course, err
}

func (s *Server) GetAdminCourses(ctx context.Context, email string) ([]string, error) {
	var n int
	err := s.DB.GetContext(ctx, &n,
		"SELECT COUNT(*) FROM site_admins WHERE email=$1",
		email,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to check site admin status: %w", err)
	}

	courses := make([]string, 0)
	// Check if user is site admin; if so, they are admin for all courses
	if n > 0 {
		err = s.DB.SelectContext(ctx, &courses,
			"SELECT id FROM courses",
		)
		return courses, nil
	}

	err = s.DB.SelectContext(ctx, &courses,
		"SELECT course FROM course_admins WHERE email=$1",
		email,
	)
	return courses, nil
}

func (s *Server) GetQueues(ctx context.Context, course ksuid.KSUID) ([]*api.Queue, error) {
	queues := make([]*api.Queue, 0)
	err := s.DB.SelectContext(ctx, &queues,
		"SELECT id, course, type, name, location, map, active FROM queues WHERE course=$1 AND active ORDER BY id",
		course,
	)
	return queues, err
}

func (s *Server) CourseAdmin(ctx context.Context, course ksuid.KSUID, email string) (bool, error) {
	var n int
	err := s.DB.GetContext(ctx, &n,
		"SELECT COUNT(*) FROM (SELECT email FROM site_admins UNION SELECT email FROM course_admins WHERE course=$1) AS admins WHERE email=$2",
		course, email,
	)
	return n > 0, err
}

func (s *Server) AddCourse(ctx context.Context, shortName, fullName string) (*api.Course, error) {
	id := ksuid.New()
	var course api.Course
	err := s.DB.GetContext(ctx, &course,
		"INSERT INTO courses (id, short_name, full_name) VALUES ($1, $2, $3) RETURNING id, short_name, full_name",
		id, shortName, fullName,
	)
	return &course, err
}

func (s *Server) AddQueue(ctx context.Context, course ksuid.KSUID, queue *api.Queue) (*api.Queue, error) {
	id := ksuid.New()
	var newQueue api.Queue
	err := s.DB.GetContext(ctx, &newQueue,
		"INSERT INTO queues (id, course, type, name, location, map, active) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, course, type, name, location, map, active",
		id, course, queue.Type, queue.Name, queue.Location, queue.Map, true,
	)
	return &newQueue, err
}
