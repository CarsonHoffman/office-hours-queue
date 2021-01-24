package api

import (
	"context"
	"net/http"

	"github.com/jmoiron/sqlx"
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

type transactioner interface {
	BeginTx() (*sqlx.Tx, error)
}

const (
	RequestErrorContextKey = "request_error"
	TransactionContextKey  = "transaction"
)

// This function does tie the API package to sqlx to an extent, but it
// doesn't need to be used in tests (individual handlers can still be
// unit tested without this middleware, since the transaction is passed
// through transparently in the context). I'm not advocating that this is
// the cleanest pattern, but we definitely need to get transactions into
// each request.
func (s *Server) transaction(tr transactioner) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tx, err := tr.BeginTx()
			if err != nil {
				s.internalServerError(w, r)
				return
			}

			// Yes, this is a pointer to an interface. Yes, having handlers
			// propogate information back up via context is probably not the
			// best pattern, but go-chi doesn't directly support handlers and
			// middleware returning errors, and this only needs to occur in one
			// other place (E.ServeHTTP).
			ctx := context.WithValue(r.Context(), RequestErrorContextKey, &err)
			ctx = context.WithValue(ctx, TransactionContextKey, tx)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)

			// err might have been mutated by the handler since we passed the
			// context a pointer to it.
			if err != nil {
				err = tx.Rollback()
				// The handler already wrote a status code, so the best we can
				// do is log the failed rollback.
				if err != nil {
					s.logger.Errorw("transaction rollback failed",
						RequestIDContextKey, r.Context().Value(RequestIDContextKey),
						"err", err,
					)
				}
				return
			}

			err = tx.Commit()
			if err != nil {
				// The handler already wrote a status code, so the best we can
				// do is log the failed commit.
				s.logger.Errorw("transaction commit failed",
					RequestIDContextKey, r.Context().Value(RequestIDContextKey),
					"err", err,
				)
			}
		})
	}
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

		profilePicture, ok := session.Values["profile_pic"].(string)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		name, ok := session.Values["name"].(string)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		firstName, ok := session.Values["first_name"].(string)
		if !ok {
			next.ServeHTTP(w, r)
			return
		}

		ctx := context.WithValue(r.Context(), emailContextKey, email)
		ctx = context.WithValue(ctx, profilePictureContextKey, profilePicture)
		ctx = context.WithValue(ctx, nameContextKey, name)
		ctx = context.WithValue(ctx, firstNameContextKey, firstName)
		ctx = context.WithValue(ctx, sessionContextKey, session.Values)

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
