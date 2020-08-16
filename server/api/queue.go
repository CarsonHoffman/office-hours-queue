package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/segmentio/ksuid"
)

type getQueue interface {
	GetQueue(context.Context, ksuid.KSUID) (*Queue, error)
}

const queueContextKey = "queue"

func (s *Server) QueueIDMiddleware(gq getQueue) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			idString := chi.URLParam(r, "id")
			id, err := ksuid.Parse(idString)
			if err != nil {
				s.logger.Warnw("failed to parse queue id",
					RequestIDContextKey, r.Context().Value(RequestIDContextKey),
					"queue_id", idString,
				)
				s.errorMessage(
					http.StatusNotFound,
					"That queue is hiding from me…make sure it exists!",
					w, r,
				)
				return
			}

			q, err := gq.GetQueue(r.Context(), id)
			if errors.Is(err, sql.ErrNoRows) {
				s.logger.Warnw("failed to get non-existent queue with valid ksuid",
					RequestIDContextKey, r.Context().Value(RequestIDContextKey),
					"queue_id", idString,
				)
				s.errorMessage(
					http.StatusNotFound,
					"That queue is hiding from me…make sure it exists!",
					w, r,
				)
				return
			} else if err != nil {
				s.logger.Errorw("failed to get queue",
					RequestIDContextKey, r.Context().Value(RequestIDContextKey),
					"queue_id", idString,
					"err", err,
				)
				s.internalServerError(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), queueContextKey, q)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

const queueAdminContextKey = "queue_admin"

type queueAdmin interface {
	QueueAdmin(ctx context.Context, course ksuid.KSUID, email string) (bool, error)
}

func (s *Server) CheckQueueAdmin(qa queueAdmin) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			q := r.Context().Value(queueContextKey).(*Queue)
			email, ok := r.Context().Value(emailContextKey).(string)
			if !ok {
				ctx := context.WithValue(r.Context(), queueAdminContextKey, false)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			admin, err := qa.QueueAdmin(r.Context(), q.ID, email)
			if err != nil {
				s.logger.Errorw("failed to check admin status",
					RequestIDContextKey, r.Context().Value(RequestIDContextKey),
					"queue_id", q.ID,
					"email", email,
					"err", err,
				)
				s.internalServerError(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), queueAdminContextKey, admin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (s *Server) EnsureQueueAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		email := r.Context().Value(emailContextKey).(string)
		admin := r.Context().Value(queueAdminContextKey).(bool)
		if !admin {
			s.logger.Warnw("non-admin attempting to access resource requiring queue admin",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"queue_id", q.ID,
				"email", email,
			)
			s.errorMessage(
				http.StatusForbidden,
				"You shouldn't be here. :)",
				w, r,
			)
			return
		}

		s.logger.Infow("entering queue admin context",
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"queue_id", q.ID,
			"email", email,
		)
		next.ServeHTTP(w, r)
	})
}

type getQueueEntry interface {
	GetQueueEntry(ctx context.Context, entry ksuid.KSUID) (*QueueEntry, error)
}

type getQueueEntries interface {
	GetQueueEntries(ctx context.Context, queue ksuid.KSUID, admin bool) ([]*QueueEntry, error)
}

type getActiveQueueEntriesForUser interface {
	GetActiveQueueEntriesForUser(ctx context.Context, queue ksuid.KSUID, email string) ([]*QueueEntry, error)
}

type getQueueAnnouncements interface {
	GetQueueAnnouncements(ctx context.Context, queue ksuid.KSUID) ([]*Announcement, error)
}

type getQueueStack interface {
	GetQueueStack(ctx context.Context, queue ksuid.KSUID, limit int) ([]*RemovedQueueEntry, error)
}

type getCurrentDaySchedule interface {
	GetCurrentDaySchedule(ctx context.Context, queue ksuid.KSUID) (string, error)
}

type getQueueDetails interface {
	getQueueEntry
	getQueueEntries
	getActiveQueueEntriesForUser
	getQueueStack
	getQueueAnnouncements
	getCurrentDaySchedule
}

