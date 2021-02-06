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

const courseAdminContextKey = "course_admin"

type courseAdmin interface {
	CourseAdmin(ctx context.Context, course ksuid.KSUID, email string) (bool, error)
}

func (s *Server) CheckCourseAdmin(ca courseAdmin) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var courseID ksuid.KSUID
			course, ok := r.Context().Value(courseContextKey).(*Course)
			if ok {
				courseID = course.ID
			} else {
				q := r.Context().Value(queueContextKey).(*Queue)
				courseID = q.Course
			}

			email, ok := r.Context().Value(emailContextKey).(string)
			if !ok {
				ctx := context.WithValue(r.Context(), courseAdminContextKey, false)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}

			admin, err := ca.CourseAdmin(r.Context(), courseID, email)
			if err != nil {
				s.logger.Errorw("failed to check course admin status",
					RequestIDContextKey, r.Context().Value(RequestIDContextKey),
					"course_id", courseID,
					"email", email,
					"err", err,
				)
				s.internalServerError(w, r)
				return
			}

			ctx := context.WithValue(r.Context(), courseAdminContextKey, admin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func (s *Server) EnsureCourseAdmin(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var courseID ksuid.KSUID
		course, ok := r.Context().Value(courseContextKey).(*Course)
		if ok {
			courseID = course.ID
		} else {
			q := r.Context().Value(queueContextKey).(*Queue)
			courseID = q.Course
		}

		email := r.Context().Value(emailContextKey).(string)
		admin := r.Context().Value(courseAdminContextKey).(bool)
		if !admin {
			s.logger.Warnw("non-admin attempting to access resource requiring course admin",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"course_id", courseID,
				"email", email,
			)
			s.errorMessage(
				http.StatusForbidden,
				"You shouldn't be here. :)",
				w, r,
			)
			return
		}

		s.logger.Infow("entering course admin context",
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"course_id", courseID,
			"email", email,
		)
		next.ServeHTTP(w, r)
	})
}

type getCourses interface {
	GetCourses(context.Context) ([]*Course, error)
}

func (s *Server) GetCourses(gc getCourses) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		courses, err := gc.GetCourses(r.Context())
		if err != nil {
			s.logger.Errorw("failed to fetch courses from DB",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"err", err,
			)
			return err
		}

		return s.sendResponse(http.StatusOK, courses, w, r)
	}
}

func (s *Server) GetCourse(gc getCourse) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		return s.sendResponse(http.StatusOK, r.Context().Value(courseContextKey), w, r)
	}
}

type getQueues interface {
	getCourse
	GetQueues(ctx context.Context, course ksuid.KSUID) ([]*Queue, error)
}

func (s *Server) GetQueues(gq getQueues) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		c := r.Context().Value(courseContextKey).(*Course)

		queues, err := gq.GetQueues(r.Context(), c.ID)
		if err != nil {
			s.logger.Errorw("failed to get queues from course",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"course_id", c.ID,
				"err", err,
			)
			return err
		}

		return s.sendResponse(http.StatusOK, queues, w, r)
	}
}

type addCourse interface {
	AddCourse(ctx context.Context, shortName, fullName string) (*Course, error)
}

func (s *Server) AddCourse(ac addCourse) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		var course Course
		err := json.NewDecoder(r.Body).Decode(&course)
		if err != nil {
			s.logger.Warnw("failed to decode course from body",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"err", err,
			)
			return StatusError{
				http.StatusBadRequest,
				"We couldn't read the course from the request body.",
			}
		}

		if course.ShortName == "" || course.FullName == "" {
			s.logger.Warnw("received incomplete course",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"course", course,
			)
			return StatusError{
				http.StatusBadRequest,
				"It looks like you missed some fields in the course!",
			}
		}

		newCourse, err := ac.AddCourse(r.Context(), course.ShortName, course.FullName)
		if err != nil {
			s.logger.Errorw("failed to create course",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"err", err,
			)
			return err
		}

		s.logger.Infow("created course",
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"course_id", newCourse.ID,
			"email", r.Context().Value(emailContextKey).(string),
		)
		return s.sendResponse(http.StatusCreated, newCourse, w, r)
	}
}

type updateCourse interface {
	UpdateCourse(ctx context.Context, course ksuid.KSUID, shortName, fullName string) error
}

func (s *Server) UpdateCourse(uc updateCourse) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		course := r.Context().Value(courseContextKey).(*Course)

		var bodyCourse Course
		err := json.NewDecoder(r.Body).Decode(&bodyCourse)
		if err != nil {
			s.logger.Warnw("failed to decode course from body",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"err", err,
			)
			return StatusError{
				http.StatusBadRequest,
				"We couldn't read the course from the request body.",
			}
		}

		if bodyCourse.ShortName == "" || bodyCourse.FullName == "" {
			s.logger.Warnw("received incomplete course",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"course", bodyCourse,
			)
			return StatusError{
				http.StatusBadRequest,
				"It looks like you missed some fields in the course!",
			}
		}

		err = uc.UpdateCourse(r.Context(), course.ID, bodyCourse.ShortName, bodyCourse.FullName)
		if err != nil {
			s.logger.Errorw("failed to update course",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"err", err,
			)
			return err
		}

		s.logger.Infow("updated course",
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"course_id", course.ID,
			"email", r.Context().Value(emailContextKey).(string),
		)
		return s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}

type deleteCourse interface {
	DeleteCourse(ctx context.Context, course ksuid.KSUID) error
}

