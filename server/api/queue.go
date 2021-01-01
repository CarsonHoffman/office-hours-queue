package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/gorilla/websocket"
	"github.com/olivere/elastic/v7"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/segmentio/ksuid"
)

func init() {
	prometheus.MustRegister(websocketCounter, websocketEventCounter)
}

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

type getQueueEntry interface {
	GetQueueEntry(ctx context.Context, entry ksuid.KSUID, allowRemoved bool) (*QueueEntry, error)
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

type viewMessage interface {
	ViewMessage(ctx context.Context, queue ksuid.KSUID, receiver string) (*Message, error)
}

type getQueueDetails interface {
	getQueueEntry
	getQueueEntries
	getActiveQueueEntriesForUser
	getQueueStack
	getQueueAnnouncements
	getCurrentDaySchedule
	viewMessage
}

func (s *Server) GetQueue(gd getQueueDetails) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"queue_id", q.ID,
		)

		admin := r.Context().Value(courseAdminContextKey).(bool)
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

		if email != "" {
			message, err := gd.ViewMessage(r.Context(), q.ID, email)
			if errors.Is(err, sql.ErrNoRows) {
			} else if err != nil {
				l.Errorw("failed to fetch message", "err", err)
				s.internalServerError(w, r)
				return
			} else {
				response["message"] = message
			}
		}

		s.sendResponse(http.StatusOK, response, w, r)
	}
}

var websocketCounter = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "websocket_count",
		Help: "The number of connected WebSocket clients per queue.",
	},
	[]string{"queue"},
)

var websocketEventCounter = prometheus.NewGaugeVec(
	prometheus.GaugeOpts{
		Name: "websocket_event_count",
		Help: "The number and type of WebSocket events sent (in total, to all clients) per queue.",
	},
	[]string{"queue", "event"},
)

var upgrader = &websocket.Upgrader{
	HandshakeTimeout: 30 * time.Second,
}

func (s *Server) QueueWebsocket() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var topics []string

		q := r.Context().Value(queueContextKey).(*Queue)
		topics = append(topics, QueueTopicGeneric(q.ID))

		admin := r.Context().Value(courseAdminContextKey).(bool)
		if admin {
			topics = append(topics, QueueTopicAdmin(q.ID))
		} else {
			topics = append(topics, QueueTopicNonPrivileged(q.ID))
		}

		// Yes, this is okay---see above
		email, _ := r.Context().Value(emailContextKey).(string)
		if email != "" {
			topics = append(topics, QueueTopicEmail(q.ID, email))
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			s.logger.Errorw("failed to upgrade to websocket connection",
				"queue_id", q.ID,
				"email", email,
				"err", err,
			)
			return
		}

		websocketCounter.With(prometheus.Labels{"queue": q.ID.String()}).Inc()

		events := s.ps.Sub(topics...)

		s.logger.Infow("websocket connection opened",
			"queue_id", q.ID,
			"email", email,
		)

		// The interval at which the server will expect pings from the client.
		const pingInterval = 30 * time.Second

		// The "slack" built into the ping logic; the extra time allowed
		// to clients to ping past the interval.
		const pingSlack = 10 * time.Second

		go func() {
			for {
				conn.SetReadDeadline(time.Now().Add(pingInterval + pingSlack))
				_, _, err := conn.ReadMessage()
				if err != nil {
					s.ps.Unsub(events)
					conn.WriteControl(
						websocket.CloseMessage,
						websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
						time.Now().Add(pingSlack),
					)
					conn.Close()

					websocketCounter.With(prometheus.Labels{"queue": q.ID.String()}).Dec()
					s.logger.Infow("websocket connection closed",
						"queue_id", q.ID,
						"email", email,
					)
					return
				}
			}
		}()

		pingTicker := time.NewTicker(pingInterval)
		for {
			var eventName string
			select {
			case <-pingTicker.C:
				// Using a custom ping message rather than a ping control
				// frame because browsers can't access control frames :(
				err = conn.WriteJSON(WS("PING", nil))
				eventName = "PING"
			case event, ok := <-events:
				if !ok {
					return
				}
				err = conn.WriteJSON(event)
				e, ok := event.(*WSMessage)
				if ok {
					eventName = e.Event
				}
			}
			websocketEventCounter.With(prometheus.Labels{"queue": q.ID.String(), "event": eventName}).Inc()

			// If the write fails, we presume that the read will also
			// fail, so the read loop will take care of unsubbing and
			// closing the connection. We also can't unsub on the same
			// goroutine from which we're listening for events. We should
			// just return.
			if err != nil {
				return
			}
		}
	}
}

type updateQueue interface {
	UpdateQueue(ctx context.Context, queue ksuid.KSUID, values *Queue) error
}

