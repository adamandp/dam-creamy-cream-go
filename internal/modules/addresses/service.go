package addresses

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	sessions *sessions.Service
}

func NewService(db *gorm.DB, s *sessions.Service) *Service { return &Service{db: db, sessions: s} }
