package main

import "time"

type BaseModel struct {
	ID        string `gorm:"primaryKey;autoIncrement:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Author struct {
	BaseModel
	Name string
	Age  int
}

type Comic struct {
	BaseModel
	Name   string
	Volume []Volume `gorm:"foreignKey:ID"`
}

type Volume struct {
	BaseModel
	Vol         int
	Price       int
	PublishDate time.Time
	Comic       Comic `gorm:"foreignKey:ID"`
}
