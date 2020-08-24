package main

import (
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/CarsonHoffman/office-hours-queue/server/api"
	"github.com/CarsonHoffman/office-hours-queue/server/db"
	"github.com/go-chi/chi"
	"go.uber.org/zap"
)

func main() {
	z, _ := zap.NewProduction()
	l := z.Sugar().With("name", "queue")

	password, err := ioutil.ReadFile(os.Getenv("QUEUE_DB_PASSWORD_FILE"))
	if err != nil {
		l.Fatalw("failed to load DB password file", "err", err)
	}

	db, err := db.New(os.Getenv("QUEUE_DB_URL"), os.Getenv("QUEUE_DB_DATABASE"), os.Getenv("QUEUE_DB_USERNAME"), string(password))
	if err != nil {
		l.Fatalw("failed to set up database", "err", err)
	}

	s := api.New(db, l, db.DB.DB)

	r := chi.NewRouter()
	r.Mount("/", s)

	go func() {
		l.Fatalw("http server failed", "err", http.ListenAndServe(":8080", r))
	}()

	for range time.Tick(30 * time.Second) {
		err = db.MetricsReport(l)
		if err != nil {
			l.Errorw("failed to generate queue students report", "err", err)
		}
	}
}
