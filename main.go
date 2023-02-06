package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"net/http"
	"os"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog"
	"github.com/go-chi/render"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
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

	init := flag.Bool("init", false, "")
	flag.Parse()

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

	if *init {
		m.initData()
		return
	}

	ctx := context.Background()

	oplog := httplog.LogEntry(ctx)
	/* jsonify logging */
	httpLogger := httplog.NewLogger(appName, httplog.Options{JSON: true, LevelFieldName: "severity", Concise: true})

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Timeout(60 * time.Second))
	r.Use(httplog.RequestLogger(httpLogger))

	r.Get("/ping", func(w http.ResponseWriter, r *http.Request) {
		render.JSON(w, r, map[string]string{"message": "pong"})
	})

	r.Route("/api", func(s chi.Router) {
		s.Get("/search-albums-of-singer/{lastName}", m.getAlbumInfo)
		s.Post("/create-album-for-singer", m.createAlbum)
	})

	if err := http.ListenAndServe(":"+servicePort, r); err != nil {
		oplog.Err(err)
	}
}

var errorRender = func(w http.ResponseWriter, r *http.Request, httpCode int, err error) {
	render.Status(r, httpCode)
	render.JSON(w, r, map[string]interface{}{"ERROR": err.Error()})
}

type SingerAlbum struct {
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	AlbumName string `json:"album_name,omitempty"`
}

func (m *Music) createAlbum(w http.ResponseWriter, r *http.Request) {

	postData := SingerAlbum{}
	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&postData); err != nil {
		errorRender(w, r, 500, err)
	}
	defer r.Body.Close()

	if err := m.db.Transaction(func(tx *gorm.DB) error {
		singerId, err := CreateSinger(m.db, postData.FirstName, postData.LastName)
		if err != nil {
			errorRender(w, r, 500, err)
		}
		_, err = CreateAlbumWithRandomTracks(m.db, singerId, postData.AlbumName, randInt(1, 22))
		if err != nil {
			errorRender(w, r, 500, err)
		}
		return nil
	}); err != nil {
		errorRender(w, r, 500, err)
	}
	render.JSON(w, r, struct{}{})
}

func (m *Music) getAlbumInfo(w http.ResponseWriter, r *http.Request) {
	var singers []*Singer
	lastName := chi.URLParam(r, "lastName")
	if err := m.db.Model(&Singer{}).Preload(clause.Associations).Where("last_name = ?", lastName).Debug().Find(&singers).Error; err != nil {
		errorRender(w, r, 500, err)
	}
	if len(singers) == 0 {
		errorRender(w, r, 404, errors.New("user not found"))
	}
	render.JSON(w, r, singers)
}

func (m *Music) initData() {
	CreateRandomSingersAndAlbums(m.db)
}
