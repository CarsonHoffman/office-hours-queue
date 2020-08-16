package api

import (
	"context"
	"net/http"

	"github.com/segmentio/ksuid"
)

const RequestIDContextKey = "request_id"

func ksuidInserter(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := ksuid.New()
		ctx := context.WithValue(r.Context(), RequestIDContextKey, id)
		w.Header().Add("X-Request-ID", id.String())
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) sessionRetriever(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := s.sessions.Get(r, "session")
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}

		email, ok := session.Values["email"].(string)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), emailContextKey, email)
		ctx = context.WithValue(ctx, sessionContextKey, session.Values)

		s.logger.Infow("retrieved session",
			RequestIDContextKey, r.Context().Value(RequestIDContextKey),
			"email", email,
		)

		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) recoverMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if msg := recover(); msg != nil {
				s.logger.Errorw("recovered panic",
					RequestIDContextKey, r.Context().Value(RequestIDContextKey),
					"panic_message", msg,
				)
				s.internalServerError(w, r)
			}
		}()

		next.ServeHTTP(w, r)
	})
}

type siteAdmin interface {
	SiteAdmin(ctx context.Context, email string) (bool, error)
}

func (s *Server) EnsureSiteAdmin(sa siteAdmin) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			email := r.Context().Value(emailContextKey).(string)
			admin, err := sa.SiteAdmin(r.Context(), email)
			if err != nil || !admin {
				s.logger.Warnw("non-admin attempting to access resource requiring site admin",
					RequestIDContextKey, r.Context().Value(RequestIDContextKey),
					"email", email,
				)
				s.errorMessage(
					http.StatusForbidden,
					"You're not supposed to be here. :)",
					w, r,
				)
				return
			}

			s.logger.Infow("entering site admin context",
				RequestIDContextKey, r.Context().Value(RequestIDContextKey),
				"email", email,
			)
			next.ServeHTTP(w, r)
		})
	}
}
