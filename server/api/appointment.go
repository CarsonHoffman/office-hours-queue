package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi"
	"github.com/segmentio/ksuid"
)

const (
	appointmentDayContextKey      = "appointment_day"
	appointmentTimeslotContextKey = "appointment_timeslot"
	appointmentContextKey         = "appointment"
)

func (s *Server) AppointmentDayMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		day, err := strconv.Atoi(chi.URLParam(r, "day"))
		if err != nil {
			s.logger.Warnw("failed to parse day",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"day", chi.URLParam(r, "day"),
				"err", err,
			)
			s.errorMessage(
				http.StatusNotFound,
				"Are you sure that's a day?",
				w, r,
			)
			return
		}

		ctx := context.WithValue(r.Context(), appointmentDayContextKey, day)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) AppointmentTimeslotMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		timeslot, err := strconv.Atoi(chi.URLParam(r, "timeslot"))
		if err != nil {
			s.logger.Warnw("failed to parse timeslot",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"timeslot", chi.URLParam(r, "timeslot"),
				"params", chi.RouteContext(r.Context()).URLParams,
				"err", err,
			)
			s.errorMessage(
				http.StatusNotFound,
				"I don't think that timeslot exists, as much as I'd like it to.",
				w, r,
			)
			return
		}

		ctx := context.WithValue(r.Context(), appointmentTimeslotContextKey, timeslot)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

type getAppointment interface {
	GetAppointment(ctx context.Context, appointment ksuid.KSUID) (*AppointmentSlot, error)
}

func (s *Server) AppointmentIDMiddleware(ga getAppointment) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := chi.URLParam(r, "appointment_id")

			appointmentID, err := ksuid.Parse(id)
			if err != nil {
				s.logger.Warnw("failed to parse appointment ID",
					RequestIDContextKey, r.Context().Value(RequestIDContextKey),
					"appointment_id", id,
					"err", err,
				)
				s.errorMessage(
					http.StatusNotFound,
					"I called for help, but I couldn't find that appointment anywhere.",
					w, r,
				)
				return
			}

			appointment, err := ga.GetAppointment(r.Context(), appointmentID)
			if err != nil {
				s.logger.Warnw("failed to get non-existent appointment with valid ksuid",
					RequestIDContextKey, r.Context().Value(RequestIDContextKey),
					"appointment_id", id,
					"err", err,
				)
				s.errorMessage(
					http.StatusNotFound,
					"I called for help, but I couldn't find that appointment anywhere. Was it just deleted?",
					w, r,
				)
				return
			}

			ctx := context.WithValue(r.Context(), appointmentContextKey, appointment)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type getAppointmentsInTimeFrame interface {
	GetAppointments(ctx context.Context, queue ksuid.KSUID, from, to time.Time) ([]*AppointmentSlot, error)
}

type getAppointments interface {
	getAppointmentsInTimeFrame
	GetAppointmentsWithStudent(ctx context.Context, queue ksuid.KSUID, from, to time.Time) ([]*AppointmentSlot, error)
}

func (s *Server) GetAppointments(ga getAppointments) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		q := r.Context().Value(queueContextKey).(*Queue)
		admin := r.Context().Value(courseAdminContextKey).(bool)
		day := r.Context().Value(appointmentDayContextKey).(int)

		var appointments []*AppointmentSlot
		var err error
		start, end := WeekdayBounds(day)
		if admin {
			appointments, err = ga.GetAppointments(r.Context(), q.ID, start, end)
		} else {
			appointments, err = ga.GetAppointmentsWithStudent(r.Context(), q.ID, start, end)
		}

		if err != nil {
			s.logger.Errorw("failed to get appointments",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"err", err,
			)
			return err
		}

		return s.sendResponse(http.StatusOK, appointments, w, r)
	}
}

type getAppointmentsForUser interface {
	GetAppointmentsForUser(ctx context.Context, queue ksuid.KSUID, from, to time.Time, email string) ([]*AppointmentSlot, error)
}

func (s *Server) GetAppointmentsForCurrentUser(ga getAppointmentsForUser) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		q := r.Context().Value(queueContextKey).(*Queue)
		email := r.Context().Value(emailContextKey).(string)
		day := r.Context().Value(appointmentDayContextKey).(int)

		start, end := WeekdayBounds(day)
		appointments, err := ga.GetAppointmentsForUser(r.Context(), q.ID, start, end, email)
		if err != nil {
			s.logger.Errorw("failed to get appointments for user",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"queue_id", q.ID,
				"email", email,
				"day", day,
			)
			return err
		}

		return s.sendResponse(http.StatusOK, appointments, w, r)
	}
}

