package db

import (
	"context"
	"fmt"
	"time"

	"github.com/CarsonHoffman/office-hours-queue/server/api"
	"github.com/lib/pq"
	"github.com/segmentio/ksuid"
)

func (s *Server) GetQueue(ctx context.Context, queue ksuid.KSUID) (*api.Queue, error) {
	tx := getTransaction(ctx)
	var q api.Queue
	err := tx.GetContext(ctx, &q,
		"SELECT id, course, type, name, location, map, active FROM queues q WHERE active AND id=$1",
		queue,
	)
	return &q, err
}

func (s *Server) UpdateQueue(ctx context.Context, queue ksuid.KSUID, values *api.Queue) error {
	tx := getTransaction(ctx)
	_, err := tx.ExecContext(ctx,
		"UPDATE queues SET name=$1, location=$2 WHERE id=$3",
		values.Name, values.Location, queue,
	)
	return err
}

func (s *Server) RemoveQueue(ctx context.Context, queue ksuid.KSUID) error {
	tx := getTransaction(ctx)
	_, err := tx.ExecContext(ctx,
		"DELETE FROM queues WHERE id=$1",
		queue,
	)
	return err
}

func (s *Server) GetCurrentDaySchedule(ctx context.Context, queue ksuid.KSUID) (string, error) {
	tx := getTransaction(ctx)
	var schedule string
	day := time.Now().Weekday()
	err := tx.GetContext(ctx, &schedule,
		"SELECT schedule FROM schedules WHERE queue=$1 AND day=$2",
		queue, day,
	)
	return schedule, err
}

func (s *Server) GetQueueEntry(ctx context.Context, entry ksuid.KSUID, allowRemoved bool) (*api.QueueEntry, error) {
	tx := getTransaction(ctx)
	var e api.QueueEntry
	err := tx.GetContext(ctx, &e,
		"SELECT * FROM queue_entries WHERE id=$1 AND ($2 OR active IS NOT NULL)",
		entry, allowRemoved,
	)
	return &e, err
}

func (s *Server) GetQueueEntries(ctx context.Context, queue ksuid.KSUID, admin bool) ([]*api.QueueEntry, error) {
	tx := getTransaction(ctx)
	query := "SELECT id, queue, priority, pinned FROM queue_entries WHERE queue=$1 AND active IS NOT NULL ORDER BY pinned DESC, priority DESC, id"
	if admin {
		query = "SELECT * FROM queue_entries WHERE queue=$1 AND active IS NOT NULL ORDER BY pinned DESC, priority DESC, id"
	}

	entries := make([]*api.QueueEntry, 0)
	err := tx.SelectContext(ctx, &entries, query, queue)
	return entries, err
}

func (s *Server) GetActiveQueueEntriesForUser(ctx context.Context, queue ksuid.KSUID, email string) ([]*api.QueueEntry, error) {
	tx := getTransaction(ctx)
	entries := make([]*api.QueueEntry, 0)
	err := tx.SelectContext(ctx, &entries,
		"SELECT * FROM queue_entries WHERE queue=$1 AND email=$2 AND active IS NOT NULL",
		queue, email,
	)
	return entries, err
}

func (s *Server) GetQueueConfiguration(ctx context.Context, queue ksuid.KSUID) (*api.QueueConfiguration, error) {
	tx := getTransaction(ctx)
	var config api.QueueConfiguration
	err := tx.GetContext(ctx, &config,
		"SELECT id, enable_location_field, prevent_unregistered, prevent_groups, prevent_groups_boost, prioritize_new, virtual FROM queues WHERE id=$1",
		queue,
	)
	return &config, err
}

func (s *Server) UpdateQueueConfiguration(ctx context.Context, queue ksuid.KSUID, config *api.QueueConfiguration) error {
	tx := getTransaction(ctx)
	_, err := tx.ExecContext(ctx,
		"UPDATE queues SET enable_location_field=$1, prevent_unregistered=$2, prevent_groups=$3, prevent_groups_boost=$4, prioritize_new=$5, virtual=$6 WHERE id=$7",
		config.EnableLocationField, config.PreventUnregistered, config.PreventGroups, config.PreventGroupsBoost, config.PrioritizeNew, config.Virtual, queue,
	)
	return err
}

func (s *Server) GetQueueRoster(ctx context.Context, queue ksuid.KSUID) ([]string, error) {
	tx := getTransaction(ctx)
	roster := make([]string, 0)
	err := tx.SelectContext(ctx, &roster, "SELECT email FROM roster WHERE queue=$1 ORDER BY email", queue)
	return roster, err
}

