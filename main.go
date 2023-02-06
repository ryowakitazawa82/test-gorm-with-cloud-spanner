package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	"flag"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httplog"
	"github.com/go-chi/render"
	"gorm.io/datatypes"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"gorm.io/gorm/clause"
)

var appName = "sampleapp"

// like, export CONNECTION_STRING="host=localhost port=15432 database=musics"
var connString string = os.Getenv("CONNECTION_STRING")
var servicePort string = os.Getenv("PORT")

type Music struct {
	db *gorm.DB
}

func main() {

	migrateMode := flag.Bool("automigrate", false, "")
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

	if *migrateMode {
		doAutoMigrate(db)
		return
	}

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
		s.Get("/search-album", m.SearchAlbumsUsingNamedArgument)
		s.Get("/search-album-with-singer/{lastName}", m.getAlbumInfo)
	})

	if err := http.ListenAndServe(":"+servicePort, r); err != nil {
		oplog.Err(err)
	}

}

var errorRender = func(w http.ResponseWriter, r *http.Request, httpCode int, err error) {
	render.Status(r, httpCode)
	render.JSON(w, r, map[string]interface{}{"ERROR": err.Error()})
}

func (m *Music) getAlbumInfo(w http.ResponseWriter, r *http.Request) {
	var singers []*Singer
	lastName := chi.URLParam(r, "lastName")
	if err := m.db.Debug().Model(&Singer{}).Preload(clause.Associations).Where("last_name = ?", lastName).Find(&singers).Error; err != nil {
		errorRender(w, r, 500, err)
	}
	if len(singers) == 0 {
		errorRender(w, r, 404, errors.New("User not found"))
	}
	render.JSON(w, r, singers)
}

func (m *Music) CreateRandomSingersAndAlbums(w http.ResponseWriter, r *http.Request) {
	CreateRandomSingersAndAlbums(m.db)
	render.JSON(w, r, struct{}{})
}

func (m *Music) SearchAlbumsUsingNamedArgument(w http.ResponseWriter, r *http.Request) {
	log.Println("Searching for albums released before 1900")
	var albums []*Album
	if err := m.db.Where(
		"release_date < ?",
		datatypes.Date(time.Date(1900, time.January, 1, 0, 0, 0, 0, time.UTC)),
	).Order("release_date asc").Find(&albums).Error; err != nil {
		fmt.Printf("Failed to load albums: %v", err)
		errorRender(w, r, 500, err)
	}
	if len(albums) == 0 {
		errorRender(w, r, 500, errors.New("album not found"))
	} else {
		for _, album := range albums {
			log.Printf("Album %q was released at %v\n", album.Title, time.Time(album.ReleaseDate).Format("2006-01-02"))
		}
		render.JSON(w, r, albums)
	}
}

func doAutoMigrate(db *gorm.DB) {
	db.AutoMigrate(&Singer{}, &Album{}, &Track{}, &Venue{}, &Concert{})
}
