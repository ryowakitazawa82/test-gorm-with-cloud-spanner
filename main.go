package main

import (
	"context"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog"
	"github.com/go-chi/render"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var appName = "sampleapp"

// like, export CONNECTION_STRING="host=localhost port=15432 database=musics"
var connString string = os.Getenv("CONNECTION_STRING")
var servicePort string = os.Getenv("PORT")

type Music struct {
	db *gorm.DB
}

func main() {

	db, err := gorm.Open(postgres.Open(connString), &gorm.Config{
		DisableNestedTransaction: true,
		Logger:                   logger.Default.LogMode(logger.Error),
	})

	if err != nil {
		panic(err)
	}

	defer func() {
		db, _ := db.DB()
		db.Close()
	}()

	m := Music{db: db}

	ctx := context.Background()

	oplog := httplog.LogEntry(ctx)
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

	r.Route("/api", func(s chi.Router) {
		s.Post("/create-random-simgers-and-albums", m.CreateRandomSingersAndAlbums)
	})

	if err := http.ListenAndServe(":"+servicePort, r); err != nil {
		oplog.Err(err)
	}

}

var errorRender = func(w http.ResponseWriter, r *http.Request, httpCode int, err error) {
	render.Status(r, httpCode)
	render.JSON(w, r, map[string]interface{}{"ERROR": err.Error()})
}

func (m *Music) CreateRandomSingersAndAlbums(w http.ResponseWriter, r *http.Request) {
	CreateRandomSingersAndAlbums(m.db)
	render.JSON(w, r, struct{}{})
}
