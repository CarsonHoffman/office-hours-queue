package api

import (
	"database/sql"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/antonlindstrom/pgstore"
	"github.com/go-chi/chi"
	"github.com/gorilla/sessions"
	"go.uber.org/zap"
)

type Server struct {
	chi.Router

	logger   *zap.SugaredLogger
	sessions *pgstore.PGStore
}

// All of the abilities that a complete backing
// store for the queue should have.
type queueStore interface {
	siteAdmin
	queueAdmin

	getCourses
	getCourse
	getAdminCourses
	addCourse

	getQueues
	getQueue
	addQueue
	getQueueEntry
	getQueueEntries
	addQueueEntry
	updateQueueEntry
	clearQueueEntries
	removeQueueEntry
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

func New(q queueStore, sessionsStore *sql.DB) *Server {
	var s Server
	z, _ := zap.NewProduction()
	s.logger = z.Sugar().With("name", "queue")

	key, err := ioutil.ReadFile(os.Getenv("QUEUE_SESSIONS_KEY_FILE"))
	if err != nil {
		log.Fatalln("couldn't load sessions key:", err)
	}

	s.sessions, err = pgstore.NewPGStoreFromPool(sessionsStore, key)
	if err != nil {
		log.Fatalln("couldn't set up session store:", err)
	}
	s.sessions.Options = &sessions.Options{
		HttpOnly: true,
		Secure:   os.Getenv("USE_SECURE_COOKIES") == "true",
	}

	s.Router = chi.NewRouter()
	s.Router.Use(ksuidInserter, s.recoverMiddleware, s.sessionRetriever)

	// Public API routes
	s.Route("/courses", func(r chi.Router) {
		r.Get("/", s.GetCourses(q))
		r.With(s.ValidLoginMiddleware, s.EnsureSiteAdmin(q)).Post("/", s.AddCourse(q))

		r.Route("/{id:[a-zA-Z0-9]{27}}", func(r chi.Router) {
			r.Use(s.CourseIDMiddleware(q))
			r.Get("/", s.GetCourse(q))
			r.Get("/queues", s.GetQueues(q))
			r.With(s.ValidLoginMiddleware, s.EnsureSiteAdmin(q)).Post("/queues", s.AddQueue(q))
		})

		r.With(s.ValidLoginMiddleware).Get("/admin/@me", s.GetAdminCourses(q))
	})

	s.Route("/queues/{id:[a-zA-Z0-9]{27}}", func(r chi.Router) {
		r.Use(s.QueueIDMiddleware(q), s.CheckQueueAdmin(q))
		r.Get("/", s.GetQueue(q))
		r.With(s.ValidLoginMiddleware, s.EnsureQueueAdmin).Get("/stack", s.GetQueueStack(q))

		r.Route("/entries", func(r chi.Router) {
			r.With(s.ValidLoginMiddleware).Post("/", s.AddQueueEntry(q))
			r.With(s.ValidLoginMiddleware).Put("/{entry_id:[a-zA-Z0-9]{27}}", s.UpdateQueueEntry(q))
			r.With(s.ValidLoginMiddleware).Delete("/{entry_id:[a-zA-Z0-9]{27}}", s.RemoveQueueEntry(q))
			r.With(s.ValidLoginMiddleware, s.EnsureQueueAdmin).Delete("/", s.ClearQueueEntries(q))
		})

		r.Route("/announcements", func(r chi.Router) {
			r.Use(s.ValidLoginMiddleware, s.EnsureQueueAdmin)
			r.Post("/", s.AddQueueAnnouncement(q))
			r.Delete("/{announcement_id:[a-zA-Z0-9]{27}}", s.RemoveQueueAnnouncement(q))
		})

		r.Route("/schedule", func(r chi.Router) {
			r.Get("/", s.GetQueueSchedule(q))
			r.With(s.ValidLoginMiddleware, s.EnsureQueueAdmin).Put("/", s.UpdateQueueSchedule(q))
		})

		r.Route("/configuration", func(r chi.Router) {
			r.Get("/", s.GetQueueConfiguration(q))
			r.With(s.ValidLoginMiddleware, s.EnsureQueueAdmin).Put("/", s.UpdateQueueConfiguration(q))
		})

		r.With(s.ValidLoginMiddleware, s.EnsureQueueAdmin).Post("/messages", s.SendMessage(q))

		r.Route("/appointments", func(r chi.Router) {
			r.Route(`/{day:\d+}`, func(r chi.Router) {
				r.Use(s.AppointmentDayMiddleware)
				r.Get("/", s.GetAppointments(q))
				r.With(s.ValidLoginMiddleware).Get("/@me", s.GetAppointmentsForCurrentUser(q))

				r.With(s.ValidLoginMiddleware, s.AppointmentTimeslotMiddleware).Post(`/{timeslot:\d+}`, s.SignupForAppointment(q))

				r.Route(`/claims/{timeslot:\d+}`, func(r chi.Router) {
					r.Use(s.ValidLoginMiddleware, s.EnsureQueueAdmin, s.AppointmentTimeslotMiddleware)
					r.Put("/", s.ClaimTimeslot(q))
				})
			})

			r.Route(`/claims/{appointment_id:[a-zA-Z0-9]{27}}`, func(r chi.Router) {
				r.Use(s.ValidLoginMiddleware, s.EnsureQueueAdmin, s.AppointmentIDMiddleware(q))
				r.Delete("/", s.UnclaimAppointment(q))
			})

			r.Route(`/{appointment_id:[a-zA-Z0-9]{27}}`, func(r chi.Router) {
				r.Use(s.AppointmentIDMiddleware(q))
				r.Put("/", s.UpdateAppointment(q))
				r.Delete("/", s.RemoveAppointmentSignup(q))
			})

			r.Route("/schedule", func(r chi.Router) {
				r.Get("/", s.GetAppointmentSchedule(q))
				r.Route(`/{day:\d+}`, func(r chi.Router) {
					r.Use(s.AppointmentDayMiddleware)
					r.Get("/", s.GetAppointmentScheduleForDay(q))
					r.With(s.ValidLoginMiddleware, s.EnsureQueueAdmin).Put("/", s.UpdateAppointmentSchedule(q))
				})
			})
		})
	})

	// Auth
	s.Post("/login", s.Login())

	s.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	return &s
}
