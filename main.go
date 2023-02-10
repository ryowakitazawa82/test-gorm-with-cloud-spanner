package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"log"
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

// like, CONNECTION_STRING="host=localhost port=5432"
var connString string = os.Getenv("CONNECTION_STRING")
var servicePort string = os.Getenv("PORT")
var maxRetry = 10

type MusicOperation struct {
	db *gorm.DB
}

func newDbConn(connString string) (*gorm.DB, error) {
	log.Println("connString ", connString)
	for i := 0; i < maxRetry; i++ {
		db, err := gorm.Open(postgres.Open(connString), &gorm.Config{
			DisableNestedTransaction: true,
			Logger:                   logger.Default.LogMode(logger.Error),
		})
		if err != nil {
			log.Println(err, " Retrying...", i+1)
			time.Sleep(time.Second * 2)
			continue
		}
		return db, nil
	}
	return nil, errors.New("connection failure")
}

func main() {

	init := flag.Bool("init", false, "Generate initial data")
	flag.Parse()

	db, err := newDbConn(connString)

	if err != nil {
		panic(err)
	}

	defer func() {
		db, _ := db.DB()
		db.Close()
	}()

	m := MusicOperation{db: db}

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
		s.Get("/get-albums-of-singerid/{singerId}", m.getAlbumInfoWithSingerId)
		s.Post("/register-singer-with-album", m.createSingerAlbum)
	})

	if servicePort != "" {
		servicePort = "8080"
	}

	if err := http.ListenAndServe(":"+servicePort, r); err != nil {
		oplog.Err(err)
	}
}

var errorRender = func(w http.ResponseWriter, r *http.Request, httpCode int, err error) {
	render.Status(r, httpCode)
	render.JSON(w, r, map[string]interface{}{"ERROR": err.Error()})
}

func (m MusicOperation) createSingerAlbum(w http.ResponseWriter, r *http.Request) {

	type SingerAlbum struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		AlbumName string `json:"album_name"`
	}

	postData := SingerAlbum{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&postData); err != nil {
		errorRender(w, r, 500, err)
	}
	defer r.Body.Close()

	var (
		newSingerId string
		newAlbumId  string
	)
	if err := m.db.Transaction(func(tx *gorm.DB) error {
		singerId, err := CreateSinger(m.db, postData.FirstName, postData.LastName)
		if err != nil {
			errorRender(w, r, 500, err)
		}
		albumId, err := CreateAlbumWithRandomTracks(m.db, singerId, postData.AlbumName, randInt(1, 22))
		if err != nil {
			errorRender(w, r, 500, err)
		}
		newSingerId = singerId
		newAlbumId = albumId
		return nil
	}); err != nil {
		errorRender(w, r, 500, err)
	}
	render.JSON(w, r, map[string]string{"singer_id": newSingerId, "album_id": newAlbumId})
}

func (m MusicOperation) getAlbumInfoWithSingerId(w http.ResponseWriter, r *http.Request) {
	var singers []*Singer
	singerId := chi.URLParam(r, "singerId")
	if err := m.db.Model(&Singer{}).Preload(clause.Associations).
		Where("id = ?", singerId).Find(&singers).Error; err != nil {
		errorRender(w, r, 500, err)
	}
	if len(singers) == 0 {
		errorRender(w, r, 404, errors.New("user not found"))
	}
	render.JSON(w, r, singers)
}

func (m MusicOperation) initData() {
	CreateRandomSingersAndAlbums(m.db)
}
