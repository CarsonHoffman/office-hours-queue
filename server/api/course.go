package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/lib/pq"
	"github.com/segmentio/ksuid"
)

const courseContextKey = "course"

type getCourse interface {
	GetCourse(context.Context, ksuid.KSUID) (*Course, error)
}

func (s *Server) CourseIDMiddleware(gc getCourse) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			idString := chi.URLParam(r, "id")
			id, err := ksuid.Parse(idString)
			if err != nil {
				s.logger.Warnw("failed to parse course id",
					RequestIDContextKey, r.Context().Value(RequestIDContextKey),
					"course_id", idString,
				)
				s.errorMessage(
					http.StatusNotFound,
					"I've looked everywhere, but I can't find that course.",
					w, r,
				)
				return
			}

			c, err := gc.GetCourse(r.Context(), id)
			if errors.Is(err, sql.ErrNoRows) {
				s.logger.Warnw("failed to get non-existent course with valid ksuid",
					RequestIDContextKey, r.Context().Value(RequestIDContextKey),
					"course_id", idString,
				)
				s.errorMessage(
					http.StatusNotFound,
					"I've looked everywhere, but I can't find that course.",
					w, r,
				)
				return
			} else if err != nil {
				s.logger.Errorw("failed to get queue",
					RequestIDContextKey, r.Context().Value(RequestIDContextKey),
					"course_id", idString,
					"err", err,
				)
				s.internalServerError(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), courseContextKey, c)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

type getCourses interface {
	GetCourses(context.Context) ([]*Course, error)
}

func (s *Server) GetCourses(gc getCourses) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		courses, err := gc.GetCourses(r.Context())
		if err != nil {
			s.logger.Errorw("failed to fetch courses from DB",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		s.sendResponse(http.StatusOK, courses, w, r)
	}
}

func (s *Server) GetCourse(gc getCourse) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		s.sendResponse(http.StatusOK, r.Context().Value(courseContextKey), w, r)
	}
}

type getAdminCourses interface {
	GetAdminCourses(ctx context.Context, email string) ([]string, error)
}

func (s *Server) GetAdminCourses(gc getAdminCourses) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.Context().Value(emailContextKey).(string)
		courses, err := gc.GetAdminCourses(r.Context(), email)
		if err != nil {
			s.logger.Errorw("failed to get admin courses",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"email", email,
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		s.sendResponse(http.StatusOK, courses, w, r)
	}
}

type getQueues interface {
	getCourse
	GetQueues(ctx context.Context, course ksuid.KSUID) ([]*Queue, error)
}

func (s *Server) GetQueues(gq getQueues) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := r.Context().Value(courseContextKey).(*Course)

		queues, err := gq.GetQueues(r.Context(), c.ID)
		if err != nil {
			s.logger.Errorw("failed to get queues from course",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"course_id", c.ID,
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		s.sendResponse(http.StatusOK, queues, w, r)
	}
}

type addCourse interface {
	AddCourse(ctx context.Context, shortName, fullName string) (*Course, error)
}

func (s *Server) AddCourse(ac addCourse) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var course Course
		err := json.NewDecoder(r.Body).Decode(&course)
		if err != nil {
			s.logger.Warnw("failed to decode course from body",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"err", err,
			)
			s.errorMessage(
				http.StatusBadRequest,
				"We couldn't read the course from the request body.",
				w, r,
			)
			return
		}

		if course.ShortName == "" || course.FullName == "" {
			s.logger.Warnw("received incomlete course",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"course", course,
			)
			s.errorMessage(
				http.StatusBadRequest,
				"It looks like you missed some fields in the course!",
				w, r,
			)
			return
		}

		newCourse, err := ac.AddCourse(r.Context(), course.ShortName, course.FullName)
		if err != nil {
			s.logger.Errorw("failed to create course",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		s.logger.Infow("created course",
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"course_id", newCourse.ID,
			"email", r.Context().Value(emailContextKey).(string),
		)
		s.sendResponse(http.StatusCreated, newCourse, w, r)
	}
}

const defaultQueueSchedule = "cccccccccccccccccccccccccccccccccccccccccccccccc"

var defaultAppointmentSchedule = &AppointmentSchedule{
	Duration: 15,
	Padding:  2,
	Schedule: "100000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
}

type addQueue interface {
	AddQueue(ctx context.Context, course ksuid.KSUID, queue *Queue) (*Queue, error)
	AddQueueSchedule(ctx context.Context, queue ksuid.KSUID, day int, schedule string) error
	AddAppointmentSchedule(ctx context.Context, queue ksuid.KSUID, day int, schedule *AppointmentSchedule) error
}

