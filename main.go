package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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
var logLevel logger.LogLevel = logger.Info // for debug

var maxRetry = 10

type MusicDbOperation struct {
	db *gorm.DB
}

func newDbConn(connString string, logMode logger.LogLevel) (*gorm.DB, error) {
	log.Println("connString ", connString)
	for i := 0; i < maxRetry; i++ {
		db, err := gorm.Open(postgres.Open(connString), &gorm.Config{
			DisableNestedTransaction: true,
			Logger:                   logger.Default.LogMode(logMode),
		})
		if err != nil {
			log.Println(" Retrying...", i+1, " ", err)
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

	db, err := newDbConn(connString, logLevel)

	if err != nil {
		panic(err)
	}

	defer func() {
		sqlDB, _ := db.DB()
		sqlDB.Close()
	}()

	m := MusicDbOperation{db: db}

	if *init {
		m.db.Logger = m.db.Logger.LogMode(logger.Error)
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

func (m MusicDbOperation) createSingerAlbum(w http.ResponseWriter, r *http.Request) {

	type SingerAlbumInfo struct {
		FirstName string `json:"first_name"`
		LastName  string `json:"last_name"`
		AlbumName string `json:"album_name"`
	}

	postData := SingerAlbumInfo{}

	decoder := json.NewDecoder(r.Body)
	if err := decoder.Decode(&postData); err != nil {
		err = fmt.Errorf("lack of some parameters: error %w", err)
		errorRender(w, r, 500, err)
		return
	}
	defer r.Body.Close()

	var (
		newSingerId string
		newAlbumId  string
	)
	if err := m.db.Transaction(func(tx *gorm.DB) error {
		singerId, err := CreateSinger(tx, postData.FirstName, postData.LastName)
		if err != nil {
			return err
		}
		albumId, err := CreateAlbumWithRandomTracks(tx, singerId, postData.AlbumName, randInt(1, 22))
		if err != nil {
			return err
		}
		newSingerId = singerId
		newAlbumId = albumId
		return nil
	}); err != nil {
		errorRender(w, r, 500, err)
		return
	}
	render.JSON(w, r, map[string]string{"singer_id": newSingerId, "album_id": newAlbumId})
}

func (m MusicDbOperation) getAlbumInfoWithSingerId(w http.ResponseWriter, r *http.Request) {
	var albums []*Album
	singerId := chi.URLParam(r, "singerId")
	if err := m.db.Model(&Album{}).Preload(clause.Associations).
		Where("singer_id = ?", singerId).Find(&albums).Error; err != nil {
		errorRender(w, r, 500, err)
	}
	if len(albums) == 0 {
		errorRender(w, r, 404, errors.New("user not found"))
	}
	render.JSON(w, r, albums)
}

func (m MusicDbOperation) initData() {
	CreateRandomSingersAndAlbums(m.db)
}