type getAppointmentSchedule interface {
	GetAppointmentSchedule(ctx context.Context, queue ksuid.KSUID) ([]*AppointmentSchedule, error)
}

func (s *Server) GetAppointmentSchedule(gs getAppointmentSchedule) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		q := r.Context().Value(queueContextKey).(*Queue)

		schedules, err := gs.GetAppointmentSchedule(r.Context(), q.ID)
		if err != nil {
			s.logger.Errorw("failed to get appointment schedule",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"queue_id", q.ID,
				"err", err,
			)
			return err
		}

		return s.sendResponse(http.StatusOK, schedules, w, r)
	}
}

type getAppointmentScheduleForDay interface {
	GetAppointmentScheduleForDay(ctx context.Context, queue ksuid.KSUID, day int) (*AppointmentSchedule, error)
}

func (s *Server) GetAppointmentScheduleForDay(gs getAppointmentScheduleForDay) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		q := r.Context().Value(queueContextKey).(*Queue)
		day := r.Context().Value(appointmentDayContextKey).(int)

		schedule, err := gs.GetAppointmentScheduleForDay(r.Context(), q.ID, day)
		if err != nil {
			s.logger.Errorw("failed to get appointment schedule",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"queue_id", q.ID,
				"day", day,
				"err", err,
			)
			return err
		}

		return s.sendResponse(http.StatusOK, schedule, w, r)
	}
}

type claimTimeslot interface {
	ClaimTimeslot(ctx context.Context, queue ksuid.KSUID, day, timeslot int, email string) (*AppointmentSlot, error)
}

func (s *Server) ClaimTimeslot(cs claimTimeslot) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		q := r.Context().Value(queueContextKey).(*Queue)
		email := r.Context().Value(emailContextKey).(string)
		day := r.Context().Value(appointmentDayContextKey).(int)
		timeslot := r.Context().Value(appointmentTimeslotContextKey).(int)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"queue_id", q.ID,
			"day", day,
			"timeslot", timeslot,
			"email", email,
		)

		appointment, err := cs.ClaimTimeslot(r.Context(), q.ID, day, timeslot, email)
		if err != nil {
			l.Errorw("failed to claim timeslot", "err", err)
			return StatusError{
				http.StatusBadRequest,
				"Failed to claim timeslot. Perhaps it has already been claimed? error: " + err.Error(),
			}
		}

		l.Infow("appointment claimed")

		s.ps.Pub(WS("APPOINTMENT_CREATE", appointment), QueueTopicAdmin(q.ID))

		return s.sendResponse(http.StatusCreated, nil, w, r)
	}
}

type unclaimAppointment interface {
	UnclaimAppointment(ctx context.Context, appointment ksuid.KSUID) (deleted bool, err error)
}

func (s *Server) UnclaimAppointment(us unclaimAppointment) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		q := r.Context().Value(queueContextKey).(*Queue)
		appointment := r.Context().Value(appointmentContextKey).(*AppointmentSlot)

		deleted, err := us.UnclaimAppointment(r.Context(), appointment.ID)
		if err != nil {
			s.logger.Errorw("failed to remove appointment claim",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"appointment_id", appointment.ID,
				"err", err,
			)
			return err
		}

		s.logger.Infow("removed appointment claim",
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"appointment_id", appointment.ID,
			"email", r.Context().Value(emailContextKey),
		)

		if deleted {
			s.ps.Pub(WS("APPOINTMENT_REMOVE", appointment), QueueTopicAdmin(q.ID))
		} else {
			appointment.StaffEmail = nil
			s.ps.Pub(WS("APPOINTMENT_UPDATE", appointment), QueueTopicAdmin(q.ID))
		}

		return s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}

type updateAppointmentSchedule interface {
	getAppointmentsInTimeFrame
	getAppointmentScheduleForDay
	getAppointmentsByTimeslot
	UpdateAppointmentSchedule(ctx context.Context, queue ksuid.KSUID, day int, schedule *AppointmentSchedule) error
}

