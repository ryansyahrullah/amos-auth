package user

import (
	"time"
)

type Permission struct {
	ID          uint   `gorm:"primaryKey"`
	Name        string `gorm:"unique;not null"` // e.g., "user.create", "hcgs.read"
	Description string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Role struct {
	ID          uint         `gorm:"primaryKey"`
	Name        string       `gorm:"unique;not null"`
	Menus       string       `json:"menus"` // Comma-separated list of menu keys
	Permissions []Permission `gorm:"many2many:role_permissions;constraint:OnDelete:CASCADE;"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type User struct {
	ID            uint   `gorm:"primaryKey"`
	NRP           string `gorm:"unique;index"` // Nullable if email is used, but usually NRP is unique
	Name          string `json:"name"`         // Display name for profile, independent of Pegawai name
	Email         string `gorm:"unique;index;not null"`
	Password      string `gorm:"not null"`
	PlainPassword string `json:"plain_password,omitempty"` // Only visible to Super Admin
	RoleID        *uint  `gorm:"index"`
	Role          Role   `gorm:"foreignKey:RoleID;constraint:OnDelete:CASCADE"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type RefreshToken struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    uint      `gorm:"type:bigint unsigned;not null"`
	Token     string    `gorm:"type:varchar(512);uniqueIndex;not null"`
	ExpiresAt time.Time `gorm:"not null"`
	Revoked   bool      `gorm:"default:false"`
	CreatedAt time.Time
	UpdatedAt time.Time
	User      User `gorm:"foreignKey:UserID;constraint:OnDelete:CASCADE"`
}
