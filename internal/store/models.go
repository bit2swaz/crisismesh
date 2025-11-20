package store

import (
	"time"
)

type Peer struct {
	ID       string `gorm:"primaryKey"`
	Nick     string
	Addr     string
	LastSeen time.Time
	IsActive bool
}

type Message struct {
	ID          string `gorm:"primaryKey"`
	SenderID    string
	RecipientID string
	Content     string
	Priority    int
	Timestamp   int64
	TTL         int
	HopCount    int
	Status      string
}