func (s *Server) UpdateAppointmentSchedule(us updateAppointmentSchedule) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		q := r.Context().Value(queueContextKey).(*Queue)
		email := r.Context().Value(emailContextKey).(string)
		day := r.Context().Value(appointmentDayContextKey).(int)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"queue_id", q.ID,
			"day", day,
			"email", email,
		)

		currentSchedule, err := us.GetAppointmentScheduleForDay(r.Context(), q.ID, day)
		if err != nil {
			l.Errorw("failed to get existing appointment schedule", "err", err)
			return err
		}

		var schedule AppointmentSchedule
		err = json.NewDecoder(r.Body).Decode(&schedule)
		if err != nil {
			l.Warnw("failed to decode schedule from body", "err", err)
			return StatusError{
				http.StatusBadRequest,
				"We couldn't read the schedule in the request body.",
			}
		}

		from, to := WeekdayBounds(day)
		appointments, err := us.GetAppointments(r.Context(), q.ID, from, to)
		if err != nil {
			l.Errorw("failed to get appointments", "err", err)
			return err
		}

		if len(appointments) > 0 && currentSchedule.Duration != schedule.Duration {
			l.Warnw("appointment schedule duration update attempted with existing appointments")
			return StatusError{
				http.StatusConflict,
				"You can't change the appointment duration with active or past appointments on this day.",
			}
		}

		for i, n := range schedule.Schedule {
			currentTimeslotUsage, err := us.GetAppointmentsByTimeslot(r.Context(), q.ID, from, to, i)
			if err != nil {
				l.Errorw("failed to check appointments for timeslot", "err", err, "timeslot", i)
				return err
			}

			newTimeslotAvailability := int(n - '0')
			if newTimeslotAvailability < len(currentTimeslotUsage) {
				l.Warnw("tried to change appointment schedule to one without room",
					"conflicting_timeslot", i,
					"current_appointments", len(currentTimeslotUsage),
					"new_slots", newTimeslotAvailability,
				)
				return StatusError{
					http.StatusConflict,
					fmt.Sprintf("Setting that appointment schedule would remove an existing appointment. There are %d appointments at timeslot %d, but the new schedule only has %d slots at that time.",
						len(currentTimeslotUsage), i, newTimeslotAvailability),
				}
			}
		}

		err = us.UpdateAppointmentSchedule(r.Context(), q.ID, day, &schedule)
		if err != nil {
			l.Errorw("failed to update appointment schedule", "err", err)
			return err
		}

		l.Infow("updated appointment schedule")

		s.ps.Pub(WS("REFRESH", nil), QueueTopicGeneric(q.ID))

		return s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}

type getAppointmentsByTimeslot interface {
	GetAppointmentsByTimeslot(ctx context.Context, queue ksuid.KSUID, from, to time.Time, timeslot int) ([]*AppointmentSlot, error)
}

type signupForAppointment interface {
	getQueueConfiguration
	getAppointmentScheduleForDay
	getAppointmentsForUser
	getAppointmentsByTimeslot
	UserInQueueRoster(ctx context.Context, queue ksuid.KSUID, email string) (bool, error)
	TeammateHasAppointment(ctx context.Context, queue ksuid.KSUID, from, to time.Time, email string) (bool, error)
	SignupForAppointment(ctx context.Context, queue ksuid.KSUID, appointment *AppointmentSlot) (*AppointmentSlot, error)
}

