package store

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func Init(path string) (*gorm.DB, error) {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Peer{}, &Message{}); err != nil {
		return nil, err
	}

	return db, nil
}

func SaveMessage(db *gorm.DB, msg *Message) error {
	return db.Create(msg).Error
}

func GetMessages(db *gorm.DB, limit int) ([]Message, error) {
	var messages []Message
	result := db.Order("timestamp desc").Limit(limit).Find(&messages)
	return messages, result.Error
}

func UpsertPeer(db *gorm.DB, peer Peer) error {
	return db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		UpdateAll: true,
	}).Create(&peer).Error
}