func (s *Server) GetQueueGroups(ctx context.Context, queue ksuid.KSUID) ([][]string, error) {
	tx := getTransaction(ctx)
	var groupIDs []string
	groups := make([][]string, 0)

	err := tx.SelectContext(ctx, &groupIDs,
		"SELECT DISTINCT group_id FROM groups WHERE queue=$1",
		queue,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch group IDs: %w", err)
	}

	for _, id := range groupIDs {
		var group []string
		err = tx.SelectContext(ctx, &group,
			"SELECT email FROM groups WHERE queue=$1 AND group_id=$2",
			queue, id,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to get members in group %s: %w", id, err)
		}
		groups = append(groups, group)
	}

	return groups, nil
}

func (s *Server) UpdateQueueGroups(ctx context.Context, queue ksuid.KSUID, groups [][]string) error {
	tx := getTransaction(ctx)

	_, err := tx.Exec("DELETE FROM groups WHERE queue=$1", queue)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete existing groups: %w", err)
	}

	insert, err := tx.Prepare(pq.CopyIn("groups", "queue", "group_id", "email"))
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer insert.Close()

	for _, group := range groups {
		groupID := ksuid.New()
		for _, student := range group {
			_, err = insert.Exec(queue, groupID, student)
			if err != nil {
				return fmt.Errorf("failed to insert student %s into group %s: %w", student, groupID, err)
			}
		}
	}

	_, err = insert.Exec()
	return err
}

func (s *Server) UserInQueueRoster(ctx context.Context, queue ksuid.KSUID, email string) (bool, error) {
	tx := getTransaction(ctx)
	var n int
	err := tx.GetContext(ctx, &n,
		"SELECT COUNT(*) FROM roster WHERE queue=$1 AND email=$2",
		queue, email,
	)
	return n > 0, err
}

func (s *Server) UpdateQueueRoster(ctx context.Context, queue ksuid.KSUID, students []string) error {
	tx := getTransaction(ctx)

	_, err := tx.Exec("DELETE FROM roster WHERE queue=$1", queue)
	if err != nil {
		return fmt.Errorf("failed to delete existing roster: %w", err)
	}

	insert, err := tx.Prepare(pq.CopyIn("roster", "queue", "email"))
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer insert.Close()

	for _, student := range students {
		_, err = insert.Exec(queue, student)
		if err != nil {
			return fmt.Errorf("failed to insert student %s into roster: %w", student, err)
		}
	}

	_, err = insert.Exec()
	return err
}