func (s *Server) SignupForAppointment(sa signupForAppointment) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		q := r.Context().Value(queueContextKey).(*Queue)
		day := r.Context().Value(appointmentDayContextKey).(int)
		timeslot := r.Context().Value(appointmentTimeslotContextKey).(int)
		email := r.Context().Value(emailContextKey).(string)
		name := r.Context().Value(nameContextKey).(string)
		admin := r.Context().Value(courseAdminContextKey).(bool)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"queue_id", q.ID,
			"day", day,
			"timeslot", timeslot,
			"email", email,
		)

		config, err := sa.GetQueueConfiguration(r.Context(), q.ID)
		if err != nil {
			l.Errorw("failed to get queue configuration", "err", err)
			return err
		}

		if config.PreventUnregistered {
			inRoster, err := sa.UserInQueueRoster(r.Context(), q.ID, email)
			if err != nil {
				l.Errorw("failed to get queue roster", "err", err)
				return err
			}

			if !inRoster {
				l.Warnw("student not in queue roster attempted to sign up for appointment")
				return StatusError{
					http.StatusForbidden,
					"It doesn't look like you're in the roster for this queue. Contact your course staff if you think this is a mistake!",
				}
			}
		}

		schedule, err := sa.GetAppointmentScheduleForDay(r.Context(), q.ID, day)
		if err != nil {
			l.Errorw("failed to get appointment schedule", "err", err)
			return err
		}

		if config.PreventGroups {
			// Check if a group member has a future or ongoing appointment
			teammateHasAppointment, err := sa.TeammateHasAppointment(r.Context(), q.ID, time.Now().Add(-time.Minute*time.Duration(schedule.Duration)), BigTime(), email)
			if err != nil {
				l.Errorw("failed to get teammate appointments", "err", err)
				return err
			}

			if teammateHasAppointment {
				l.Warnw("student attempted to sign up for appointment with teammate on queue")
				return StatusError{
					http.StatusConflict,
					"It looks like one of your group members already has an appointment!",
				}
			}
		}

		var appointment AppointmentSlot
		err = json.NewDecoder(r.Body).Decode(&appointment)
		if err != nil {
			l.Warnw("failed to decode appointment", "err", err)
			return StatusError{
				http.StatusBadRequest,
				"We couldn't read your appointment in the request body.",
			}
		}
		appointment.Name = &name

		if appointment.Description == nil || appointment.Name == nil || appointment.Location == nil ||
			*appointment.Description == "" || *appointment.Name == "" || *appointment.Location == "" {
			l.Warnw("got incomplete appointment", "appointment", appointment)
			return StatusError{
				http.StatusBadRequest,
				"It looks like you left out some fields in the appointment.",
			}
		}

		if timeslot > len(schedule.Schedule) {
			l.Warnw("attempted to sign up for non-existent timeslot", "num_slots", len(schedule.Schedule))
			return StatusError{
				http.StatusNotFound,
				"That timeslot doesn't exist!",
			}
		}

		start, end := WeekdayBounds(day)

		// First: check if there are any slots open at this timeslot
		timeslotAppointments, err := sa.GetAppointmentsByTimeslot(r.Context(), q.ID, start, end, timeslot)
		if err != nil {
			l.Errorw("failed to get appointments for timeslot", "err", err)
			return err
		}

		open := int(schedule.Schedule[timeslot] - '0')
		for _, a := range timeslotAppointments {
			if a.StudentEmail != nil {
				open--
			}
		}

		if open < 1 {
			l.Warnw("no appointment slots available at timeslot")
			return StatusError{
				http.StatusConflict,
				"There are no slots open at that time!",
			}
		}

		// Check if the user has an appointment starting in the future
		// (or in the previous duration minutes, meaning they have an ongoing appointment)
		startFutureCheck := time.Now().Add(-time.Duration(schedule.Duration) * time.Minute)
		appointments, err := sa.GetAppointmentsForUser(r.Context(), q.ID, startFutureCheck, BigTime(), email)
		if err != nil {
			l.Errorw("failed to get future appointments for user", "err", err)
			return err
		}

		if len(appointments) > 0 {
			l.Warn("user attempted to sign up for appointment with one in future")
			return StatusError{
				http.StatusConflict,
				"You already have an appointment in the future!",
			}
		}

		// Force some values that were previously validated by middleware
		appointment.Queue = q.ID
		appointment.Timeslot = timeslot
		appointment.ScheduledTime = TimeslotToTime(day, timeslot, schedule.Duration)
		appointment.Duration = schedule.Duration
		appointment.StudentEmail = &email

		var zero float32
		if appointment.MapX == nil {
			appointment.MapX = &zero
		}
		if appointment.MapY == nil {
			appointment.MapY = &zero
		}

		newAppointment, err := sa.SignupForAppointment(r.Context(), q.ID, &appointment)
		if err != nil {
			l.Errorw("failed to sign up for appointment", "err", err)
			return err
		}

		l.Infow("new appointment sign up",
			"appointment_id", newAppointment.ID,
			"scheduled_time", appointment.ScheduledTime,
		)

		s.ps.Pub(WS("APPOINTMENT_CREATE", newAppointment), QueueTopicAdmin(q.ID))
		s.ps.Pub(WS("APPOINTMENT_CREATE", newAppointment.Anonymized()), QueueTopicNonPrivileged(q.ID))
		if !admin {
			s.ps.Pub(WS("APPOINTMENT_UPDATE", newAppointment.NoStaffEmail()), QueueTopicEmail(q.ID, email))
		}

		return s.sendResponse(http.StatusCreated, newAppointment, w, r)
	}
}

