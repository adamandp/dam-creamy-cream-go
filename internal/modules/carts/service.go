package carts

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"gorm.io/gorm"
)

type Service struct {
	db      *gorm.DB
	session *sessions.Service
}

func NewService(db *gorm.DB, s *sessions.Service) *Service { return &Service{db: db, session: s} }

func (s *Service) validateRelated(tx *gorm.DB, userID, productID string) error {
	var count int64
	if err := tx.Model(&domain.User{}).Where("id = ?", userID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return httpx.NotFound("User")
	}
	if err := tx.Model(&domain.Product{}).Where("id = ?", productID).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return httpx.NotFound("Product")
	}
	return nil
}
