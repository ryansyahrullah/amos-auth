package audit

import (
	"time"

	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateAuditLog(log *AuditLog) error {
	return r.db.Create(log).Error
}

func (r *Repository) DeleteOldLogs(retention time.Duration) error {
	cutoff := time.Now().Add(-retention)
	return r.db.Where("created_at < ?", cutoff).Delete(&AuditLog{}).Error
}