type removeAppointmentSignup interface {
	RemoveAppointmentSignup(ctx context.Context, appointment ksuid.KSUID) (deleted bool, newAppointment *AppointmentSlot, err error)
}

type updateAppointment interface {
	getAppointmentsByTimeslot
	getAppointmentScheduleForDay
	signupForAppointment
	removeAppointmentSignup
	UpdateAppointment(ctx context.Context, appointment ksuid.KSUID, newAppointment *AppointmentSlot) error
}

func (s *Server) UpdateAppointment(ua updateAppointment) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		q := r.Context().Value(queueContextKey).(*Queue)
		a := r.Context().Value(appointmentContextKey).(*AppointmentSlot)
		email := r.Context().Value(emailContextKey).(string)
		name := r.Context().Value(nameContextKey).(string)
		admin := r.Context().Value(courseAdminContextKey).(bool)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"appointment_id", a.ID,
			"email", email,
		)

		if a.StudentEmail == nil {
			l.Warnw("attempted to update deleted appointment", "appointment_id", a.ID)
			return StatusError{
				http.StatusNotFound,
				"This appointment doesn't exist. Perhaps it was already deleted?",
			}
		}

		if *a.StudentEmail != email {
			l.Warnw("user attempted to update appointment with other email",
				"expected_email", *a.StudentEmail,
			)
			return StatusError{
				http.StatusForbidden,
				"You can't update someone else's appointment!",
			}
		}

		var newAppointment AppointmentSlot
		err := json.NewDecoder(r.Body).Decode(&newAppointment)
		if err != nil {
			l.Warnw("failed to decode appointment", "err", err)
			return StatusError{
				http.StatusBadRequest,
				"We couldn't read your appointment in the request body.",
			}
		}
		newAppointment.Name = &name

		if newAppointment.Description == nil || newAppointment.Name == nil || newAppointment.Location == nil ||
			*newAppointment.Description == "" || *newAppointment.Name == "" || *newAppointment.Location == "" {
			l.Warnw("got incomplete appointment", "appointment", newAppointment)
			return StatusError{
				http.StatusBadRequest,
				"It looks like you left out some fields in the appointment.",
			}
		}

		newAppointment.ID = a.ID
		newAppointment.Queue = a.Queue
		newAppointment.Duration = a.Duration
		newAppointment.ScheduledTime = a.ScheduledTime
		newAppointment.StudentEmail = &email
		newAppointment.StaffEmail = a.StaffEmail

		var zero float32
		if newAppointment.MapX == nil {
			newAppointment.MapX = &zero
		}
		if newAppointment.MapY == nil {
			newAppointment.MapY = &zero
		}

		// We're not changing any times; simple.
		if newAppointment.Timeslot == a.Timeslot {
			err = ua.UpdateAppointment(r.Context(), a.ID, &newAppointment)
			if err != nil {
				l.Errorw("failed to update appointment", "err", err)
				return err
			}
			l.Infow("updated appointment")

			s.ps.Pub(WS("APPOINTMENT_UPDATE", &newAppointment), QueueTopicAdmin(q.ID))
			if !admin {
				s.ps.Pub(WS("APPOINTMENT_UPDATE", newAppointment.NoStaffEmail()), QueueTopicEmail(q.ID, email))
			}

			return s.sendResponse(http.StatusNoContent, nil, w, r)
		}

		// We're changing the appointment time. Not so simple.
		day := int(time.Now().Local().Weekday())
		schedule, err := ua.GetAppointmentScheduleForDay(r.Context(), a.Queue, day)
		if err != nil {
			l.Errorw("failed to get appointment schedule", "err", err)
			return err
		}

		start, end := WeekdayBounds(day)
		newTime := TimeslotToTime(day, newAppointment.Timeslot, schedule.Duration)
		newAppointment.ScheduledTime = newTime

		// If the new time is in the past, stop.
		if time.Now().After(newTime) {
			l.Warnw("user attempted to change appointment to past", "new_time", newTime)
			return StatusError{
				http.StatusBadRequest,
				"You can't change your appointment to the past! Let us know if you have a time machine.",
			}
		}

		if newAppointment.Timeslot > len(schedule.Schedule) {
			l.Warnw("attempted to change appointment to non-existent timeslot",
				"timeslot", newAppointment.Timeslot,
				"num_slots", len(schedule.Schedule),
			)
			return StatusError{
				http.StatusNotFound,
				"That timeslot doesn't exist!",
			}
		}

		timeslotAppointments, err := ua.GetAppointmentsByTimeslot(r.Context(), a.Queue, start, end, newAppointment.Timeslot)
		if err != nil {
			l.Errorw("failed to get appointments for timeslot", "timeslot", newAppointment.Timeslot, "err", err)
			return err
		}

		open := int(schedule.Schedule[newAppointment.Timeslot] - '0')
		for _, a := range timeslotAppointments {
			if a.StudentEmail != nil {
				open--
			}
		}

		if open < 1 {
			l.Warnw("no appointment slots available at timeslot", "timeslot", newAppointment.Timeslot)
			return StatusError{
				http.StatusConflict,
				"There are no slots open at that time!",
			}
		}

		// Add first so student doesn't lose appointment if the add fails
		createdAppointment, err := ua.SignupForAppointment(r.Context(), a.Queue, &newAppointment)
		if err != nil {
			l.Errorw("failed to create new appointment for update", "err", err)
			return err
		}
		l.Infow("created appointment for update", "new_appointment_id", createdAppointment.ID)

		// If adding the new appointment succeeded, ditch the old one.
		deleted, newSlot, err := ua.RemoveAppointmentSignup(r.Context(), a.ID)
		if err != nil {
			l.Errorw("failed to remove appointment for update", "err", err)
			return err
		}
		l.Infow("removed appointment for update")

		if deleted {
			s.ps.Pub(WS("APPOINTMENT_REMOVE", a.Anonymized()), QueueTopicGeneric(q.ID))
		} else {
			s.ps.Pub(WS("APPOINTMENT_UPDATE", newSlot), QueueTopicAdmin(q.ID))
			s.ps.Pub(WS("APPOINTMENT_REMOVE", a.Anonymized()), QueueTopicNonPrivileged(q.ID))
		}

		s.ps.Pub(WS("APPOINTMENT_CREATE", createdAppointment), QueueTopicAdmin(q.ID))
		s.ps.Pub(WS("APPOINTMENT_CREATE", createdAppointment.Anonymized()), QueueTopicNonPrivileged(q.ID))
		if !admin {
			s.ps.Pub(WS("APPOINTMENT_UPDATE", createdAppointment.NoStaffEmail()), QueueTopicEmail(q.ID, email))
		}

		return s.sendResponse(http.StatusCreated, createdAppointment, w, r)
	}
}

