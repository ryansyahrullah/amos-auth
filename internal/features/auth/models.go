package auth

import (
	"time"
)

type PasswordReset struct {
	ID        uint   `gorm:"primaryKey"`
	Email     string `gorm:"index;not null"`
	Token     string `gorm:"not null"`
	ExpiresAt time.Time
	CreatedAt time.Time
}