func (s *Server) UpdateQueue(uq updateQueue) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		email := r.Context().Value(emailContextKey).(string)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"queue_id", q.ID,
			"email", email,
		)

		var queue Queue
		err := json.NewDecoder(r.Body).Decode(&queue)
		if err != nil {
			l.Warnw("failed to decode queue from body", "err", err)
			s.errorMessage(
				http.StatusBadRequest,
				"We couldn't read the queue from the request body.",
				w, r,
			)
			return
		}

		if queue.Name == "" {
			l.Warnw("got incomplete queue", "queue", queue)
			s.errorMessage(
				http.StatusBadRequest,
				"It looks like you missed some fields in the queue!",
				w, r,
			)
			return
		}

		err = uq.UpdateQueue(r.Context(), q.ID, &queue)
		if err != nil {
			l.Errorw("failed to update queue", "err", err)
			s.internalServerError(w, r)
			return
		}

		l.Infow("updated queue")
		s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}

type removeQueue interface {
	RemoveQueue(ctx context.Context, queue ksuid.KSUID) error
}

func (s *Server) RemoveQueue(rq removeQueue) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		email := r.Context().Value(emailContextKey).(string)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"queue_id", q.ID,
			"email", email,
		)

		err := rq.RemoveQueue(r.Context(), q.ID)
		if err != nil {
			l.Errorw("failed to remove queue", "err", err)
			s.internalServerError(w, r)
			return
		}

		l.Infow("removed queue")
		s.sendResponse(http.StatusNoContent, nil, w, r)
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
	GetEntryPriority(ctx context.Context, queue ksuid.KSUID, email string) (int, error)
	AddQueueEntry(context.Context, *QueueEntry) (*QueueEntry, error)
}

func (s *Server) AddQueueEntry(ae addQueueEntry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		email := r.Context().Value(emailContextKey).(string)
		name := r.Context().Value(nameContextKey).(string)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"queue_id", q.ID,
			"course_id", q.Course,
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

		entry.Queue = q.ID
		entry.Email = email
		entry.Name = name
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

		priority, err := ae.GetEntryPriority(r.Context(), q.ID, email)
		if err != nil {
			l.Errorw("failed to get entry priority", "err", err)
			s.internalServerError(w, r)
			return
		}
		entry.Priority = priority

		newEntry, err := ae.AddQueueEntry(r.Context(), &entry)
		if err != nil {
			l.Errorw("failed to insert queue entry", "err", err)
			s.internalServerError(w, r)
			return
		}

		l.Infow("created queue entry", "entry_id", newEntry.ID)
		s.sendResponse(http.StatusCreated, newEntry, w, r)

		s.ps.Pub(WS("ENTRY_CREATE", newEntry), QueueTopicAdmin(q.ID))
		s.ps.Pub(WS("ENTRY_CREATE", newEntry.Anonymized()), QueueTopicNonPrivileged(q.ID))

		// Send an update with more information to the user who
		// created the queue entry.
		s.ps.Pub(WS("ENTRY_UPDATE", newEntry), QueueTopicEmail(q.ID, email))
	}
}

type updateQueueEntry interface {
	getQueueEntry
	UpdateQueueEntry(ctx context.Context, entry ksuid.KSUID, newEntry *QueueEntry) error
}

func (s *Server) UpdateQueueEntry(ue updateQueueEntry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		id := chi.URLParam(r, "entry_id")
		email := r.Context().Value(emailContextKey).(string)
		name := r.Context().Value(nameContextKey).(string)
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

		e, err := ue.GetQueueEntry(r.Context(), entry, false)
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
		newEntry.Name = name

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

		newEntry.ID = entry
		newEntry.Queue = q.ID
		newEntry.Email = e.Email
		newEntry.Pinned = e.Pinned
		newEntry.Priority = e.Priority

		s.ps.Pub(WS("ENTRY_UPDATE", &newEntry), QueueTopicAdmin(q.ID))
		s.ps.Pub(WS("ENTRY_UPDATE", &newEntry), QueueTopicEmail(q.ID, email))
	}
}

type canRemoveQueueEntry interface {
	CanRemoveQueueEntry(ctx context.Context, queue ksuid.KSUID, entry ksuid.KSUID, email string) (bool, error)
}

type removeQueueEntry interface {
	canRemoveQueueEntry
	RemoveQueueEntry(ctx context.Context, entry ksuid.KSUID, remover string) (*RemovedQueueEntry, error)
}