func (s *Server) RemoveAppointmentSignup(rs removeAppointmentSignup) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		q := r.Context().Value(queueContextKey).(*Queue)
		a := r.Context().Value(appointmentContextKey).(*AppointmentSlot)
		email := r.Context().Value(emailContextKey).(string)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"appointment_id", a.ID,
			"email", email,
		)

		if a.StudentEmail == nil {
			l.Warnw("attempted to remove signup for already deleted appointment")
			// Return 200 for idempotency---if someone tries to delete an appointment
			// twice, the second request still had the intended effect
			w.WriteHeader(http.StatusOK)
			return nil
		}

		if *a.StudentEmail != email {
			l.Warnw("user attempted to delete appointment with other email",
				"expected_email", *a.StudentEmail,
			)
			return StatusError{
				http.StatusForbidden,
				"You can't delete someone else's appointment!",
			}
		}

		// If an appointment happened, it happened. How did people do this in Spring D:
		if time.Now().After(a.ScheduledTime) {
			l.Warnw("user attempted to delete appointment in the past")
			return StatusError{
				http.StatusBadRequest,
				"You can't delete an appointment that already happened! Let's try not to cause a paradox here.",
			}
		}

		deleted, newSlot, err := rs.RemoveAppointmentSignup(r.Context(), a.ID)
		if err != nil {
			l.Errorw("failed to remove signup for appointment", "err", err)
			return err
		}

		l.Infow("removed signup for appointment")

		if deleted {
			s.ps.Pub(WS("APPOINTMENT_REMOVE", a.Anonymized()), QueueTopicGeneric(q.ID))
		} else {
			s.ps.Pub(WS("APPOINTMENT_UPDATE", newSlot), QueueTopicAdmin(q.ID))
			s.ps.Pub(WS("APPOINTMENT_REMOVE", a.Anonymized()), QueueTopicNonPrivileged(q.ID))
		}

		return s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}
