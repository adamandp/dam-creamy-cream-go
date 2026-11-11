package uservouchers

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	sessions *sessions.Service
}

func NewService(db *gorm.DB, sessions *sessions.Service) *Service {
	return &Service{db: db, sessions: sessions}
}

// The same current-database authorization is shared with Roles, Users and
// Promotions, so renaming a role cannot silently bypass a separate guard.