func (s *Server) GetQueue(gd getQueueDetails) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"queue_id", q.ID,
		)

		admin := r.Context().Value(queueAdminContextKey).(bool)
		// This is a bit of a hack, but we're okay with the zero value
		// of string if the assertion fails, but we don't want it to panic,
		// so we need to do the two-value assertion
		email, _ := r.Context().Value(emailContextKey).(string)

		// This isn't pretty, but it resembles the dynamic
		// response structure of the PHP API
		response := map[string]interface{}{}
		entries, err := gd.GetQueueEntries(r.Context(), q.ID, admin)
		if err != nil {
			l.Errorw("failed to get queue entries", "err", err)
			s.internalServerError(w, r)
			return
		}

		// If user is logged in but not admin, check to
		// add their info to their queue entry(-ies)
		if !admin && email != "" {
			userEntries, err := gd.GetActiveQueueEntriesForUser(r.Context(), q.ID, email)
			if err != nil {
				l.Errorw("failed to get active queue entries for user",
					"err", err,
				)
				s.internalServerError(w, r)
				return
			}

			for _, userEntry := range userEntries {
				for i, e := range entries {
					if userEntry.ID == e.ID {
						entries[i] = userEntry
						break
					}
				}
			}
		}
		response["queue"] = entries

		if admin {
			stack, err := gd.GetQueueStack(r.Context(), q.ID, 20)
			if err != nil {
				l.Errorw("failed to get queue stack", "err", err)
				s.internalServerError(w, r)
				return
			}
			response["stack"] = stack
		}

		schedule, err := gd.GetCurrentDaySchedule(r.Context(), q.ID)
		if err != nil {
			l.Errorw("failed to get queue schedule", "err", err)
			s.internalServerError(w, r)
			return
		}
		response["schedule"] = schedule

		halfHour := CurrentHalfHour()
		response["half_hour"] = halfHour
		response["open"] = schedule[halfHour] == 'o' || schedule[halfHour] == 'p'

		announcements, err := gd.GetQueueAnnouncements(r.Context(), q.ID)
		if err != nil {
			l.Errorw("failed to get queue announcements", "err", err)
			s.internalServerError(w, r)
			return
		}
		response["announcements"] = announcements

		s.sendResponse(http.StatusOK, response, w, r)
	}
}

func (s *Server) GetQueueStack(gs getQueueStack) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		email := r.Context().Value(emailContextKey).(string)

		stack, err := gs.GetQueueStack(r.Context(), q.ID, 10000)
		if err != nil {
			s.logger.Errorw("failed to fetch stack",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"queue_id", q.ID,
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		s.logger.Infow("fetched stack",
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"queue_id", q.ID,
			"email", email,
			"stack_length", len(stack),
		)
		s.sendResponse(http.StatusOK, stack, w, r)
	}
}

type canAddEntry interface {
	CanAddEntry(ctx context.Context, queue ksuid.KSUID, email string) (bool, error)
}

type addQueueEntry interface {
	getQueueEntries
	getActiveQueueEntriesForUser
	canAddEntry
	AddQueueEntry(context.Context, *QueueEntry) (*QueueEntry, error)
}

