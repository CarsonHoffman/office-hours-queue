package main

import (
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/CarsonHoffman/office-hours-queue/server/api"
	"github.com/CarsonHoffman/office-hours-queue/server/db"
	"github.com/go-chi/chi"
)

func main() {
	password, err := ioutil.ReadFile(os.Getenv("QUEUE_DB_PASSWORD_FILE"))
	if err != nil {
		log.Fatalln("failed to load DB password file:", err)
	}

	db, err := db.New(os.Getenv("QUEUE_DB_URL"), os.Getenv("QUEUE_DB_DATABASE"), os.Getenv("QUEUE_DB_USERNAME"), string(password))
	if err != nil {
		log.Fatalln("failed to set up database:", err)
	}

	s := api.New(db, db.DB.DB)

	r := chi.NewRouter()
	r.Mount("/", s)
	log.Fatalln(http.ListenAndServe(":8080", r))
}
