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
	addComicAndVolume(context.Context, io.Writer, Comic, Volume) error
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
	randomId := genId()

	author.ID = randomId
	res := d.db.Debug().Create(&author)
	if res.Error != nil {
		return "", res.Error
	}

	return randomId, nil
}

func (d dbClient) addComicAndVolume(ctx context.Context, w io.Writer, comic Comic, volume Volume) error {
	if err := d.db.Transaction(func(tx *gorm.DB) error {
		if res := tx.Create(&comic); res.Error != nil {
			return res.Error
		}
		if res := tx.Create(&volume); res.Error != nil {
			return res.Error
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}
