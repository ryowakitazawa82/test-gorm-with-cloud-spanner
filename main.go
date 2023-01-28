package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog"
	"github.com/go-chi/render"
)

var appName = "myapp"

// like, export CONNECTION_STRING="host=localhost port=15432 database=musics"
var spannerPgString string = os.Getenv("CONNECTION_STRING")

type Serving struct {
	Client BookOperation
}

func main() {

	var servicePort string = os.Getenv("PORT")

	ctx := context.Background()

	client, err := newClient(ctx, spannerPgString)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		db, _ := client.db.DB()
		db.Close()
	}()

	s := Serving{
		Client: client,
	}

	oplog := httplog.LogEntry(context.Background())
	/* jsonify logging */
	httpLogger := httplog.NewLogger(appName, httplog.Options{JSON: true, LevelFieldName: "severity", Concise: true})

	r := chi.NewRouter()
	// r.Use(middleware.Throttle(8))
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(httplog.RequestLogger(httpLogger))

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, map[string]string{"message": "pong"})
	})

	r.Post("/api/author/{user_name:[a-z0-9-.]+}", s.registerAuthor)

	if err := http.ListenAndServe(":"+servicePort, r); err != nil {
		oplog.Err(err)
	}

}

var errorRender = func(w http.ResponseWriter, r *http.Request, httpCode int, err error) {
	render.Status(r, httpCode)
	render.JSON(w, r, map[string]interface{}{"ERROR": err.Error()})
}

func (s Serving) registerAuthor(w http.ResponseWriter, r *http.Request) {

	p := map[string]string{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&p); err != nil {
		errorRender(w, r, 500, err)
	}

	defer r.Body.Close()

	birthDate := p["birth_date"]
	userName := chi.URLParam(r, "user_name")

	birthTime, err := time.Parse("2006-01-02", birthDate)
	if err != nil {
		errorRender(w, r, 500, err)
	}
	ctx := r.Context()
	author := Author{
		Name:      userName,
		BirthDate: birthTime,
	}
	id, err := s.Client.registerAuthor(ctx, w, author)
	if err != nil {
		errorRender(w, r, http.StatusInternalServerError, err)
		return
	}
	res := map[string]string{"id": id, "username": userName}
	render.JSON(w, r, res)
}