func (s *Server) AddQueueEntry(ae addQueueEntry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		email := r.Context().Value(emailContextKey).(string)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"queue_id", q.ID,
			"email", email,
		)

		currentEntries, err := ae.GetActiveQueueEntriesForUser(r.Context(), q.ID, email)
		if err != nil {
			l.Errorw("failed to fetch current queue entries for user", "err", err)
			s.internalServerError(w, r)
			return
		}

		if len(currentEntries) > 0 {
			l.Warnw("attempted queue sign up with already existing entry",
				"conflicting_entry", currentEntries[0].ID,
			)
			s.errorMessage(
				http.StatusConflict,
				"Don't get greedy! You can only be on the queue once at a time.",
				w, r,
			)
			return
		}

		canSignUp, err := ae.CanAddEntry(r.Context(), q.ID, email)
		if err != nil || !canSignUp {
			l.Warnw("user attempting to sign up for queue not allowed to", "err", err)
			s.errorMessage(
				http.StatusForbidden,
				"My records say you aren't allowed to sign up right now. Are you in the course, or is another group member on the queue?",
				w, r,
			)
			return
		}

		var entry QueueEntry
		entry.Queue = q.ID
		entry.Email = email
		err = json.NewDecoder(r.Body).Decode(&entry)
		if err != nil {
			l.Warnw("failed to decode queue entry from body", "err", err)
			s.errorMessage(
				http.StatusBadRequest,
				"We couldn't read the queue entry from the request body.",
				w, r,
			)
			return
		}

		// Don't check location because it could be a map location;
		// we're using the frontend as a bit of a crutch here
		if entry.Description == "" || entry.Name == "" {
			l.Warnw("incomplete queue entry", "entry", entry)
			s.errorMessage(
				http.StatusBadRequest,
				"It looks like you left out some fields in the queue entry!",
				w, r,
			)
			return
		}

		newEntry, err := ae.AddQueueEntry(r.Context(), &entry)
		if err != nil {
			l.Errorw("failed to insert queue entry", "err", err)
			s.internalServerError(w, r)
			return
		}

		l.Infow("created queue entry", "entry_id", newEntry.ID)
		s.sendResponse(http.StatusCreated, newEntry, w, r)
	}
}

type updateQueueEntry interface {
	getQueueEntry
	UpdateQueueEntry(ctx context.Context, entry ksuid.KSUID, newEntry *QueueEntry) error
}

func (s *Server) UpdateQueueEntry(ue updateQueueEntry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "entry_id")
		email := r.Context().Value(emailContextKey).(string)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"entry_id", id,
			"email", email,
		)

		entry, err := ksuid.Parse(id)
		if err != nil {
			l.Warnw("failed to parse entry ID", "err", err)
			s.errorMessage(
				http.StatusNotFound,
				"I'm not able to find that queue entry.",
				w, r,
			)
			return
		}

		e, err := ue.GetQueueEntry(r.Context(), entry)
		if err != nil {
			l.Warnw("failed to get entry with valid ksuid", "err", err)
			s.errorMessage(
				http.StatusNotFound,
				"I'm not able to find that queue entry. Perhaps you were popped off quite recently?",
				w, r,
			)
			return
		}

		if e.Email != email {
			l.Warnw("user tried to update other user's queue entry", "entry_email", e.Email)
			s.errorMessage(
				http.StatusForbidden,
				"You can't edit someone else's queue entry!",
				w, r,
			)
			return
		}

		var newEntry QueueEntry
		err = json.NewDecoder(r.Body).Decode(&newEntry)
		if err != nil {
			l.Warnw("failed to decode queue entry from body", "err", err)
			s.errorMessage(
				http.StatusBadRequest,
				"We couldn't read the queue entry from the request body.",
				w, r,
			)
			return
		}

		if newEntry.Name == "" || newEntry.Description == "" {
			l.Warnw("incomplete queue entry", "entry", entry)
			s.errorMessage(
				http.StatusBadRequest,
				"It looks like you left out some fields in the queue entry!",
				w, r,
			)
			return
		}

		err = ue.UpdateQueueEntry(r.Context(), entry, &newEntry)
		if err != nil {
			l.Errorw("failed to update queue entry", "err", err)
			s.internalServerError(w, r)
			return
		}

		l.Infow("queue entry updated", "old_entry", e)
		s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}

type canRemoveQueueEntry interface {
	CanRemoveQueueEntry(ctx context.Context, queue ksuid.KSUID, entry ksuid.KSUID, email string) (bool, error)
}

type removeQueueEntry interface {
	canRemoveQueueEntry
	RemoveQueueEntry(ctx context.Context, entry ksuid.KSUID, remover string) error
}