func (s *Server) RemoveQueueEntry(re removeQueueEntry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		id := chi.URLParam(r, "entry_id")
		email := r.Context().Value(emailContextKey).(string)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"entry_id", id,
			"queue_id", q.ID,
			"course_id", q.Course,
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

		e, err := re.RemoveQueueEntry(r.Context(), entry, email)
		if errors.Is(err, sql.ErrNoRows) {
			l.Warnw("attempted to remove already-removed queue entry", "err", err)
			s.errorMessage(
				http.StatusNotFound,
				"That queue entry was already removed by another staff member! Try the next one on the queue.",
				w, r,
			)
			return
		} else if err != nil {
			l.Errorw("failed to remove queue entry", "err", err)
			s.internalServerError(w, r)
			return
		}

		l.Infow("removed queue entry",
			"student_email", e.Email,
			"time_spent", time.Now().Sub(e.ID.Time()),
		)
		s.sendResponse(http.StatusNoContent, nil, w, r)

		s.ps.Pub(WS("ENTRY_REMOVE", e), QueueTopicAdmin(q.ID))
		s.ps.Pub(WS("ENTRY_REMOVE", e.Anonymized()), QueueTopicNonPrivileged(q.ID))
	}
}

type pinQueueEntry interface {
	getQueueEntry
	getActiveQueueEntriesForUser
	PinQueueEntry(ctx context.Context, entry ksuid.KSUID) error
}

func (s *Server) PinQueueEntry(pb pinQueueEntry) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		id := chi.URLParam(r, "entry_id")
		email := r.Context().Value(emailContextKey).(string)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"entry_id", id,
			"queue_id", q.ID,
			"course_id", q.Course,
			"email", email,
		)

		entryID, err := ksuid.Parse(id)
		if err != nil {
			l.Warnw("failed to parse entry ID", "err", err)
			s.errorMessage(
				http.StatusNotFound,
				"I'm not able to find that queue entry.",
				w, r,
			)
			return
		}

		entry, err := pb.GetQueueEntry(r.Context(), entryID, true)
		if err != nil {
			l.Warnw("attempted to get non-existent queue entry with valid ksuid")
			s.errorMessage(
				http.StatusNotFound,
				"I'm not able to find that queue entry.",
				w, r,
			)
			return
		}

		entries, err := pb.GetActiveQueueEntriesForUser(r.Context(), q.ID, entry.Email)
		if err != nil {
			l.Errorw("failed to get queue entries for user")
		}

		if entry.Removed && len(entries) > 0 {
			l.Warnw("attempted to pin queue entry with student on queue")
			s.errorMessage(
				http.StatusConflict,
				"That user is already on the queue. Pin their new entry!",
				w, r,
			)
			return
		}

		err = pb.PinQueueEntry(r.Context(), entryID)
		if err != nil {
			l.Errorw("failed to pin queue entry", "err", err)
			s.internalServerError(w, r)
			return
		}

		entry.Pinned = true

		l.Infow("pinned queue entry")
		s.sendResponse(http.StatusNoContent, nil, w, r)

		s.ps.Pub(WS("STACK_REMOVE", entry), QueueTopicAdmin(q.ID))
		s.ps.Pub(WS("ENTRY_CREATE", entry), QueueTopicAdmin(q.ID))
		s.ps.Pub(WS("ENTRY_CREATE", entry.Anonymized()), QueueTopicNonPrivileged(q.ID))

		// Send an update with more information to the user who
		// created the queue entry.
		s.ps.Pub(WS("ENTRY_UPDATE", entry), QueueTopicEmail(q.ID, email))
		s.ps.Pub(WS("ENTRY_PINNED", entry), QueueTopicEmail(q.ID, entry.Email))
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

		s.ps.Pub(WS("QUEUE_CLEAR", email), QueueTopicAdmin(q.ID))
		s.ps.Pub(WS("QUEUE_CLEAR", nil), QueueTopicNonPrivileged(q.ID))
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
			"announcement", newAnnouncement,
		)
		s.sendResponse(http.StatusOK, newAnnouncement, w, r)

		s.ps.Pub(WS("ANNOUNCEMENT_CREATE", newAnnouncement), QueueTopicGeneric(q.ID))
	}
}

type removeQueueAnnouncement interface {
	RemoveQueueAnnouncement(context.Context, ksuid.KSUID) error
}

func (s *Server) RemoveQueueAnnouncement(ra removeQueueAnnouncement) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)

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

		s.ps.Pub(WS("ANNOUNCEMENT_DELETE", announcement.String()), QueueTopicGeneric(q.ID))
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

		s.ps.Pub(WS("REFRESH", nil), QueueTopicGeneric(q.ID))
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

		s.ps.Pub(WS("MESSAGE_CREATE", newMessage), QueueTopicEmail(q.ID, message.Receiver))
	}
}

type getQueueRoster interface {
	GetQueueRoster(ctx context.Context, queue ksuid.KSUID) ([]string, error)
}

