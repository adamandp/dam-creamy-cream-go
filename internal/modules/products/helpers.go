package products

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/pricing"
)

func summary(p domain.Product, rate float64, d ActiveDiscount) CatalogItem {
	item := CatalogItem{ID: p.ID.ID, Name: p.Name, Rate: rate, Price: p.Price}
	if p.ImageURL != nil {
		item.ImageURL = *p.ImageURL
	}
	if p.Description != nil {
		item.Description = *p.Description
	}
	if discounted := pricing.DiscountPrice(p.Price, d.Type, d.Value); discounted != nil {
		item.DiscountType = &d.Type
		item.DiscountValue = d.Value
		item.DiscountPrice = discounted
	}
	return item
}

func valueOrNil(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}
