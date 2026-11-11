package auth

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	sessions   *sessions.Service
	production bool
}

func NewService(db *gorm.DB, s *sessions.Service, production bool) *Service {
	return &Service{db: db, sessions: s, production: production}
}