func (s *Server) DeleteCourse(dc deleteCourse) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		course := r.Context().Value(courseContextKey).(*Course)

		err := dc.DeleteCourse(r.Context(), course.ID)
		if err != nil {
			s.logger.Errorw("failed to delete course",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"email", r.Context().Value(emailContextKey).(string),
				"err", err,
			)
			return err
		}

		s.logger.Infow("deleted course",
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"course_id", course.ID,
			"email", r.Context().Value(emailContextKey).(string),
		)
		return s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}

const defaultQueueSchedule = "cccccccccccccccccccccccccccccccccccccccccccccccc"

var defaultAppointmentSchedule = &AppointmentSchedule{
	Duration: 15,
	Padding:  2,
	Schedule: "000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
}

type addQueue interface {
	AddQueue(ctx context.Context, course ksuid.KSUID, queue *Queue) (*Queue, error)
	AddQueueSchedule(ctx context.Context, queue ksuid.KSUID, day int, schedule string) error
	AddAppointmentSchedule(ctx context.Context, queue ksuid.KSUID, day int, schedule *AppointmentSchedule) error
}

func (s *Server) AddQueue(aq addQueue) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		c := r.Context().Value(courseContextKey).(*Course)
		l := s.logger.With(
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"course_id", c.ID,
		)

		var queue Queue
		err := json.NewDecoder(r.Body).Decode(&queue)
		if err != nil {
			l.Warnw("failed to decode queue from body", "err", err)
			return StatusError{
				http.StatusBadRequest,
				"We couldn't read the queue from the request body.",
			}
		}

		if queue.Name == "" {
			l.Warnw("got incomplete queue", "queue", queue)
			return StatusError{
				http.StatusBadRequest,
				"It looks like you missed some fields in the queue!",
			}
		}

		if queue.Type != Ordered && queue.Type != Appointments {
			l.Warnw("got unknown queue type", "type", queue.Type)
			return StatusError{
				http.StatusBadRequest,
				fmt.Sprintf(`I haven't seen the queue type "%s" before.`, queue.Type),
			}
		}

		newQueue, err := aq.AddQueue(r.Context(), c.ID, &queue)
		if err != nil {
			l.Errorw("failed to create queue", "err", err)
			return err
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
				return err
			}

			if queue.Type == Appointments {
				err = aq.AddAppointmentSchedule(r.Context(), newQueue.ID, day, defaultAppointmentSchedule)
				if err != nil {
					l.Errorw("failed to add default appointment schedule",
						"day", day,
						"err", err,
					)
					return err
				}
			}
		}

		return s.sendResponse(http.StatusCreated, newQueue, w, r)
	}
}

type getCourseAdmins interface {
	GetCourseAdmins(ctx context.Context, course ksuid.KSUID) ([]string, error)
}

func (s *Server) GetCourseAdmins(ga getCourseAdmins) E {
	return func(w http.ResponseWriter, r *http.Request) error {
		c := r.Context().Value(courseContextKey).(*Course)

		admins, err := ga.GetCourseAdmins(r.Context(), c.ID)
		if err != nil {
			s.logger.Errorw("failed to get course admins",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"course_id", c.ID,
				"err", err,
			)
			return err
		}

		return s.sendResponse(http.StatusOK, admins, w, r)
	}
}

type addCourseAdmins interface {
	AddCourseAdmins(ctx context.Context, course ksuid.KSUID, admins []string, overwrite bool) error
}

func (s *Server) AddCourseAdmins(aa addCourseAdmins) E {
	return func(w http.ResponseWriter, r *http.Request) error {
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
			return StatusError{
				http.StatusBadRequest,
				"I couldn't decode the body. Are you sure it's a JSON array of emails (strings)? This error might help: " + err.Error(),
			}
		}

		err = aa.AddCourseAdmins(r.Context(), c.ID, admins, false)
		var pqerr *pq.Error
		if errors.As(err, &pqerr) && pqerr.Code == "23505" {
			l.Warnw("site admin attempted to add already existing course admin", "err", err)
			return StatusError{
				http.StatusBadRequest,
				"It looks like one of the admins you attempted to insert is already an admin. No admins were inserted. Unfortunately I don't know more.",
			}
		} else if err != nil {
			l.Errorw("failed to update course admins", "err", err)
			return err
		}

		l.Infow("added admins", "admins", admins)
		return s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}

func (s *Server) UpdateCourseAdmins(aa addCourseAdmins) E {
	return func(w http.ResponseWriter, r *http.Request) error {
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
			return StatusError{
				http.StatusBadRequest,
				"I couldn't decode the body. Are you sure it's a JSON array of emails (strings)? This error might help: " + err.Error(),
			}
		}

		err = aa.AddCourseAdmins(r.Context(), c.ID, admins, true)
		if err != nil {
			l.Errorw("failed to update course admins", "err", err)
			return err
		}

		l.Infow("overwrote admins", "admins", admins)
		return s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}

type removeCourseAdmins interface {
	RemoveCourseAdmins(ctx context.Context, course ksuid.KSUID, admins []string) error
}

func (s *Server) RemoveCourseAdmins(ra removeCourseAdmins) E {
	return func(w http.ResponseWriter, r *http.Request) error {
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
			return StatusError{
				http.StatusBadRequest,
				"I couldn't decode the body. Are you sure it's a JSON array of emails (strings)? This error might help: " + err.Error(),
			}
		}

		err = ra.RemoveCourseAdmins(r.Context(), c.ID, admins)
		if err != nil {
			l.Errorw("failed to remove admins", "err", err)
			return err
		}

		l.Infow("removed admins", "admins", admins)
		return s.sendResponse(http.StatusNoContent, nil, w, r)
	}
}
