package store

import (
	"time"
)

type Peer struct {
	ID       string `gorm:"primaryKey"`
	Nick     string
	Addr     string
	PubKey   string
	LastSeen time.Time
	IsActive bool
}
type Message struct {
	ID          string `gorm:"primaryKey"`
	SenderID    string
	RecipientID string
	Content     string
	Priority    int
	Author      string  `json:"author"`
	Lat         float64 `json:"lat"`
	Long        float64 `json:"long"`
	Timestamp   int64
	TTL         int
	HopCount    int
	Status      string
	IsEncrypted bool
}