func (s *Server) RemoveQueueEntry(re removeQueueEntry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		id := chi.URLParam(r, "entry_id")
		email := r.Context().Value(emailContextKey).(string)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"entry_id", id,
			"email", email,
		)

		entry, err := ksuid.Parse(id)
		if err != nil {
			l.Warnw("failed to parse entry ID", "err", err)
			s.errorMessage(
				http.StatusNotFound,
				"I'm not able to find that queue entry.",
				w, r,
			)
			return
		}

		canRemove, err := re.CanRemoveQueueEntry(r.Context(), q.ID, entry, email)
		if err != nil || !canRemove {
			l.Warnw("attempted to remove queue entry without access", "err", err)
			s.errorMessage(
				http.StatusForbidden,
				"Removing someone else's queue entry isn't very nice!",
				w, r,
			)
			return
		}

		err = re.RemoveQueueEntry(r.Context(), entry, email)
		if err != nil {
			l.Errorw("failed to remove queue entry", "err", err)
			s.internalServerError(w, r)
			return
		}

		l.Infow("removed queue entry")
		s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}

type clearQueueEntries interface {
	ClearQueueEntries(ctx context.Context, queue ksuid.KSUID, remover string) error
}

func (s *Server) ClearQueueEntries(ce clearQueueEntries) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		email := r.Context().Value(emailContextKey).(string)
		err := ce.ClearQueueEntries(r.Context(), q.ID, email)
		if err != nil {
			s.logger.Errorw("failed to clear queue",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"queue_id", q.ID,
				"email", email,
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		s.logger.Infow("cleared queue",
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"queue_id", q.ID,
			"email", email,
		)

		s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}

type addQueueAnnouncement interface {
	AddQueueAnnouncement(context.Context, ksuid.KSUID, *Announcement) (*Announcement, error)
}

func (s *Server) AddQueueAnnouncement(aa addQueueAnnouncement) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		email := r.Context().Value(emailContextKey).(string)

		var announcement Announcement
		err := json.NewDecoder(r.Body).Decode(&announcement)
		if err != nil {
			s.logger.Warnw("failed to decode announcement from body",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"email", email,
				"err", err,
			)
			s.errorMessage(
				http.StatusBadRequest,
				"We couldn't read the announcement from the request body.",
				w, r,
			)
			return
		}

		announcement.Queue = q.ID
		if announcement.Content == "" {
			s.logger.Warnw("received incomplete announcement",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"email", email,
				"announcement", announcement,
			)
			s.errorMessage(
				http.StatusBadRequest,
				"It looks like you left out some fields in the announcement.",
				w, r,
			)
			return
		}

		newAnnouncement, err := aa.AddQueueAnnouncement(r.Context(), q.ID, &announcement)
		if err != nil {
			s.logger.Errorw("failed to create new announcement",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"email", email,
				"announcement", announcement,
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		s.logger.Infow("created announcement",
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"email", email,
			"announcement_id", newAnnouncement,
		)
		s.sendResponse(http.StatusOK, newAnnouncement, w, r)
	}
}

type removeQueueAnnouncement interface {
	RemoveQueueAnnouncement(context.Context, ksuid.KSUID) error
}

func (s *Server) RemoveQueueAnnouncement(ra removeQueueAnnouncement) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := chi.URLParam(r, "announcement_id")
		announcement, err := ksuid.Parse(id)
		if err != nil {
			s.logger.Warnw("failed to parse announcement ID",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"announcement_id", id,
				"err", err,
			)
			s.errorMessage(
				http.StatusNotFound,
				"I couldn't find that announcement anywhere.",
				w, r,
			)
			return
		}

		err = ra.RemoveQueueAnnouncement(r.Context(), announcement)
		if err != nil {
			s.logger.Errorw("failed to remove announcement",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"announcement_id", announcement,
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		s.logger.Infow("removed announcement",
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"announcement_id", announcement,
			"email", r.Context().Value(emailContextKey),
		)
		s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}

type getQueueSchedule interface {
	GetQueueSchedule(ctx context.Context, queue ksuid.KSUID) ([]string, error)
}

func (s *Server) GetQueueSchedule(gs getQueueSchedule) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		schedules, err := gs.GetQueueSchedule(r.Context(), q.ID)
		if err != nil {
			s.logger.Errorw("failed to get queue schedule",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"queue_id", q.ID,
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		s.sendResponse(http.StatusOK, schedules, w, r)
	}
}

type updateQueueSchedule interface {
	UpdateQueueSchedule(ctx context.Context, queue ksuid.KSUID, schedules []string) error
}

func (s *Server) UpdateQueueSchedule(us updateQueueSchedule) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)

		var schedules []string
		err := json.NewDecoder(r.Body).Decode(&schedules)
		if err != nil {
			s.logger.Warnw("failed to decode schedules",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"queue_id", q.ID,
				"err", err,
			)
			s.errorMessage(
				http.StatusBadRequest,
				"We couldn't read the schedules from the request body.",
				w, r,
			)
			return
		}

		for i, schedule := range schedules {
			if len(schedule) != 48 {
				s.logger.Warnw("got schedule with length not 48",
					RequestIDContextKey, r.Context().Value(RequestIDContextKey),
					"queue_id", q.ID,
					"len", len(schedule),
					"day", i,
					"schedule", schedule,
				)
				s.errorMessage(
					http.StatusBadRequest,
					"Make sure your schedule is 48 characters long!",
					w, r,
				)
				return
			}
		}

		err = us.UpdateQueueSchedule(r.Context(), q.ID, schedules)
		if err != nil {
			s.logger.Errorw("failed to update schedule",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"queue_id", q.ID,
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		s.logger.Infow("updated queue schedule",
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"queue_id", q.ID,
		)
		s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}

type getQueueConfiguration interface {
	GetQueueConfiguration(ctx context.Context, queue ksuid.KSUID) (*QueueConfiguration, error)
}

func (s *Server) GetQueueConfiguration(gc getQueueConfiguration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)

		config, err := gc.GetQueueConfiguration(r.Context(), q.ID)
		if err != nil {
			s.logger.Errorw("failed to get queue configuration",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"queue_id", q.ID,
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		s.sendResponse(http.StatusOK, config, w, r)
	}
}

type updateQueueConfiguration interface {
	UpdateQueueConfiguration(ctx context.Context, queue ksuid.KSUID, configuration *QueueConfiguration) error
}

func (s *Server) UpdateQueueConfiguration(uc updateQueueConfiguration) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)

		var config QueueConfiguration
		err := json.NewDecoder(r.Body).Decode(&config)
		if err != nil {
			s.logger.Warnw("failed to decode configuration",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"queue_id", q.ID,
				"err", err,
			)
			s.errorMessage(
				http.StatusBadRequest,
				"We couldn't read the configuration from the request body.",
				w, r,
			)
			return
		}

		err = uc.UpdateQueueConfiguration(r.Context(), q.ID, &config)
		if err != nil {
			s.logger.Errorw("failed to update queue configuration",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"queue_id", q.ID,
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		s.logger.Infow("updated queue configuration", RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"queue_id", q.ID,
			"configuration", config,
		)
		s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}

type sendMessage interface {
	SendMessage(ctx context.Context, queue ksuid.KSUID, content, from, to string) (*Message, error)
}

func (s *Server) SendMessage(sm sendMessage) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		email := r.Context().Value(emailContextKey).(string)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"queue_id", q.ID,
			"from", email,
		)

		var message Message
		err := json.NewDecoder(r.Body).Decode(&message)
		if err != nil {
			l.Warnw("failed to decode message from body", "err", err)
			s.errorMessage(
				http.StatusBadRequest,
				"We couldn't read the message from the request body.",
				w, r,
			)
			return
		}

		if message.Receiver == "" || message.Content == "" {
			l.Warnw("got incomplete message", "message", message)
			s.errorMessage(
				http.StatusBadRequest,
				"It looks like you left out some fields from the message.",
				w, r,
			)
			return
		}

		newMessage, err := sm.SendMessage(r.Context(), q.ID, message.Content, email, message.Receiver)
		if err != nil {
			l.Errorw("failed to create message", "err", err)
			s.internalServerError(w, r)
			return
		}

		s.sendResponse(http.StatusCreated, newMessage, w, r)
	}
}
