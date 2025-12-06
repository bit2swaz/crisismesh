package store

import (
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/logger"
)

func Init(path string) (*gorm.DB, error) {
	dsn := path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, err
	}

	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(1)

	if err := Vacuum(db); err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(&Peer{}, &Message{}); err != nil {
		return nil, err
	}
	return db, nil
}

func Vacuum(db *gorm.DB) error {
	return db.Exec("VACUUM").Error
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
func GetActivePeers(db *gorm.DB) ([]Peer, error) {
	var peers []Peer
	result := db.Where("is_active = ?", true).Find(&peers)
	return peers, result.Error
}
