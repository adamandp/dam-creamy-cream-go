package productdiscounts

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	sessions *sessions.Service
}

func NewService(db *gorm.DB, ses *sessions.Service) *Service { return &Service{db: db, sessions: ses} }
