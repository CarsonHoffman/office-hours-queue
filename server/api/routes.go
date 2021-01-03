package api

import (
	"database/sql"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/antonlindstrom/pgstore"
	"github.com/cskr/pubsub"
	"github.com/go-chi/chi"
	"github.com/gorilla/sessions"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
)

type Server struct {
	chi.Router

	logger          *zap.SugaredLogger
	sessions        *pgstore.PGStore
	ps              *pubsub.PubSub
	oauthConfig     oauth2.Config
	baseURL         string
	metricsPassword string
}

// All of the abilities that a complete backing
// store for the queue should have.
type queueStore interface {
	siteAdmin
	courseAdmin
	getUserInfo

	getCourses
	getCourse
	getAdminCourses
	addCourse
	updateCourse
	getCourseAdmins
	addCourseAdmins
	removeCourseAdmins

	getQueues
	getQueue
	addQueue
	updateQueue
	removeQueue
	getQueueEntry
	getQueueEntries
	addQueueEntry
	updateQueueEntry
	clearQueueEntries
	removeQueueEntry
	pinQueueEntry
	getQueueStack
	getQueueAnnouncements
	addQueueAnnouncement
	removeQueueAnnouncement
	getCurrentDaySchedule
	getQueueSchedule
	updateQueueSchedule
	getQueueConfiguration
	updateQueueConfiguration
	sendMessage
	viewMessage
	getQueueRoster
	getQueueGroups
	updateQueueGroups
	setNotHelped
	queueStats

	getAppointment
	getAppointments
	getAppointmentsForUser
	getAppointmentsByTimeslot
	getAppointmentSchedule
	getAppointmentScheduleForDay
	updateAppointmentSchedule
	claimTimeslot
	unclaimAppointment
	signupForAppointment
	updateAppointment
	removeAppointmentSignup
}

