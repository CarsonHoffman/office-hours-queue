package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/segmentio/ksuid"
)

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
			Message: "Oops! Something bad happened on our end. If this is happening consistently, please get in touch with us, and include the following ID: " +
				r.Context().Value(RequestIDContextKey).(ksuid.KSUID).String(),
		},
		w, r,
	)
}

func (s *Server) sendResponse(code int, data interface{}, w http.ResponseWriter, r *http.Request) {
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
			return
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

// BigTime returns (roughly) the maximum time representable by PostgreSQL.
// It might be off by a bit. They can deal with that in 294276.
func BigTime() time.Time {
	return time.Date(294276, 0, 0, 0, 0, 0, 0, time.UTC)
}
