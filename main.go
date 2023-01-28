package main

import (
	"context"
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
	Client GameUserOperation
}

type User struct {
	Name string `json:"name"`
	Id   string `json:"id"`
}

func main() {

	var servicePort string = os.Getenv("PORT")

	ctx := context.Background()

	client, err := newClient(ctx, spannerPgString)
	if err != nil {
		log.Fatal(err)
	}
	defer func() {
		db, _ := client.sc.DB()
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

	r.Get("/ping", s.pingPong)

	r.Get("/api/user_id/{user_id:[a-z0-9-.]+}", s.getUserItems)

	r.Post("/api/user/{user_name:[a-z0-9-.]+}", s.createUser)

	r.Put("/api/user_id/{user_id:[a-z0-9-.]+}/{item_id:[a-z0-9-.]+}", s.addItemToUser)

	if err := http.ListenAndServe(":"+servicePort, r); err != nil {
		oplog.Err(err)
	}

}

var errorRender = func(w http.ResponseWriter, r *http.Request, httpCode int, err error) {
	render.Status(r, httpCode)
	render.JSON(w, r, map[string]interface{}{"ERROR": err.Error()})
}

func (s Serving) getUserItems(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	ctx := r.Context()
	results, err := s.Client.getUserItems(ctx, w, userID)
	if err != nil {
		errorRender(w, r, http.StatusInternalServerError, err)
		return
	}
	render.JSON(w, r, results)
}

func (s Serving) createUser(w http.ResponseWriter, r *http.Request) {
	userName := chi.URLParam(r, "user_name")
	ctx := r.Context()
	id, err := s.Client.createUser(ctx, w, userName)
	if err != nil {
		errorRender(w, r, http.StatusInternalServerError, err)
		return
	}
	render.JSON(w, r, User{
		Id:   id,
		Name: userName,
	})
}

func (s Serving) addItemToUser(w http.ResponseWriter, r *http.Request) {
	userID := chi.URLParam(r, "user_id")
	itemID := chi.URLParam(r, "item_id")
	ctx := r.Context()
	err := s.Client.addItemToUser(ctx, w, Users{UserID: userID}, ItemParams{ItemID: itemID})
	if err != nil {
		errorRender(w, r, http.StatusInternalServerError, err)
		return
	}
	render.JSON(w, r, map[string]string{})
}

func (s Serving) pingPong(w http.ResponseWriter, r *http.Request) {
	render.Status(r, http.StatusOK)
	render.PlainText(w, r, "Pong!\n")
}
