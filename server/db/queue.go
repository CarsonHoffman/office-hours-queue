package db

import (
	"context"
	"fmt"
	"time"

	"github.com/CarsonHoffman/office-hours-queue/server/api"
	"github.com/segmentio/ksuid"
)

func (s *Server) QueueAdmin(ctx context.Context, queue ksuid.KSUID, email string) (bool, error) {
	var n int
	err := s.DB.GetContext(ctx, &n,
		"SELECT COUNT(*) FROM (SELECT email FROM site_admins UNION SELECT email FROM queues q JOIN course_admins c ON q.course=c.course WHERE q.id=$1) AS admins WHERE email=$2",
		queue, email,
	)
	return n > 0, err
}

func (s *Server) GetQueue(ctx context.Context, queue ksuid.KSUID) (*api.Queue, error) {
	var q api.Queue
	err := s.DB.GetContext(ctx, &q,
		"SELECT id, course, type, name, location, map, active FROM queues q WHERE active AND id=$1",
		queue,
	)
	return &q, err
}

func (s *Server) GetCurrentDaySchedule(ctx context.Context, queue ksuid.KSUID) (string, error) {
	var schedule string
	day := time.Now().Weekday()
	err := s.DB.GetContext(ctx, &schedule,
		"SELECT schedule FROM schedules WHERE queue=$1 AND day=$2",
		queue, day,
	)
	return schedule, err
}

func (s *Server) GetQueueEntry(ctx context.Context, entry ksuid.KSUID) (*api.QueueEntry, error) {
	var e api.QueueEntry
	err := s.DB.GetContext(ctx, &e,
		"SELECT * FROM queue_entries WHERE id=$1 AND NOT removed",
		entry,
	)
	return &e, err
}

func (s *Server) GetQueueEntries(ctx context.Context, queue ksuid.KSUID, admin bool) ([]*api.QueueEntry, error) {
	query := "SELECT id, queue, priority FROM queue_entries WHERE queue=$1 AND NOT removed"
	if admin {
		query = "SELECT * FROM queue_entries WHERE queue=$1 AND NOT removed"
	}

	entries := make([]*api.QueueEntry, 0)
	err := s.DB.SelectContext(ctx, &entries, query, queue)
	return entries, err
}

func (s *Server) GetActiveQueueEntriesForUser(ctx context.Context, queue ksuid.KSUID, email string) ([]*api.QueueEntry, error) {
	entries := make([]*api.QueueEntry, 0)
	err := s.DB.SelectContext(ctx, &entries,
		"SELECT * FROM queue_entries WHERE queue=$1 AND email=$2 AND NOT removed",
		queue, email,
	)
	return entries, err
}

func (s *Server) GetQueueConfiguration(ctx context.Context, queue ksuid.KSUID) (*api.QueueConfiguration, error) {
	var config api.QueueConfiguration
	err := s.DB.GetContext(ctx, &config,
		"SELECT id, prevent_unregistered, prevent_groups, prevent_groups_boost, prioritize_new FROM queues WHERE id=$1",
		queue,
	)
	return &config, err
}

func (s *Server) UpdateQueueConfiguration(ctx context.Context, queue ksuid.KSUID, config *api.QueueConfiguration) error {
	_, err := s.DB.ExecContext(ctx,
		"UPDATE queues SET prevent_unregistered=$1, prevent_groups=$2, prevent_groups_boost=$3, prioritize_new=$4 WHERE id=$5",
		config.PreventUnregistered, config.PreventGroups, config.PreventGroupsBoost, config.PrioritizeNew, queue,
	)
	return err
}

func (s *Server) UserInQueueRoster(ctx context.Context, queue ksuid.KSUID, email string) (bool, error) {
	var n int
	err := s.DB.GetContext(ctx, &n,
		"SELECT COUNT(*) FROM roster WHERE queue=$1 AND email=$2",
		queue, email,
	)
	return n > 0, err
}

func (s *Server) TeammateInQueue(ctx context.Context, queue ksuid.KSUID, email string) (bool, error) {
	var n int
	err := s.DB.GetContext(ctx, &n,
		"SELECT COUNT(*) FROM groups g JOIN queue_entries qe ON g.teammate_email = qe.email WHERE g.queue = $1 AND g.email = $2",
		queue, email,
	)
	return n > 0, err
}

func (s *Server) CanAddEntry(ctx context.Context, queue ksuid.KSUID, email string) (bool, error) {
	q, err := s.GetQueue(ctx, queue)
	if err != nil {
		return false, fmt.Errorf("failed to get queue: %w", err)
	}

	admin, err := s.CourseAdmin(ctx, q.Course, email)
	if err != nil {
		return false, fmt.Errorf("failed to determine admin status in course: %w", err)
	}

	if admin {
		return true, nil
	}

	schedule, err := s.GetCurrentDaySchedule(ctx, queue)
	if err != nil {
		return false, fmt.Errorf("failed to get queue schedule: %w", err)
	}

	halfHour := api.CurrentHalfHour()
	if schedule[halfHour] == 'c' {
		return false, fmt.Errorf("queue closed")
	}

	config, err := s.GetQueueConfiguration(ctx, queue)
	if err != nil {
		return false, fmt.Errorf("failed to fetch configuration for queue: %w", err)
	}

	if config.PreventUnregistered {
		isInRoster, err := s.UserInQueueRoster(ctx, queue, email)
		if err != nil {
			return false, fmt.Errorf("failed to determine roster status in queue: %w", err)
		}

		if !isInRoster {
			return false, fmt.Errorf("user not in roster")
		}
	}

	if config.PreventGroups {
		teammateInQueue, err := s.TeammateInQueue(ctx, queue, email)
		if err != nil {
			return false, fmt.Errorf("failed to determine teammate status in queue: %w", err)
		}

		if teammateInQueue {
			return false, fmt.Errorf("teammate in queue")
		}
	}

	return true, nil
}

