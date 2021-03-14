package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/segmentio/ksuid"
)

// E is a custom handler type that supports returning an error.
// The name is short to cut down on repetition since every handler
// needs to be converted to this type---see testing.T, for example.
type E func(w http.ResponseWriter, r *http.Request) error

type StatusError struct {
	status  int
	message string
}

func (s StatusError) Error() string { return s.message }

// A custom handler wrapper to support the use of error-returning
// handlers that have their errors serialized automatically.
func (e E) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := e(w, r)
	if err != nil {
		*(r.Context().Value(RequestErrorContextKey).(*error)) = err
		if errors.Is(err, context.Canceled) {
			// Custom nginx response code indicating client close.
			// The client won't actually receive this (since they
			// closed), but it'll help internal bookkeeping, like
			// in metrics (rather than reporting a 500).
			w.WriteHeader(499)
			return
		}

		m := struct {
			Message string `json:"message"`
		}{}
		var s StatusError
		if !errors.As(err, &s) {
			s = internalServerError(r)
		}
		m.Message = s.message

		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(s.status)
		json.NewEncoder(w).Encode(m)
		return
	}
}

func internalServerError(r *http.Request) StatusError {
	return StatusError{status: http.StatusInternalServerError,
		message: "Oops! Something bad happened on our end. If this is happening consistently, please get in touch with us, and include the following ID: " +
			r.Context().Value(RequestIDContextKey).(ksuid.KSUID).String()}
}

// Types and functions supporting error returns from middleware.

type ErrorMessage struct {
	Message string `json:"message"`
}

func (s *Server) errorMessage(status int, message string, w http.ResponseWriter, r *http.Request) {
	s.sendResponse(
		status,
		ErrorMessage{Message: message},
		w, r,
	)
}

func (s *Server) internalServerError(w http.ResponseWriter, r *http.Request) {
	s.sendResponse(
		http.StatusInternalServerError,
		ErrorMessage{
			Message: internalServerError(r).message,
		},
		w, r,
	)
}

func (s *Server) sendResponse(code int, data interface{}, w http.ResponseWriter, r *http.Request) error {
	var body []byte
	if data != nil {
		w.Header().Add("Content-Type", "application/json")
		var err error
		body, err = json.Marshal(data)
		if err != nil {
			s.logger.Errorw("failed to marshal response",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"err", err,
			)
			w.WriteHeader(http.StatusInternalServerError)
			return err
		}
	}

	w.WriteHeader(code)
	_, err := w.Write(body)
	if err != nil {
		s.logger.Warnw("failed to write response to client",
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"err", err,
		)
	}
	return err
}

func CurrentHalfHour() int {
	return (time.Now().Hour()*60 + time.Now().Minute()) / 30
}

// WeekdayBounds gets the bounds of the specified
// day of the week in the local time zone. start is the first instant
// of the day, and end is the last nanosecond of the day.
// If the value of day is less than the current day, it is
// assumed to represent the day in the next week.
func WeekdayBounds(day int) (start time.Time, end time.Time) {
	currentDay := time.Now().Local().Weekday()
	difference := day - int(currentDay)

	// If difference is negative, it's next week
	if difference < 0 {
		difference += 7
	}

	// Get the absolute day value in the month
	day = time.Now().Local().Day() + difference

	start = time.Date(time.Now().Local().Year(), time.Now().Month(), day, 0, 0, 0, 0, time.Local)
	end = time.Date(time.Now().Local().Year(), time.Now().Month(), day+1, 0, 0, 0, -1, time.Local)
	return
}

// TimeslotToTime converts an appointment timeslot number to its time.
// Takes daylight savings time into account (i.e. it gives the "normal" time,
// rather than just the index of the timeslot in the day in terms of minutes)
func TimeslotToTime(day, timeslot, duration int) time.Time {
	start, _ := WeekdayBounds(day)
	return time.Date(start.Year(), start.Month(), start.Day(), (timeslot*duration)/60, (timeslot*duration)%60, 0, 0, start.Local().Location())
}

// BigTime returns (roughly) the maximum time representable by PostgreSQL.
// It might be off by a bit. They can deal with that in 294276.
func BigTime() time.Time {
	return time.Date(294276, 0, 0, 0, 0, 0, 0, time.UTC)
}