func New(q queueStore, logger *zap.SugaredLogger, sessionsStore *sql.DB, oauthConfig oauth2.Config) *Server {
	var s Server
	s.logger = logger

	key, err := ioutil.ReadFile(os.Getenv("QUEUE_SESSIONS_KEY_FILE"))
	if err != nil {
		logger.Fatalw("couldn't load sessions key", "err", err)
	}

	s.sessions, err = pgstore.NewPGStoreFromPool(sessionsStore, key)
	if err != nil {
		logger.Fatalw("couldn't set up session store", "err", err)
	}
	s.sessions.Options = &sessions.Options{
		HttpOnly: true,
		Secure:   os.Getenv("USE_SECURE_COOKIES") == "true",
		MaxAge:   60 * 60 * 24 * 30,
		Path:     "/",
	}

	metricsPassword, err := ioutil.ReadFile(os.Getenv("METRICS_PASSWORD_FILE"))
	if err != nil {
		logger.Fatalw("couldn't load metrics password", "err", err)
	}
	s.metricsPassword = string(metricsPassword)

	// TODO: evaluate capacity choice for channel. This assumes that
	// there isn't likely to be more than 5 events in "quick" succession
	// to any particular connection, and reduces overall latency between
	// sending on different connections in that case, but allocates room
	// for 5 events on every connection. There isn't an empirical basis here.
	// Just a guess.
	s.ps = pubsub.New(5)

	s.oauthConfig = oauthConfig

	s.baseURL = os.Getenv("QUEUE_BASE_URL")

	s.Router = chi.NewRouter()
	s.Router.Use(instrumenter, ksuidInserter, s.recoverMiddleware, s.sessionRetriever)

	// Course endpoints
	s.Route("/courses", func(r chi.Router) {
		// Get all courses
		r.Get("/", s.GetCourses(q))

		// Create course (course admin)
		r.With(s.ValidLoginMiddleware, s.EnsureSiteAdmin(q)).Post("/", s.AddCourse(q))

		// Course by ID endpoints
		r.Route("/{id:[a-zA-Z0-9]{27}}", func(r chi.Router) {
			r.Use(s.CourseIDMiddleware(q))

			// Get course by ID
			r.Get("/", s.GetCourse(q))

			// Get course's queues
			r.Get("/queues", s.GetQueues(q))

			// Update course (site admin)
			r.With(s.ValidLoginMiddleware, s.EnsureSiteAdmin(q)).Put("/", s.UpdateCourse(q))

			// Create queue on course (course admin)
			r.With(s.ValidLoginMiddleware, s.CheckCourseAdmin(q), s.EnsureCourseAdmin).Post("/queues", s.AddQueue(q))

			// Course admin management (course admin)
			r.Route("/admins", func(r chi.Router) {
				r.Use(s.ValidLoginMiddleware, s.CheckCourseAdmin(q), s.EnsureCourseAdmin)

				// Get course admins (course admin)
				r.Get("/", s.GetCourseAdmins(q))

				// Add course admins (course admin)
				r.Post("/", s.AddCourseAdmins(q))

				// Overwrite course admins (course admin)
				r.Put("/", s.UpdateCourseAdmins(q))

				// Remove course admins (course admin)
				r.Delete("/", s.RemoveCourseAdmins(q))
			})
		})
	})

	// Queue by ID endpoints
	s.Route("/queues/{id:[a-zA-Z0-9]{27}}", func(r chi.Router) {
		r.Use(s.QueueIDMiddleware(q), s.CheckCourseAdmin(q))

		// Get queue by ID (more information with queue admin)
		r.Get("/", s.GetQueue(q))

		r.Get("/ws", s.QueueWebsocket())

		r.With(s.ValidLoginMiddleware, s.EnsureCourseAdmin).Put("/", s.UpdateQueue(q))

		r.With(s.ValidLoginMiddleware, s.EnsureSiteAdmin(q)).Delete("/", s.RemoveQueue(q))

		// Get queue's stack (queue admin)
		r.With(s.ValidLoginMiddleware, s.EnsureCourseAdmin).Get("/stack", s.GetQueueStack(q))

		// Get queue logs (course admin)
		r.With(s.ValidLoginMiddleware, s.EnsureCourseAdmin).Get("/logs", s.GetQueueLogs())

		// Entry by ID endpoints
		r.Route("/entries", func(r chi.Router) {
			r.Use(s.ValidLoginMiddleware)

			// Add queue entry (valid login)
			r.Post("/", s.AddQueueEntry(q))

			// Update queue entry (valid login, same user as creator)
			r.Put("/{entry_id:[a-zA-Z0-9]{27}}", s.UpdateQueueEntry(q))

			// Remove queue entry (valid login, same user or queue admin)
			r.Delete("/{entry_id:[a-zA-Z0-9]{27}}", s.RemoveQueueEntry(q))

			// Pin queue entry (course admin)
			r.With(s.EnsureCourseAdmin).Post("/{entry_id:[a-zA-Z0-9]{27}}/pin", s.PinQueueEntry(q))

			// Set student not helped (queue admin)
			r.With(s.EnsureCourseAdmin).Delete("/{entry_id:[a-zA-Z0-9]{27}}/helped", s.SetNotHelped(q))

			// Clear queue (queue admin)
			r.With(s.EnsureCourseAdmin).Delete("/", s.ClearQueueEntries(q))
		})

		// Announcements endpoints
		r.Route("/announcements", func(r chi.Router) {
			r.Use(s.ValidLoginMiddleware, s.EnsureCourseAdmin)

			// Create announcement (queue admin)
			r.Post("/", s.AddQueueAnnouncement(q))

			// Remove announcement (queue admin)
			r.Delete("/{announcement_id:[a-zA-Z0-9]{27}}", s.RemoveQueueAnnouncement(q))
		})

		// Queue-wide (all days) schedule endpoints
		r.Route("/schedule", func(r chi.Router) {
			// Get queue schedule
			r.Get("/", s.GetQueueSchedule(q))

			// Update queue schedule (queue admin)
			r.With(s.ValidLoginMiddleware, s.EnsureCourseAdmin).Put("/", s.UpdateQueueSchedule(q))
		})

		// Queue configuration endpoints
		r.Route("/configuration", func(r chi.Router) {
			// Get queue configuration
			r.Get("/", s.GetQueueConfiguration(q))

			// Update queue configuration (queue admin)
			r.With(s.ValidLoginMiddleware, s.EnsureCourseAdmin).Put("/", s.UpdateQueueConfiguration(q))
		})

		// Send message (queue admin)
		r.With(s.ValidLoginMiddleware, s.EnsureCourseAdmin).Post("/messages", s.SendMessage(q))

		// Get queue roster (queue admin)
		r.With(s.ValidLoginMiddleware, s.EnsureCourseAdmin).Get("/roster", s.GetQueueRoster(q))

		// Queue groups endpoints
		r.Route("/groups", func(r chi.Router) {
			r.Use(s.ValidLoginMiddleware, s.EnsureCourseAdmin)

			// Get queue groups (queue admin)
			r.Get("/", s.GetQueueGroups(q))

			// Update queue groups (queue admin)
			r.Put("/", s.UpdateQueueGroups(q))
		})

		// Appointments endpoints
		r.Route("/appointments", func(r chi.Router) {
			// Specific day endpoints
			r.Route(`/{day:\d+}`, func(r chi.Router) {
				r.Use(s.AppointmentDayMiddleware)

				// Get endpoints on day (more information with queue admin)
				r.Get("/", s.GetAppointments(q))

				// Get appointments for current user on day
				r.With(s.ValidLoginMiddleware).Get("/@me", s.GetAppointmentsForCurrentUser(q))

				// Create appointment on day at timeslot
				r.With(s.ValidLoginMiddleware, s.AppointmentTimeslotMiddleware).Post(`/{timeslot:\d+}`, s.SignupForAppointment(q))

				// Appointment claiming (queue admin)
				r.Route(`/claims/{timeslot:\d+}`, func(r chi.Router) {
					r.Use(s.ValidLoginMiddleware, s.EnsureCourseAdmin, s.AppointmentTimeslotMiddleware)

					// Claim appointment on day at timeslot (queue admin)
					r.Put("/", s.ClaimTimeslot(q))
				})
			})

			// Existing appointment claims by ID (queue admin)
			r.Route(`/claims/{appointment_id:[a-zA-Z0-9]{27}}`, func(r chi.Router) {
				r.Use(s.ValidLoginMiddleware, s.EnsureCourseAdmin, s.AppointmentIDMiddleware(q))

				// Un-claim appointment (queue admin)
				r.Delete("/", s.UnclaimAppointment(q))
			})

			// Appointment by ID endpoints
			r.Route(`/{appointment_id:[a-zA-Z0-9]{27}}`, func(r chi.Router) {
				r.Use(s.ValidLoginMiddleware, s.AppointmentIDMiddleware(q))

				// Update appointment (valid login, same user as creator)
				r.Put("/", s.UpdateAppointment(q))

				// Cancel appointment (valid login, same user as creator)
				r.Delete("/", s.RemoveAppointmentSignup(q))
			})

			// Appointment schedule endpoints
			r.Route("/schedule", func(r chi.Router) {
				// Get appointment schedule for all days
				r.Get("/", s.GetAppointmentSchedule(q))

				// Per-day schedules
				r.Route(`/{day:\d+}`, func(r chi.Router) {
					r.Use(s.AppointmentDayMiddleware)

					// Get appointment schedule for day
					r.Get("/", s.GetAppointmentScheduleForDay(q))

					// Update appointment schedule for day (queue admin)
					r.With(s.ValidLoginMiddleware, s.EnsureCourseAdmin).Put("/", s.UpdateAppointmentSchedule(q))
				})
			})
		})
	})

	// Login handler (takes Google idtoken, sets up session)
	s.Post("/login", s.Login())

	s.Get("/oauth2login", s.OAuth2LoginLink())

	s.Get("/oauth2callback", s.OAuth2Callback())

	s.Get("/logout", s.Logout())

	s.With(s.ValidLoginMiddleware).Get("/users/@me", s.GetCurrentUserInfo(q))

	s.Get("/metrics", s.MetricsHandler())

	s.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	s.RegisterQueueStats(q)

	return &s
}