func (s *Server) AddQueueEntry(ctx context.Context, e *api.QueueEntry) (*api.QueueEntry, error) {
	var newEntry api.QueueEntry
	id := ksuid.New()
	err := s.DB.GetContext(ctx, &newEntry,
		"INSERT INTO queue_entries (id, queue, email, name, location, map_x, map_y, description, priority) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING *",
		id, e.Queue, e.Email, e.Name, e.Location, e.MapX, e.MapY, e.Description, e.Priority,
	)
	return &newEntry, err
}

func (s *Server) UpdateQueueEntry(ctx context.Context, entry ksuid.KSUID, e *api.QueueEntry) error {
	_, err := s.DB.ExecContext(ctx,
		"UPDATE queue_entries SET name=$1, location=$2, description=$3, map_x=$4, map_y=$5 WHERE id=$6 AND NOT removed",
		e.Name, e.Location, e.Description, e.MapX, e.MapY, entry,
	)
	return err
}

func (s *Server) CanRemoveQueueEntry(ctx context.Context, queue ksuid.KSUID, entry ksuid.KSUID, email string) (bool, error) {
	admin, err := s.QueueAdmin(ctx, queue, email)
	if err != nil {
		return false, fmt.Errorf("failed to determine admin status: %w", err)
	}

	if admin {
		return true, nil
	}

	var n int
	err = s.DB.GetContext(ctx, &n,
		"SELECT COUNT(*) FROM queue_entries WHERE id=$1 AND email=$2",
		entry, email,
	)
	return n > 0, err
}

func (s *Server) RemoveQueueEntry(ctx context.Context, entry ksuid.KSUID, remover string) error {
	_, err := s.DB.ExecContext(ctx,
		"UPDATE queue_entries SET removed=TRUE, removed_at=NOW(), removed_by=$1 WHERE NOT removed AND id=$2",
		remover, entry,
	)
	return err
}

func (s *Server) ClearQueueEntries(ctx context.Context, queue ksuid.KSUID, remover string) error {
	_, err := s.DB.ExecContext(ctx,
		"UPDATE queue_entries SET removed=TRUE, removed_at=NOW(), removed_by=$1 WHERE NOT removed AND queue=$2",
		remover, queue,
	)
	return err
}

func (s *Server) GetQueueStack(ctx context.Context, queue ksuid.KSUID, limit int) ([]*api.RemovedQueueEntry, error) {
	entries := make([]*api.RemovedQueueEntry, 0)
	err := s.DB.SelectContext(ctx, &entries,
		"SELECT * FROM queue_entries WHERE queue=$1 AND removed ORDER BY id DESC LIMIT $2",
		queue, limit,
	)
	return entries, err
}

func (s *Server) GetQueueAnnouncements(ctx context.Context, queue ksuid.KSUID) ([]*api.Announcement, error) {
	announcements := make([]*api.Announcement, 0)
	err := s.DB.SelectContext(ctx, &announcements,
		"SELECT id, queue, content FROM announcements WHERE queue=$1 ORDER BY id",
		queue,
	)
	return announcements, err
}

func (s *Server) AddQueueAnnouncement(ctx context.Context, queue ksuid.KSUID, announcement *api.Announcement) (*api.Announcement, error) {
	var newAnnouncement api.Announcement
	id := ksuid.New()
	err := s.DB.GetContext(ctx, &newAnnouncement,
		"INSERT INTO announcements (id, queue, content) VALUES ($1, $2, $3) RETURNING id, queue, content",
		id, announcement.Queue, announcement.Content,
	)
	return &newAnnouncement, err
}

func (s *Server) RemoveQueueAnnouncement(ctx context.Context, announcement ksuid.KSUID) error {
	_, err := s.DB.ExecContext(ctx,
		"DELETE FROM announcements WHERE id=$1",
		announcement,
	)
	return err
}

func (s *Server) GetQueueSchedule(ctx context.Context, queue ksuid.KSUID) ([]string, error) {
	schedules := make([]string, 0)
	err := s.DB.SelectContext(ctx, &schedules,
		"SELECT schedule FROM schedules WHERE queue=$1 ORDER BY day",
		queue,
	)
	return schedules, err
}

func (s *Server) AddQueueSchedule(ctx context.Context, queue ksuid.KSUID, day int, schedule string) error {
	_, err := s.DB.ExecContext(ctx,
		"INSERT INTO schedules (queue, day, schedule) VALUES ($1, $2, $3)",
		queue, day, schedule,
	)
	return err
}

func (s *Server) UpdateQueueSchedule(ctx context.Context, queue ksuid.KSUID, schedules []string) error {
	for i, schedule := range schedules {
		_, err := s.DB.ExecContext(ctx,
			"UPDATE schedules SET schedule=$1 WHERE queue=$2 AND day=$3",
			schedule, queue, i,
		)
		if err != nil {
			return fmt.Errorf("failed to update schedule for day %d: %w", i, err)
		}
	}

	return nil
}

func (s *Server) SendMessage(ctx context.Context, queue ksuid.KSUID, content, sender, receiver string) (*api.Message, error) {
	id := ksuid.New()
	var message api.Message
	err := s.DB.GetContext(ctx, &message,
		"INSERT INTO messages (id, queue, content, sender, receiver) VALUES ($1, $2, $3, $4, $5) RETURNING id, queue, content, sender, receiver",
		id, queue, content, sender, receiver,
	)
	return &message, err
}