func (s *Server) GetQueueRoster(gr getQueueRoster) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)

		roster, err := gr.GetQueueRoster(r.Context(), q.ID)
		if err != nil {
			s.logger.Errorw("failed to fetch queue roster",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"queue_id", q.ID,
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		s.sendResponse(http.StatusOK, roster, w, r)
	}
}

type getQueueGroups interface {
	GetQueueGroups(ctx context.Context, queue ksuid.KSUID) ([][]string, error)
}

func (s *Server) GetQueueGroups(gg getQueueGroups) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)

		groups, err := gg.GetQueueGroups(r.Context(), q.ID)
		if err != nil {
			s.logger.Errorw("failed to fetch queue groups",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"queue_id", q.ID,
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		s.sendResponse(http.StatusOK, groups, w, r)
	}
}

type updateQueueGroups interface {
	UpdateQueueRoster(ctx context.Context, queue ksuid.KSUID, students []string) error
	UpdateQueueGroups(ctx context.Context, queue ksuid.KSUID, groups [][]string) error
}

func (s *Server) UpdateQueueGroups(ug updateQueueGroups) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)

		var groups [][]string
		err := json.NewDecoder(r.Body).Decode(&groups)
		if err != nil {
			s.logger.Warnw("failed to read groups from body",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"queue_id", q.ID,
				"err", err,
			)
			s.errorMessage(http.StatusBadRequest, fmt.Sprintf("I couldn't read the groups you uploaded. Make sure the file is structured as an array of arrays of students' emails, each inner array representing a group. This error might help: %v", err), w, r)
			return
		}

		err = ug.UpdateQueueGroups(r.Context(), q.ID, groups)
		if err != nil {
			s.logger.Errorw("failed to update groups",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"queue_id", q.ID,
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		var students []string
		for _, group := range groups {
			for _, student := range group {
				students = append(students, student)
			}
		}

		err = ug.UpdateQueueRoster(r.Context(), q.ID, students)
		if err != nil {
			s.logger.Errorw("failed to update roster",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"queue_id", q.ID,
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		s.logger.Infow("updated groups",
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"queue_id", q.ID,
			"email", r.Context().Value(emailContextKey),
		)
		s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}

func (s *Server) GetQueueLogs() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		email := r.Context().Value(emailContextKey).(string)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"queue_id", q.ID,
			"email", email,
		)

		es, err := elastic.NewClient(elastic.SetURL("http://elasticsearch:9200"))
		if err != nil {
			l.Errorw("couldn't set up elastic connection", "err", err)
			s.internalServerError(w, r)
			return
		}

		result, err := es.Search().
			Index("logstash-api-*").
			Query(elastic.NewTermQuery("queue_id.keyword", q.ID.String())).
			Sort("@timestamp", false).
			Size(10000).
			Do(r.Context())

		if err != nil {
			l.Errorw("failed to fetch elastic results", "err", err)
			s.internalServerError(w, r)
			return
		}

		var output []json.RawMessage
		for _, hit := range result.Hits.Hits {
			output = append(output, hit.Source)
		}

		l.Infow("fetched logs", "num_entries", len(result.Hits.Hits))
		s.sendResponse(http.StatusOK, output, w, r)
	}
}

type setNotHelped interface {
	getQueueEntry
	SetHelpedStatus(ctx context.Context, entry ksuid.KSUID, helped bool) error
}

func (s *Server) SetNotHelped(sh setNotHelped) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.Context().Value(queueContextKey).(*Queue)
		id := chi.URLParam(r, "entry_id")
		email := r.Context().Value(emailContextKey).(string)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"entry_id", id,
			"queue_id", q.ID,
			"course_id", q.Course,
			"email", email,
		)

		entryID, err := ksuid.Parse(id)
		if err != nil {
			l.Warnw("failed to parse entry ID", "err", err)
			s.errorMessage(
				http.StatusNotFound,
				"I'm not able to find that queue entry.",
				w, r,
			)
			return
		}

		entry, err := sh.GetQueueEntry(r.Context(), entryID, true)
		if err != nil {
			l.Warnw("attempted to get non-existent queue entry with valid ksuid")
			s.errorMessage(
				http.StatusNotFound,
				"I'm not able to find that queue entry.",
				w, r,
			)
			return
		}

		err = sh.SetHelpedStatus(r.Context(), entryID, false)
		if err != nil {
			l.Errorw("failed to set entry to not helped", "err", err)
			s.internalServerError(w, r)
			return
		}

		entry.Helped = false

		l.Infow("set entry to not helped")
		s.sendResponse(http.StatusNoContent, nil, w, r)

		s.ps.Pub(WS("ENTRY_UPDATE", entry.RemovedEntry()), QueueTopicAdmin(q.ID))
		s.ps.Pub(WS("NOT_HELPED", nil), QueueTopicEmail(q.ID, email))
	}
}
