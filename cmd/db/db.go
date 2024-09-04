package db

import (
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Message struct {
	MessageId     int64  `gorm:"primaryKey;not null"`
	ChatId        int64  `gorm:"not null"`
	Message       string `gorm:"not null"`
	Date          int64  `gorm:"not null"`
	FromFirstName string
	FromLastName  string
}

type PublicKeysForChats struct {
	ChatId    int64 `gorm:"primaryKey;not null;unique"`
	PublicKey string
}

type Tabler interface {
	TableName() string
}

func (Message) TableName() string {
	return "Summarize_Messages"
}

func (PublicKeysForChats) TableName() string {
	return "Summarize_Keys"
}

func Init(pgUrl string) *gorm.DB {

	err := os.MkdirAll("./db", os.ModePerm)
	if err != nil {
		panic("failed to make ./db directory")
	}

	var dbConnection gorm.Dialector

	if len(pgUrl) > 0 {
		dbConnection = postgres.Open(pgUrl)
	} else {
		dbConnection = sqlite.Open("./db/messages.db")
	}

	db, err := gorm.Open(dbConnection, &gorm.Config{})
	if err != nil {
		panic("failed to connect database")
	}

	db.AutoMigrate(&Message{})
	db.AutoMigrate(&PublicKeysForChats{})

	return db
}