func (s *Server) AddQueue(aq addQueue) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := r.Context().Value(courseContextKey).(*Course)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"course_id", c.ID,
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

		if queue.Type != Ordered && queue.Type != Appointments {
			l.Warnw("got unknown queue type", "type", queue.Type)
			s.errorMessage(
				http.StatusBadRequest,
				fmt.Sprintf(`I haven't seen the queue type "%s" before.`, queue.Type),
				w, r,
			)
			return
		}

		newQueue, err := aq.AddQueue(r.Context(), c.ID, &queue)
		if err != nil {
			l.Errorw("failed to create queue", "err", err)
			s.internalServerError(w, r)
			return
		}
		l.Infow("created queue",
			"queue_id", newQueue.ID,
			"email", r.Context().Value(emailContextKey).(string),
		)

		for day := 0; day < 7; day++ {
			err = aq.AddQueueSchedule(r.Context(), newQueue.ID, day, defaultQueueSchedule)
			if err != nil {
				l.Errorw("failed to add default queue schedule",
					"day", day,
					"err", err,
				)
				s.internalServerError(w, r)
				return
			}

			if queue.Type == Appointments {
				err = aq.AddAppointmentSchedule(r.Context(), newQueue.ID, day, defaultAppointmentSchedule)
				if err != nil {
					l.Errorw("failed to add default appointment schedule",
						"day", day,
						"err", err,
					)
					s.internalServerError(w, r)
					return
				}
			}
		}

		s.sendResponse(http.StatusCreated, newQueue, w, r)
	}
}

type getCourseAdmins interface {
	GetCourseAdmins(ctx context.Context, course ksuid.KSUID) ([]string, error)
}

func (s *Server) GetCourseAdmins(ga getCourseAdmins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := r.Context().Value(courseContextKey).(*Course)

		admins, err := ga.GetCourseAdmins(r.Context(), c.ID)
		if err != nil {
			s.logger.Errorw("failed to get course admins",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"course_id", c.ID,
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		s.sendResponse(http.StatusOK, admins, w, r)
	}
}

type addCourseAdmins interface {
	AddCourseAdmins(ctx context.Context, course ksuid.KSUID, admins []string, overwrite bool) error
}

func (s *Server) AddCourseAdmins(aa addCourseAdmins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := r.Context().Value(courseContextKey).(*Course)
		email := r.Context().Value(emailContextKey).(string)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"course_id", c.ID,
			"email", email,
		)

		var admins []string
		err := json.NewDecoder(r.Body).Decode(&admins)
		if err != nil {
			l.Warnw("failed to decode admins from body", "err", err)
			s.errorMessage(
				http.StatusBadRequest,
				"I couldn't decode the body. Are you sure it's a JSON array of emails (strings)? This error might help: "+err.Error(),
				w, r,
			)
			return
		}

		err = aa.AddCourseAdmins(r.Context(), c.ID, admins, false)
		var pqerr *pq.Error
		if errors.As(err, &pqerr) && pqerr.Code == "23505" {
			l.Warnw("site admin attempted to add already existing course admin", "err", err)
			s.errorMessage(
				http.StatusBadRequest,
				"It looks like one of the admins you attempted to insert is already an admin. No admins were inserted. Unfortunately I don't know more.",
				w, r,
			)
			return
		} else if err != nil {
			l.Errorw("failed to update course admins", "err", err)
			s.internalServerError(w, r)
			return
		}

		l.Infow("added admins", "admins", admins)
		s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}

func (s *Server) UpdateCourseAdmins(aa addCourseAdmins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := r.Context().Value(courseContextKey).(*Course)
		email := r.Context().Value(emailContextKey).(string)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"course_id", c.ID,
			"email", email,
		)

		var admins []string
		err := json.NewDecoder(r.Body).Decode(&admins)
		if err != nil {
			l.Warnw("failed to decode admins from body", "err", err)
			s.errorMessage(
				http.StatusBadRequest,
				"I couldn't decode the body. Are you sure it's a JSON array of emails (strings)? This error might help: "+err.Error(),
				w, r,
			)
			return
		}

		err = aa.AddCourseAdmins(r.Context(), c.ID, admins, true)
		if err != nil {
			l.Errorw("failed to update course admins", "err", err)
			s.internalServerError(w, r)
			return
		}

		l.Infow("overwrote admins", "admins", admins)
		s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}

type removeCourseAdmins interface {
	RemoveCourseAdmins(ctx context.Context, course ksuid.KSUID, admins []string) error
}

func (s *Server) RemoveCourseAdmins(ra removeCourseAdmins) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c := r.Context().Value(courseContextKey).(*Course)
		email := r.Context().Value(emailContextKey).(string)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"course_id", c.ID,
			"email", email,
		)

		var admins []string
		err := json.NewDecoder(r.Body).Decode(&admins)
		if err != nil {
			l.Warnw("failed to decode admins from body", "err", err)
			s.errorMessage(
				http.StatusBadRequest,
				"I couldn't decode the body. Are you sure it's a JSON array of emails (strings)? This error might help: "+err.Error(),
				w, r,
			)
			return
		}

		err = ra.RemoveCourseAdmins(r.Context(), c.ID, admins)
		if err != nil {
			l.Errorw("failed to remove admins", "err", err)
			s.internalServerError(w, r)
			return
		}

		l.Infow("removed admins", "admins", admins)
		s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}
