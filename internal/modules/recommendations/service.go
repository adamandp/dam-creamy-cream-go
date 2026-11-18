package recommendations

import (
	"cmp"
	"context"
	"errors"
	"slices"

	"dam-creamy-cream-go/internal/dal/model"
	"dam-creamy-cream-go/internal/dal/query"
	"dam-creamy-cream-go/internal/exceptions"
	"dam-creamy-cream-go/internal/types"
	"dam-creamy-cream-go/internal/utils"

	"github.com/sirupsen/logrus"
	"gorm.io/gen"
	"gorm.io/gorm"
)

type Service struct {
	query *query.Query
	log   *logrus.Entry
}

func NewService(db *gorm.DB, log *logrus.Entry) *Service {
	return &Service{
		query: query.Use(db),
		log:   log.WithField("module", "recommendations"),
	}
}

func (s *Service) GetClassic(
	ctx context.Context,
) (*types.WebResponse[*[]ProductResponse], error) {
	product := s.query.Product
	category := s.query.Category

	classicCategories := category.WithContext(ctx).
		Select(category.ID).
		Where(
			category.ID.EqCol(product.CategoryID),
			category.Name.Lower().Like(classicCategoryPattern),
		)

	products, err := s.findProducts(
		ctx,
		0,
		gen.Exists(classicCategories),
	)

	if err != nil {
		return nil, err
	}

	if len(products) == 0 {
		return nil, exceptions.NotFound(classicResourceName)
	}

	sortByRatingAndSales(products)

	data := mapProducts(products)

	return &types.WebResponse[*[]ProductResponse]{
		Message: utils.GetMessage(classicResourceName),
		Data:    &data,
	}, nil
}

func (s *Service) GetCategories(
	ctx context.Context,
) (*types.WebResponse[*[]CategoryResponse], error) {
	s.log.Trace("GetCategories called")

	category := s.query.Category
	product := s.query.Product
	orderItem := s.query.OrderItem

	categories, err := category.WithContext(ctx).
		Where(category.IsActive.Is(true)).
		Find()

	if err != nil {
		s.log.WithError(err).
			Error("Failed to retrieve recommendation categories")

		return nil, exceptions.InternalServerError()
	}

	if len(categories) == 0 {
		return nil, exceptions.NotFound(categoriesResourceName)
	}

	categoryIDs := make([]string, 0, len(categories))

	for _, item := range categories {
		categoryIDs = append(categoryIDs, item.ID)
	}

	products, err := product.WithContext(ctx).
		Where(
			product.IsActive.Is(true),
			product.CategoryID.In(categoryIDs...),
		).
		Preload(
			product.OrderItems.Select(
				orderItem.ID,
				orderItem.ProductID,
			),
		).
		Find()

	if err != nil {
		s.log.WithError(err).
			Error("Failed to retrieve category products")

		return nil, exceptions.InternalServerError()
	}

	bestProducts := make(map[string]*model.Product, len(categories))

	for _, item := range products {
		current, exists := bestProducts[item.CategoryID]

		if !exists || len(item.OrderItems) > len(current.OrderItems) {
			bestProducts[item.CategoryID] = item
		}
	}

	data := make([]CategoryResponse, 0, len(categories))

	for _, item := range categories {
		bestProduct, exists := bestProducts[item.ID]

		if !exists {
			continue
		}

		response := CategoryResponse{
			ID:         bestProduct.ID,
			Category:   item.Name,
			CategoryID: item.ID,
		}

		if bestProduct.ImageURL != nil {
			response.ImageURL = *bestProduct.ImageURL
		}

		data = append(data, response)
	}

	return &types.WebResponse[*[]CategoryResponse]{
		Message: utils.GetMessage(categoriesResourceName),
		Data:    &data,
	}, nil
}

func (s *Service) GetOffers(
	ctx context.Context,
) (*types.WebResponse[*[]ProductResponse], error) {
	product := s.query.Product
	productDiscount := s.query.ProductDiscount

	activeDiscounts := productDiscount.WithContext(ctx).
		Select(productDiscount.ID).
		Where(
			productDiscount.ProductID.EqCol(product.ID),
			productDiscount.IsActive.Is(true),
		)

	products, err := s.findProducts(
		ctx,
		recommendationLimit,
		gen.Exists(activeDiscounts),
	)

	if err != nil {
		return nil, err
	}

	if len(products) == 0 {
		return nil, exceptions.NotFound(offersResourceName)
	}

	slices.SortStableFunc(products, func(a, b *model.Product) int {
		discountA, nullA := offerDiscountAmount(a)
		discountB, nullB := offerDiscountAmount(b)

		if discountA != discountB || nullA != nullB {
			return cmp.Compare(discountB, discountA)
		}

		return compareRatingAndSales(a, b)
	})

	data := mapProducts(products)

	return &types.WebResponse[*[]ProductResponse]{
		Message: utils.GetMessage(offersResourceName),
		Data:    &data,
	}, nil
}

