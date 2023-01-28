package main

import (
	"context"
	"io"

	"github.com/google/uuid"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type BookOperation interface {
	registerAuthor(context.Context, io.Writer, Author) (string, error)
	// addComic(context.Context, io.Writer, Comic) error
	// listComics(context.Context, io.Writer, string) ([]Comic, error)
}

type dbClient struct {
	db *gorm.DB
}

func genId() string {
	newUUID, _ := uuid.NewRandom()
	return newUUID.String()
}

func newClient(ctx context.Context, spannerString string) (dbClient, error) {

	db, err := gorm.Open(postgres.Open(spannerString), &gorm.Config{
		DisableNestedTransaction: true,
		Logger:                   logger.Default.LogMode(logger.Error),
	})

	if err != nil {
		return dbClient{}, err
	}
	return dbClient{
		db: db,
	}, nil
}

func (d dbClient) registerAuthor(ctx context.Context, w io.Writer, author Author) (string, error) {
	return "", nil
}
