package models

import (
	"time"
)

type User struct {
	ID                  uint   `gorm:"primarykey"`
	Email               string `gorm:"unique;not null"`
	Username            string `gorm:"unique;not null"`
	PasswordHash        string `gorm:"not null"`
	FullName            string
	CreatedAt           time.Time
	UpdatedAt           time.Time
	LastLogin           *time.Time
	IsActive            bool `gorm:"default:true"`
	FailedLoginAttempts int  `gorm:"default:0"`
	LastFailedAttempt   *time.Time
}
