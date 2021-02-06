package db

import (
	"context"
	"fmt"

	"github.com/CarsonHoffman/office-hours-queue/server/api"
	"github.com/lib/pq"
	"github.com/segmentio/ksuid"
)

func (s *Server) GetCourses(ctx context.Context) ([]*api.Course, error) {
	tx := getTransaction(ctx)
	courses := make([]*api.Course, 0)
	err := tx.SelectContext(ctx, &courses,
		"SELECT id, short_name, full_name FROM courses ORDER BY id",
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get courses: %w", err)
	}

	qStmt, err := tx.Preparex("SELECT id, course, type, name, location, map, active FROM queues WHERE active AND course=$1 ORDER BY id")
	if err != nil {
		return nil, fmt.Errorf("failed to set up queues statement: %w", err)
	}
	defer qStmt.Close()

	for _, course := range courses {
		course.Queues = make([]*api.Queue, 0)
		err = qStmt.SelectContext(ctx, &course.Queues, course.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get queues for course %s: %w", course.ID, err)
		}
	}

	return courses, nil
}

func (s *Server) GetCourse(ctx context.Context, id ksuid.KSUID) (*api.Course, error) {
	tx := getTransaction(ctx)
	var course api.Course
	err := tx.GetContext(ctx, &course,
		"SELECT id, short_name, full_name FROM courses WHERE id=$1",
		id,
	)
	return &course, err
}

func (s *Server) GetAdminCourses(ctx context.Context, email string) ([]string, error) {
	tx := getTransaction(ctx)
	var n int
	err := tx.GetContext(ctx, &n,
		"SELECT COUNT(*) FROM site_admins WHERE email=$1",
		email,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to check site admin status: %w", err)
	}

	courses := make([]string, 0)
	// Check if user is site admin; if so, they are admin for all courses
	if n > 0 {
		err = tx.SelectContext(ctx, &courses,
			"SELECT id FROM courses",
		)
		return courses, err
	}

	err = tx.SelectContext(ctx, &courses,
		"SELECT course FROM course_admins WHERE email=$1",
		email,
	)
	return courses, err
}

func (s *Server) GetQueues(ctx context.Context, course ksuid.KSUID) ([]*api.Queue, error) {
	tx := getTransaction(ctx)
	queues := make([]*api.Queue, 0)
	err := tx.SelectContext(ctx, &queues,
		"SELECT id, course, type, name, location, map, active FROM queues WHERE course=$1 AND active ORDER BY id",
		course,
	)
	return queues, err
}

func (s *Server) CourseAdmin(ctx context.Context, course ksuid.KSUID, email string) (bool, error) {
	tx := getTransaction(ctx)
	var n int
	err := tx.GetContext(ctx, &n,
		"SELECT COUNT(*) FROM (SELECT email FROM site_admins UNION SELECT email FROM course_admins WHERE course=$1) AS admins WHERE email=$2",
		course, email,
	)
	return n > 0, err
}

func (s *Server) AddCourse(ctx context.Context, shortName, fullName string) (*api.Course, error) {
	tx := getTransaction(ctx)
	id := ksuid.New()
	var course api.Course
	err := tx.GetContext(ctx, &course,
		"INSERT INTO courses (id, short_name, full_name) VALUES ($1, $2, $3) RETURNING id, short_name, full_name",
		id, shortName, fullName,
	)
	return &course, err
}

func (s *Server) UpdateCourse(ctx context.Context, course ksuid.KSUID, shortName, fullName string) error {
	tx := getTransaction(ctx)
	_, err := tx.ExecContext(ctx,
		"UPDATE courses SET short_name=$1, full_name=$2 WHERE id=$3",
		shortName, fullName, course,
	)
	return err
}

func (s *Server) DeleteCourse(ctx context.Context, course ksuid.KSUID) error {
	tx := getTransaction(ctx)
	_, err := tx.ExecContext(ctx,
		"DELETE FROM courses WHERE id=$1",
		course,
	)
	return err
}

func (s *Server) AddQueue(ctx context.Context, course ksuid.KSUID, queue *api.Queue) (*api.Queue, error) {
	tx := getTransaction(ctx)
	id := ksuid.New()
	var newQueue api.Queue
	err := tx.GetContext(ctx, &newQueue,
		"INSERT INTO queues (id, course, type, name, location, map, active) VALUES ($1, $2, $3, $4, $5, $6, $7) RETURNING id, course, type, name, location, map, active",
		id, course, queue.Type, queue.Name, queue.Location, queue.Map, true,
	)
	return &newQueue, err
}

func (s *Server) GetCourseAdmins(ctx context.Context, course ksuid.KSUID) ([]string, error) {
	tx := getTransaction(ctx)
	admins := make([]string, 0)
	err := tx.SelectContext(ctx, &admins, "SELECT email FROM course_admins WHERE course=$1", course)
	return admins, err
}

func (s *Server) AddCourseAdmins(ctx context.Context, course ksuid.KSUID, admins []string, overwrite bool) error {
	tx := getTransaction(ctx)

	if overwrite {
		_, err := tx.Exec("DELETE FROM course_admins WHERE course=$1", course)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to delete existing admins: %w", err)
		}
	}

	insert, err := tx.Prepare(pq.CopyIn("course_admins", "course", "email"))
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer insert.Close()

	for _, email := range admins {
		_, err = insert.Exec(course, email)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert %s into course %s admins: %w", email, course, err)
		}
	}

	_, err = insert.Exec()
	return err
}

func (s *Server) RemoveCourseAdmins(ctx context.Context, course ksuid.KSUID, admins []string) error {
	tx := getTransaction(ctx)

	for _, email := range admins {
		_, err := tx.Exec("DELETE FROM course_admins WHERE course=$1 AND email=$2", course, email)
		if err != nil {
			return fmt.Errorf("failed to delete %s from course %s admins: %w", email, course, err)
		}
	}

	return nil
}
