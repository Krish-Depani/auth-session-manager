package models

import (
	"time"
)

type UserSession struct {
	ID           uint   `gorm:"primarykey"`
	UserID       uint   `gorm:"not null"`
	SessionToken string `gorm:"unique;not null"`
	DeviceInfo   string
	IPAddress    string
	UserAgent    string
	Location     string
	CreatedAt    time.Time
	LastActivity time.Time
	ExpiresAt    time.Time
	IsActive     bool `gorm:"default:true"`
	User         User `gorm:"foreignkey:UserID"`
}
