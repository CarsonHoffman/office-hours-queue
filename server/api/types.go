package api

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/segmentio/ksuid"
)

type Course struct {
	ID        ksuid.KSUID `json:"id" db:"id"`
	ShortName string      `json:"short_name" db:"short_name"`
	FullName  string      `json:"full_name" db:"full_name"`
	Queues    []*Queue    `json:"queues"`
}

type QueueType string

const (
	Ordered      QueueType = "ordered"
	Appointments           = "appointments"
)

type Queue struct {
	ID       ksuid.KSUID `json:"id" db:"id"`
	Course   ksuid.KSUID `json:"course" db:"course"`
	Type     QueueType   `json:"type" db:"type"`
	Name     string      `json:"name" db:"name"`
	Location string      `json:"location" db:"location"`
	Map      string      `json:"map" db:"map"`
	Active   bool        `json:"active" db:"active"`
}

type QueueConfiguration struct {
	ID                  ksuid.KSUID `json:"id" db:"id"`
	PreventUnregistered bool        `json:"prevent_unregistered" db:"prevent_unregistered"`
	PreventGroups       bool        `json:"prevent_groups" db:"prevent_groups"`
	PreventGroupsBoost  bool        `json:"prevent_groups_boost" db:"prevent_groups_boost"`
	PrioritizeNew       bool        `json:"prioritize_new" db:"prioritize_new"`
}

type Announcement struct {
	ID      ksuid.KSUID `json:"id" db:"id"`
	Queue   ksuid.KSUID `json:"queue" db:"queue"`
	Content string      `json:"content" db:"content"`
}

type QueueEntry struct {
	ID          ksuid.KSUID    `json:"id" db:"id"`
	Queue       ksuid.KSUID    `json:"queue" db:"queue"`
	Email       string         `json:"email,omitempty" db:"email"`
	Name        string         `json:"name,omitempty" db:"name"`
	Description string         `json:"description,omitempty" db:"description"`
	Location    string         `json:"location,omitempty" db:"location"`
	MapX        float32        `json:"map_x,omitempty" db:"map_x"`
	MapY        float32        `json:"map_y,omitempty" db:"map_y"`
	Priority    int            `json:"priority" db:"priority"`
	Pinned      bool           `json:"pinned,omitempty" db:"pinned"`
	Removed     bool           `json:"-" db:"removed"`
	RemovedBy   sql.NullString `json:"-" db:"removed_by"`
	RemovedAt   sql.NullTime   `json:"-" db:"removed_at"`
}

func (q *QueueEntry) MarshalJSON() ([]byte, error) {
	type QueueEntryWithTimestamp QueueEntry
	return json.Marshal(struct {
		IDTimestamp string `json:"id_timestamp"`
		*QueueEntryWithTimestamp
	}{
		IDTimestamp:             q.ID.Time().Format(time.RFC3339),
		QueueEntryWithTimestamp: (*QueueEntryWithTimestamp)(q),
	})
}

// Anonymized returns a version of this queue entry suitable for
// consumption by other users.
func (q *QueueEntry) Anonymized() *QueueEntry {
	return &QueueEntry{
		ID:       q.ID,
		Queue:    q.Queue,
		Priority: q.Priority,
		Pinned:   q.Pinned,
	}
}

type RemovedQueueEntry struct {
	ID          ksuid.KSUID `json:"id" db:"id"`
	Queue       ksuid.KSUID `json:"queue" db:"queue"`
	Email       string      `json:"email,omitempty" db:"email"`
	Name        string      `json:"name,omitempty" db:"name"`
	Description string      `json:"description,omitempty" db:"description"`
	Location    string      `json:"location,omitempty" db:"location"`
	MapX        float32     `json:"map_x,omitempty" db:"map_x"`
	MapY        float32     `json:"map_y,omitempty" db:"map_y"`
	Priority    int         `json:"priority" db:"priority"`
	Pinned      bool        `json:"pinned,omitempty" db:"pinned"`
	Removed     bool        `json:"-" db:"removed"`
	RemovedBy   string      `json:"removed_by,omitempty" db:"removed_by"`
	RemovedAt   time.Time   `json:"removed_at" db:"removed_at"`
}

func (q *RemovedQueueEntry) MarshalJSON() ([]byte, error) {
	type QueueEntryWithTimestamp RemovedQueueEntry
	q.RemovedAt = q.RemovedAt.In(time.Local)
	return json.Marshal(struct {
		IDTimestamp string `json:"id_timestamp"`
		*QueueEntryWithTimestamp
	}{
		IDTimestamp:             q.ID.Time().Format(time.RFC3339),
		QueueEntryWithTimestamp: (*QueueEntryWithTimestamp)(q),
	})
}

// Anonymized returns a version of this queue entry suitable for
// consumption by other users.
func (q *RemovedQueueEntry) Anonymized() *RemovedQueueEntry {
	return &RemovedQueueEntry{
		ID:       q.ID,
		Queue:    q.Queue,
		Priority: q.Priority,
	}
}

type Message struct {
	ID       ksuid.KSUID `json:"id" db:"id"`
	Queue    ksuid.KSUID `json:"queue" db:"queue"`
	Content  string      `json:"content" db:"content"`
	Sender   string      `json:"sender" db:"sender"`
	Receiver string      `json:"receiver" db:"receiver"`
}

type AppointmentSchedule struct {
	Queue    ksuid.KSUID  `json:"queue" db:"queue"`
	Day      time.Weekday `json:"day" db:"day"`
	Duration int          `json:"duration" db:"duration"`
	Padding  int          `json:"padding" db:"padding"`
	Schedule string       `json:"schedule" db:"schedule"`
}

type AppointmentSlot struct {
	ID            ksuid.KSUID `json:"id" db:"id"`
	Queue         ksuid.KSUID `json:"queue" db:"queue"`
	StaffEmail    *string     `json:"staff_email,omitempty" db:"staff_email"`
	StudentEmail  *string     `json:"student_email,omitempty" db:"student_email"`
	ScheduledTime time.Time   `json:"scheduled_time" db:"scheduled_time"`
	Timeslot      int         `json:"timeslot" db:"timeslot"`
	Duration      int         `json:"duration" db:"duration"`
	Name          *string     `json:"name,omitempty" db:"name"`
	Location      *string     `json:"location,omitempty" db:"location"`
	Description   *string     `json:"description,omitempty" db:"description"`
	MapX          *float32    `json:"map_x,omitempty" db:"map_x"`
	MapY          *float32    `json:"map_y,omitempty" db:"map_y"`
}

func (a *AppointmentSlot) MarshalJSON() ([]byte, error) {
	type AppointmentSlotWithTimestamp AppointmentSlot
	a.ScheduledTime = a.ScheduledTime.In(time.Local)
	return json.Marshal(struct {
		IDTimestamp string `json:"id_timestamp"`
		*AppointmentSlotWithTimestamp
	}{
		IDTimestamp:                  a.ID.Time().Format(time.RFC3339),
		AppointmentSlotWithTimestamp: (*AppointmentSlotWithTimestamp)(a),
	})
}

func (a *AppointmentSlot) Anonymized() *AppointmentSlot {
	return &AppointmentSlot{
		ID:            a.ID,
		Queue:         a.Queue,
		ScheduledTime: a.ScheduledTime,
		Timeslot:      a.Timeslot,
		Duration:      a.Duration,
	}
}

func (a *AppointmentSlot) NoStaffEmail() *AppointmentSlot {
	newAppointment := *a
	newAppointment.StaffEmail = nil
	return &newAppointment
}
