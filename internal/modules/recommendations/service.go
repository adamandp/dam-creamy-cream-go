package recommendations

import (
	"gorm.io/gorm"
)

type Service struct{ db *gorm.DB }

func NewService(db *gorm.DB) *Service { return &Service{db: db} }

func (p recommendation) view() recommendationView {
	v := recommendationView{ID: p.Product.ID.ID, Name: p.Product.Name, Rate: p.Rate, Price: p.Product.Price}
	if p.Product.ImageURL != nil {
		v.ImageURL = *p.Product.ImageURL
	}
	if p.Product.Description != nil {
		v.Description = *p.Product.Description
	}
	// This intentionally uses NestJS RecommendationsService's own calculation,
	// not the corrected shared Cart/Catalog/Checkout pricing rules.
	if p.Discount != nil && p.Discount.Value != nil && *p.Discount.Value != 0 {
		v.DiscountType, v.DiscountValue = &p.Discount.Type, p.Discount.Value
		amount := 0.0
		switch p.Discount.Type {
		case "FIXED":
			amount = *p.Discount.Value
		case "PERCENTAGE":
			amount = *p.Discount.Value * p.Product.Price / 100
		}
		price := p.Product.Price - amount
		v.DiscountPrice = &price
	}
	return v
}