func (s *Server) TeammateInQueue(ctx context.Context, queue ksuid.KSUID, email string) (bool, error) {
	tx := getTransaction(ctx)
	var n int
	err := tx.GetContext(ctx, &n,
		"SELECT COUNT(*) FROM queue_entries e JOIN teammates t ON e.email=t.teammate WHERE t.queue=$1 AND t.email=$2 AND e.queue=$3 AND e.active IS NOT NULL",
		queue, email, queue,
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

func (s *Server) GetEntryPriority(ctx context.Context, queue ksuid.KSUID, email string) (int, error) {
	tx := getTransaction(ctx)
	config, err := s.GetQueueConfiguration(ctx, queue)
	if err != nil {
		return 0, fmt.Errorf("failed to get queue configuration: %w", err)
	}

	if !config.PrioritizeNew {
		return 0, nil
	}

	start, _ := api.WeekdayBounds(int(time.Now().Local().Weekday()))
	var payload [16]byte
	firstIDOfDay, err := ksuid.FromParts(start, payload[:])
	if err != nil {
		return 0, fmt.Errorf("failed to generate first KSUID of day: %w", err)
	}

	var personalEntries int
	err = tx.GetContext(ctx, &personalEntries,
		"SELECT COUNT(*) FROM queue_entries WHERE email=$1 AND queue=$2 AND id>=$3 AND removed_by!=email AND helped",
		email, queue, firstIDOfDay,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to get number of personal queue entries today: %w", err)
	}

	if personalEntries > 0 {
		return 0, nil
	}

	if !config.PreventGroupsBoost {
		return 1, nil
	}

	var groupEntries int
	err = tx.GetContext(ctx, &groupEntries,
		"SELECT COUNT(*) FROM queue_entries e JOIN teammates t ON e.email=t.teammate AND e.queue=t.queue WHERE t.email=$1 AND e.queue=$2 AND e.id>=$3 AND e.removed_by!=e.email AND helped",
		email, queue, firstIDOfDay,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to get number of teammate queue entries today: %w", err)
	}

	if groupEntries > 0 {
		return 0, nil
	}
	return 1, nil
}

func (s *Server) AddQueueEntry(ctx context.Context, e *api.QueueEntry) (*api.QueueEntry, error) {
	tx := getTransaction(ctx)
	var newEntry api.QueueEntry
	id := ksuid.New()
	err := tx.GetContext(ctx, &newEntry,
		"INSERT INTO queue_entries (id, queue, email, name, location, map_x, map_y, description, priority) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING *",
		id, e.Queue, e.Email, e.Name, e.Location, e.MapX, e.MapY, e.Description, e.Priority,
	)
	return &newEntry, err
}

func (s *Server) UpdateQueueEntry(ctx context.Context, entry ksuid.KSUID, e *api.QueueEntry) error {
	tx := getTransaction(ctx)
	_, err := tx.ExecContext(ctx,
		"UPDATE queue_entries SET name=$1, location=$2, description=$3, map_x=$4, map_y=$5 WHERE id=$6 AND active IS NOT NULL",
		e.Name, e.Location, e.Description, e.MapX, e.MapY, entry,
	)
	return err
}

func (s *Server) CanRemoveQueueEntry(ctx context.Context, queue ksuid.KSUID, entry ksuid.KSUID, email string) (bool, error) {
	tx := getTransaction(ctx)
	q, err := s.GetQueue(ctx, queue)
	if err != nil {
		return false, fmt.Errorf("failed to get queue: %w", err)
	}

	admin, err := s.CourseAdmin(ctx, q.Course, email)
	if err != nil {
		return false, fmt.Errorf("failed to determine admin status: %w", err)
	}

	if admin {
		return true, nil
	}

	var n int
	err = tx.GetContext(ctx, &n,
		"SELECT COUNT(*) FROM queue_entries WHERE id=$1 AND email=$2",
		entry, email,
	)
	return n > 0, err
}

func (s *Server) RemoveQueueEntry(ctx context.Context, entry ksuid.KSUID, remover string) (*api.RemovedQueueEntry, error) {
	tx := getTransaction(ctx)
	var e api.RemovedQueueEntry
	err := tx.GetContext(ctx, &e,
		"UPDATE queue_entries SET pinned=FALSE, active=NULL, removed_at=NOW(), removed_by=$1, helped=TRUE WHERE active IS NOT NULL AND id=$2 RETURNING *",
		remover, entry,
	)
	return &e, err
}

func (s *Server) PinQueueEntry(ctx context.Context, entry ksuid.KSUID) error {
	tx := getTransaction(ctx)
	_, err := tx.ExecContext(ctx,
		"UPDATE queue_entries SET active=TRUE, removed_at=NULL, removed_by=NULL, helped=FALSE, pinned=TRUE WHERE id=$1",
		entry,
	)
	return err
}

func (s *Server) SetHelpedStatus(ctx context.Context, entry ksuid.KSUID, helped bool) error {
	tx := getTransaction(ctx)
	_, err := tx.ExecContext(ctx,
		"UPDATE queue_entries SET helped=$1 WHERE id=$2",
		helped, entry,
	)
	return err
}

func (s *Server) ClearQueueEntries(ctx context.Context, queue ksuid.KSUID, remover string) error {
	tx := getTransaction(ctx)
	_, err := tx.ExecContext(ctx,
		"UPDATE queue_entries SET active=NULL, removed_at=NOW(), removed_by=$1, pinned=FALSE, helped=FALSE WHERE active IS NOT NULL AND queue=$2",
		remover, queue,
	)
	return err
}

func (s *Server) GetQueueStack(ctx context.Context, queue ksuid.KSUID, limit int) ([]*api.RemovedQueueEntry, error) {
	tx := getTransaction(ctx)
	entries := make([]*api.RemovedQueueEntry, 0)
	err := tx.SelectContext(ctx, &entries,
		"SELECT * FROM queue_entries WHERE queue=$1 AND active IS NULL ORDER BY removed_at DESC, id DESC LIMIT $2",
		queue, limit,
	)
	return entries, err
}

func (s *Server) GetQueueAnnouncements(ctx context.Context, queue ksuid.KSUID) ([]*api.Announcement, error) {
	tx := getTransaction(ctx)
	announcements := make([]*api.Announcement, 0)
	err := tx.SelectContext(ctx, &announcements,
		"SELECT id, queue, content FROM announcements WHERE queue=$1 ORDER BY id",
		queue,
	)
	return announcements, err
}

func (s *Server) AddQueueAnnouncement(ctx context.Context, queue ksuid.KSUID, announcement *api.Announcement) (*api.Announcement, error) {
	tx := getTransaction(ctx)
	var newAnnouncement api.Announcement
	id := ksuid.New()
	err := tx.GetContext(ctx, &newAnnouncement,
		"INSERT INTO announcements (id, queue, content) VALUES ($1, $2, $3) RETURNING id, queue, content",
		id, announcement.Queue, announcement.Content,
	)
	return &newAnnouncement, err
}

func (s *Server) RemoveQueueAnnouncement(ctx context.Context, announcement ksuid.KSUID) error {
	tx := getTransaction(ctx)
	_, err := tx.ExecContext(ctx,
		"DELETE FROM announcements WHERE id=$1",
		announcement,
	)
	return err
}

func (s *Server) GetQueueSchedule(ctx context.Context, queue ksuid.KSUID) ([]string, error) {
	tx := getTransaction(ctx)
	schedules := make([]string, 0)
	err := tx.SelectContext(ctx, &schedules,
		"SELECT schedule FROM schedules WHERE queue=$1 ORDER BY day",
		queue,
	)
	return schedules, err
}

func (s *Server) AddQueueSchedule(ctx context.Context, queue ksuid.KSUID, day int, schedule string) error {
	tx := getTransaction(ctx)
	_, err := tx.ExecContext(ctx,
		"INSERT INTO schedules (queue, day, schedule) VALUES ($1, $2, $3)",
		queue, day, schedule,
	)
	return err
}

func (s *Server) UpdateQueueSchedule(ctx context.Context, queue ksuid.KSUID, schedules []string) error {
	tx := getTransaction(ctx)
	for i, schedule := range schedules {
		_, err := tx.ExecContext(ctx,
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
	tx := getTransaction(ctx)
	id := ksuid.New()
	var message api.Message
	err := tx.GetContext(ctx, &message,
		"INSERT INTO messages (id, queue, content, sender, receiver) VALUES ($1, $2, $3, $4, $5) RETURNING id, queue, content, sender, receiver",
		id, queue, content, sender, receiver,
	)
	return &message, err
}

func (s *Server) ViewMessage(ctx context.Context, queue ksuid.KSUID, receiver string) (*api.Message, error) {
	tx := getTransaction(ctx)
	var message api.Message
	err := tx.GetContext(ctx, &message,
		"DELETE FROM messages WHERE id IN (SELECT id FROM messages WHERE queue=$1 AND receiver=$2 ORDER BY id LIMIT 1) RETURNING id, queue, content, sender, receiver",
		queue, receiver,
	)
	return &message, err
}

func (s *Server) QueueStats() ([]api.QueueStats, error) {
	var queues []api.QueueStats

	rows, err := s.DB.Query(`SELECT q.id, c.id, COUNT(e.id) FROM queues q LEFT JOIN queue_entries e ON e.queue=q.id AND e.active IS NOT NULL
							 LEFT JOIN courses c ON c.id=q.course WHERE q.active AND q.type='ordered' GROUP BY q.id, c.id`)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch ordered queues: %w", err)
	}

	for rows.Next() {
		var q api.QueueStats
		q.Type = api.Ordered
		err = rows.Scan(&q.Queue, &q.Course, &q.Students)
		if err != nil {
			return nil, fmt.Errorf("failed to scan into metrics queue: %w", err)
		}

		queues = append(queues, q)
	}

	rows, err = s.DB.Query(`SELECT q.id, c.id, COUNT(a.id) FROM queues q LEFT JOIN appointment_slots a ON a.queue=q.id
							AND a.student_email IS NOT NULL AND a.scheduled_time >= NOW()
							LEFT JOIN courses c ON c.id=q.course WHERE q.active AND q.type='appointments' GROUP BY q.id, c.id`)

	if err != nil {
		return nil, fmt.Errorf("failed to fetch appointments queues: %w", err)
	}

	for rows.Next() {
		var q api.QueueStats
		q.Type = api.Appointments
		err = rows.Scan(&q.Queue, &q.Course, &q.Students)
		if err != nil {
			return nil, fmt.Errorf("failed to scan into metrics queue: %w", err)
		}

		queues = append(queues, q)
	}

	return queues, nil
}