func (s *Service) GetRelated(
	ctx context.Context,
	productID string,
) (*types.WebResponse[*[]ProductResponse], error) {
	product := s.query.Product

	targetProduct, err := product.WithContext(ctx).
		Select(product.CategoryID).
		Where(product.ID.Eq(productID)).
		First()

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, exceptions.NotFound(relatedResourceName)
	}

	if err != nil {
		s.log.WithError(err).
			Error("Failed to retrieve target product")

		return nil, exceptions.InternalServerError()
	}

	products, err := s.findProducts(
		ctx,
		0,
		product.ID.Neq(productID),
		product.CategoryID.Eq(targetProduct.CategoryID),
	)

	if err != nil {
		return nil, err
	}

	sortByRatingAndSales(products)

	if len(products) < recommendationLimit {
		excludedIDs := make([]string, 0, len(products)+1)

		excludedIDs = append(excludedIDs, productID)

		for _, item := range products {
			excludedIDs = append(excludedIDs, item.ID)
		}

		additionalProducts, err := s.findProducts(
			ctx,
			0,
			product.ID.NotIn(excludedIDs...),
		)

		if err != nil {
			return nil, err
		}

		products = append(products, additionalProducts...)

		sortByRatingAndSales(products)
	}

	if len(products) > recommendationLimit {
		products = products[:recommendationLimit]
	}

	if len(products) == 0 {
		return nil, exceptions.NotFound(relatedResourceName)
	}

	data := mapProducts(products)

	return &types.WebResponse[*[]ProductResponse]{
		Message: utils.GetMessage(relatedResourceName),
		Data:    &data,
	}, nil
}

func (s *Service) findProducts(
	ctx context.Context,
	limit int,
	conditions ...gen.Condition,
) ([]*model.Product, error) {
	product := s.query.Product
	review := s.query.Review
	orderItem := s.query.OrderItem
	productDiscount := s.query.ProductDiscount

	products := product.WithContext(ctx).
		Where(product.IsActive.Is(true)).
		Where(conditions...).
		Preload(
			product.Reviews.Select(
				review.ProductID,
				review.Rating,
			),
		).
		Preload(
			product.OrderItems.Select(
				orderItem.ID,
				orderItem.ProductID,
			),
		).
		Preload(
			product.ProductDiscounts.On(
				productDiscount.IsActive.Is(true),
			),
		).
		Preload(product.ProductDiscounts.Discounts)

	if limit > 0 {
		products = products.Limit(limit)
	}

	data, err := products.Find()

	if err != nil {
		s.log.WithError(err).
			Error("Failed to retrieve recommendation products")

		return nil, exceptions.InternalServerError()
	}

	return data, nil
}

func mapProducts(products []*model.Product) []ProductResponse {
	data := make([]ProductResponse, 0, len(products))

	for _, product := range products {
		data = append(data, mapProduct(product))
	}

	return data
}

func mapProduct(product *model.Product) ProductResponse {
	response := ProductResponse{
		ID:    product.ID,
		Name:  product.Name,
		Rate:  averageRating(product),
		Price: product.Price,
	}

	if product.ImageURL != nil {
		response.ImageURL = *product.ImageURL
	}

	if product.Description != nil {
		response.Description = *product.Description
	}

	if product.ProductDiscounts == nil {
		return response
	}

	discount := product.ProductDiscounts.Discounts

	if discount.Value == nil || *discount.Value == 0 {
		return response
	}

	var discountAmount float64

	switch discount.DiscountType {
	case model.DiscountTypeFixed:
		discountAmount = *discount.Value

	case model.DiscountTypePercentage:
		discountAmount = (*discount.Value * product.Price) / 100
	}

	discountPrice := product.Price - discountAmount

	response.DiscountPrice = &discountPrice
	response.DiscountType = &discount.DiscountType
	response.DiscountValue = discount.Value

	return response
}

func averageRating(product *model.Product) float64 {
	if len(product.Reviews) == 0 {
		return 0
	}

	var total float64

	for _, review := range product.Reviews {
		total += float64(review.Rating)
	}

	return total / float64(len(product.Reviews))
}

func compareRatingAndSales(a, b *model.Product) int {
	ratingA := averageRating(a)
	ratingB := averageRating(b)

	if ratingA != ratingB {
		return cmp.Compare(ratingB, ratingA)
	}

	return cmp.Compare(
		len(b.OrderItems),
		len(a.OrderItems),
	)
}

func sortByRatingAndSales(products []*model.Product) {
	slices.SortStableFunc(products, compareRatingAndSales)
}

func offerDiscountAmount(product *model.Product) (float64, bool) {
	if product.ProductDiscounts == nil {
		return 0, false
	}

	discount := product.ProductDiscounts.Discounts

	switch discount.DiscountType {
	case model.DiscountTypeFixed:
		if discount.Value == nil {
			return 0, true
		}

		return *discount.Value, false

	case model.DiscountTypePercentage:
		if discount.Value == nil {
			return 0, false
		}

		// Preserve the original NestJS offer-ranking calculation.
		return *discount.Value / 100, false

	default:
		return 0, false
	}
}
