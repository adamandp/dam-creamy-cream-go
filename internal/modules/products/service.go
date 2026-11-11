package products

import (
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/infrastructure"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/pricing"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	upload   infrastructure.ImageStore
	sessions *sessions.Service
}

func NewService(db *gorm.DB, upload infrastructure.ImageStore, ses *sessions.Service) *Service {
	return &Service{db: db, upload: upload, sessions: ses}
}

func (s *Service) categoryExists(id string) error {
	var count int64
	if err := s.db.Model(&domain.Category{}).Where("id = ?", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return httpx.NotFound("category")
	}
	return nil
}

func (s *Service) ratings(ids []string) (map[string]float64, error) {
	result := map[string]float64{}
	if len(ids) == 0 {
		return result, nil
	}
	var rows []ReviewRating
	err := s.db.Model(&domain.Review{}).Select("product_id, AVG(rating) AS average").Where("product_id IN ?", ids).Group("product_id").Scan(&rows).Error
	for _, r := range rows {
		result[r.ProductID] = r.Average
	}
	return result, err
}

func (s *Service) discounts(ids []string) (map[string]ActiveDiscount, error) {
	result := map[string]ActiveDiscount{}
	if len(ids) == 0 {
		return result, nil
	}
	var rows []ActiveDiscount
	now := time.Now().UTC()
	err := s.db.Table("product_discounts pd").Select("pd.product_id, d.discount_type, d.value").
		Joins("JOIN discounts d ON d.id = pd.discount_id").
		Where("pd.product_id IN ? AND "+pricing.ActiveAssociation, ids, now, now).
		Scan(&rows).Error
	for _, r := range rows {
		result[r.ProductID] = r
	}
	return result, err
}
