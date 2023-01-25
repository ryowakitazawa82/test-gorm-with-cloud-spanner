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
	Name  string
	Price int
	Book  []Volume `gorm:"foreignKey:ID"`
}

type Volume struct {
	BaseModel
	Vol   int
	Comic Comic `gorm:"foreignKey:ID"`
}
