package product

import (
	"context"
	"errors"

	"dam-creamy-cream-go/internal/dal/model"
	"dam-creamy-cream-go/internal/dal/query"
	"dam-creamy-cream-go/internal/exceptions"
	"dam-creamy-cream-go/internal/types"
	"dam-creamy-cream-go/internal/utils"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type Service struct {
	query *query.Query
	log   *logrus.Entry
}

func NewService(db *gorm.DB, log *logrus.Entry) *Service {
	return &Service{
		query: query.Use(db),
		log:   log.WithField("module", "product"),
	}
}

func (s *Service) FindOne(
	ctx context.Context,
	id string,
) (*types.WebResponse[*FindOneResponse], error) {
	product := s.query.Product
	productDiscount := s.query.ProductDiscount

	item, err := product.WithContext(ctx).
		Where(product.ID.Eq(id)).
		Preload(product.Categories).
		Preload(product.Reviews).
		Preload(product.OrderItems).
		Preload(
			product.ProductDiscounts.On(
				productDiscount.IsActive.Is(true),
			),
		).
		Preload(product.ProductDiscounts.Discounts).
		First()

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, exceptions.NotFound(resourceName)
	}

	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve product")
		return nil, exceptions.InternalServerError()
	}

	result := FindOneResponse{
		ID:       item.ID,
		Name:     item.Name,
		Category: item.Categories.Name,
		Price:    item.Price,
	}

	if item.ImageURL != nil {
		result.ImageURL = *item.ImageURL
	}

	if item.Description != nil {
		result.Description = *item.Description
	}

	if len(item.Reviews) > 0 {
		var totalRating float64

		for _, review := range item.Reviews {
			totalRating += float64(review.Rating)
		}

		result.Rate = totalRating / float64(len(item.Reviews))
	}

	for _, orderItem := range item.OrderItems {
		result.Sales += int64(orderItem.Quantity)
	}

	if item.ProductDiscounts != nil {
		discount := item.ProductDiscounts.Discounts

		if discount.Value != nil && *discount.Value != 0 {
			var discountAmount float64

			switch discount.DiscountType {
			case model.DiscountTypeFixed:
				discountAmount = *discount.Value

			case model.DiscountTypePercentage:
				discountAmount = (*discount.Value * item.Price) / 100
			}

			discountPrice := item.Price - discountAmount

			result.DiscountPrice = &discountPrice
			result.DiscountType = &discount.DiscountType
			result.DiscountValue = discount.Value
		}
	}

	return &types.WebResponse[*FindOneResponse]{
		Message: utils.GetMessage(resourceName),
		Data:    &result,
	}, nil
}

func (s *Service) FindInformation(
	ctx context.Context,
	id string,
) (*types.WebResponse[*FindInformationResponse], error) {
	product := s.query.Product
	productInformation := s.query.ProductInformation

	information, err := productInformation.WithContext(ctx).
		Where(productInformation.ProductID.Eq(id)).
		First()

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, exceptions.NotFound("Product Information")
	}

	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve product information")
		return nil, exceptions.InternalServerError()
	}

	item, err := product.WithContext(ctx).
		Where(product.ID.Eq(id)).
		First()

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, exceptions.NotFound(resourceName)
	}

	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve product")
		return nil, exceptions.InternalServerError()
	}

	var description string

	if item.Description != nil {
		description = *item.Description
	}

	result, err := buildInformation(description, information)
	if err != nil {
		s.log.WithError(err).Error("Failed to build product information")
		return nil, exceptions.InternalServerError()
	}

	s.log.Debug(result)

	return &types.WebResponse[*FindInformationResponse]{
		Message: utils.GetMessage(resourceName),
		Data: &FindInformationResponse{
			Information: result,
		},
	}, nil
}
