package api

import (
	"context"
	"net/http"
	"os"
	"strings"

	"google.golang.org/api/idtoken"
)

const (
	emailContextKey   = "email"
	sessionContextKey = "session"
	stateLength       = 64
)

var emptySessionCookie = &http.Cookie{
	Name:     "session",
	Value:    "",
	MaxAge:   -1,
	HttpOnly: true,
	Secure:   os.Getenv("USE_SECURE_COOKIES") == "true",
}

func (s *Server) ValidLoginMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := s.sessions.Get(r, "session")
		if err != nil {
			s.logger.Infow("got invalid session",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"err", err,
			)
			http.SetCookie(w, emptySessionCookie)
			s.errorMessage(
				http.StatusUnauthorized,
				"Try logging in again.",
				w, r,
			)
			return
		}

		email, ok := session.Values["email"].(string)
		if !ok {
			s.logger.Infow("request to protected resource without authorization",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			)
			s.errorMessage(
				http.StatusUnauthorized,
				"Come back with a login!",
				w, r,
			)
			return
		}

		domain := os.Getenv("QUEUE_VALID_DOMAIN")
		if !strings.HasSuffix(email, "@"+domain) {
			s.logger.Warnw("found valid session with email outside valid domain",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"valid_domain", domain,
				"email", email,
			)
			s.errorMessage(
				http.StatusUnauthorized,
				"Oh dear, it looks like you don't have an @"+os.Getenv("QUEUE_VALID_DOMAIN")+" account.",
				w, r,
			)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) Login() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session, err := s.sessions.New(r, "session")
		if err != nil {
			s.logger.Errorw("got invalid session on login",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"err", err,
			)
			http.SetCookie(w, emptySessionCookie)
			http.Redirect(w, r, "/api/login", http.StatusTemporaryRedirect)
			return
		}

		token := r.FormValue("idtoken")
		payload, err := idtoken.Validate(r.Context(), token, os.Getenv("QUEUE_OAUTH2_CLIENT_ID"))
		if err != nil {
			s.logger.Warnw("failed to validate token",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"err", err,
			)
			s.errorMessage(
				http.StatusUnauthorized,
				"Something doesn't look quite right with your login.",
				w, r,
			)
			return
		}

		email, ok := payload.Claims["email"].(string)
		if !ok {
			s.logger.Errorw("failed to get email from validated token",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		s.logger.Infow("processed login",
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"email", email,
		)

		session.Values["email"] = email
		s.sessions.Save(r, w, session)
	}
}

type getAdminCourses interface {
	GetAdminCourses(ctx context.Context, email string) ([]string, error)
}

type getUserInfo interface {
	siteAdmin
	getAdminCourses
}

func (s *Server) GetCurrentUserInfo(gi getUserInfo) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		email := r.Context().Value(emailContextKey).(string)

		admin, err := gi.SiteAdmin(r.Context(), email)
		if err != nil {
			s.logger.Errorw("failed to get site admin status",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"email", email,
			)
			s.internalServerError(w, r)
			return
		}

		courses, err := gi.GetAdminCourses(r.Context(), email)
		if err != nil {
			s.logger.Errorw("failed to get admin courses",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"email", email,
				"err", err,
			)
			s.internalServerError(w, r)
			return
		}

		resp := struct {
			Email        string   `json:"email"`
			SiteAdmin    bool     `json:"site_admin"`
			AdminCourses []string `json:"admin_courses"`
		}{email, admin, courses}

		s.sendResponse(http.StatusOK, resp, w, r)
	}
}
