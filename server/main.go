package main

import (
	"io/ioutil"
	"net/http"
	"os"

	"github.com/CarsonHoffman/office-hours-queue/server/api"
	"github.com/CarsonHoffman/office-hours-queue/server/db"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
)

func main() {
	z, _ := zap.NewProduction()
	l := z.Sugar().With("name", "queue")

	password, err := ioutil.ReadFile(os.Getenv("QUEUE_DB_PASSWORD_FILE"))
	if err != nil {
		l.Fatalw("failed to load DB password file", "err", err)
	}

	oauthClientSecret, err := ioutil.ReadFile(os.Getenv("QUEUE_OAUTH2_CLIENT_SECRET_FILE"))
	if err != nil {
		l.Fatalw("failed to load OAuth2 client secret file", "err", err)
	}

	config := oauth2.Config{
		Endpoint:     google.Endpoint,
		ClientID:     os.Getenv("QUEUE_OAUTH2_CLIENT_ID"),
		ClientSecret: string(oauthClientSecret),
		RedirectURL:  os.Getenv("QUEUE_OAUTH2_REDIRECT_URI"),
		Scopes:       []string{"openid", "email", "profile"},
	}

	db, err := db.New(os.Getenv("QUEUE_DB_URL"), os.Getenv("QUEUE_DB_DATABASE"), os.Getenv("QUEUE_DB_USERNAME"), string(password))
	if err != nil {
		l.Fatalw("failed to set up database", "err", err)
	}

	s := api.New(db, l, db.DB.DB, config)

	r := chi.NewRouter()
	r.Mount("/", s)

	l.Fatalw("http server failed", "err", http.ListenAndServe(":8080", r))
}
