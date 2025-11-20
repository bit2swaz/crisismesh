package store

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

func Init(path string) (*gorm.DB, error) {
	// Enable WAL mode and busy timeout for better concurrency
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"

	// Disable GORM logging to stdout to prevent TUI corruption
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
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
