package audit

import (
	"time"

	"gorm.io/gorm"
)

type AuditLog struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	UserID    *uint          `json:"user_id"` // Nullable for failed login attempts (unknown user) or system actions
	Action    string         `gorm:"size:50;not null" json:"action"`
	Details   string         `gorm:"type:text" json:"details"`
	IPAddress string         `gorm:"size:45" json:"ip_address"`
	UserAgent string         `gorm:"size:255" json:"user_agent"`
	CreatedAt time.Time      `json:"created_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
