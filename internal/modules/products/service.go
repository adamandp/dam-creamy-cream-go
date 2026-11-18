package products

import (
	"context"
	"errors"
	"strings"
	"time"

	"dam-creamy-cream-go/internal/dal/model"
	"dam-creamy-cream-go/internal/dal/query"
	"dam-creamy-cream-go/internal/exceptions"
	"dam-creamy-cream-go/internal/modules/categories"
	"dam-creamy-cream-go/internal/types"
	"dam-creamy-cream-go/internal/utils"

	"github.com/gofiber/fiber/v3"
	"github.com/minio/minio-go/v7"
	"github.com/sirupsen/logrus"
	"gorm.io/gen/field"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	query      *query.Query
	categories *categories.Service
	minio      *minio.Client
	bucket     string
	log        *logrus.Entry
}

func NewService(
	db *gorm.DB,
	categoriesService *categories.Service,
	minioClient *minio.Client,
	minioBucket string,
	log *logrus.Entry,
) *Service {
	return &Service{
		query:      query.Use(db),
		categories: categoriesService,
		minio:      minioClient,
		bucket:     minioBucket,
		log:        log.WithField("module", "products"),
	}
}

func (s *Service) ValidateProduct(ctx context.Context, id string) (*model.Product, error) {
	product := s.query.Product

	result, err := product.WithContext(ctx).
		Where(product.ID.Eq(id)).
		First()

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, exceptions.NotFound(resourceName)
	}

	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve product")
		return nil, exceptions.InternalServerError()
	}

	return result, nil
}

func (s *Service) Create(ctx context.Context, req CreateRequest) (*CreateResponse, error) {
	if _, err := s.categories.ValidateCategory(ctx, req.CategoryID); err != nil {
		return nil, err
	}

	product := &model.Product{
		CategoryID:  req.CategoryID,
		DiscountID:  req.DiscountID,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		IsActive:    *req.IsActive,
	}

	if err := s.query.Product.
		WithContext(ctx).
		Create(product); err != nil {
		s.log.WithError(err).Error("Failed to create product")
		return nil, exceptions.InternalServerError()
	}

	return &CreateResponse{
		ID: product.ID,
	}, nil
}

func (s *Service) FindAll(ctx context.Context, req FindAllRequest) (*types.WebResponse[*[]*model.Product], error) {
	product := s.query.Product
	offset := (req.Page - 1) * req.Limit

	data, err := product.WithContext(ctx).
		Offset(offset).
		Limit(req.Limit).
		Find()

	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve products")
		return nil, exceptions.InternalServerError()
	}

	total, err := product.WithContext(ctx).Count()
	if err != nil {
		s.log.WithError(err).Error("Failed to count products")
		return nil, exceptions.InternalServerError()
	}

	if total == 0 {
		return nil, exceptions.NotFound(resourceName)
	}

	if data == nil {
		data = []*model.Product{}
	}

	return &types.WebResponse[*[]*model.Product]{
		Message: utils.GetMessage(resourceName),
		Data:    &data,
		Paging: &types.Paging{
			CurrentPage: req.Page,
			PageSize:    req.Limit,
			TotalItems:  int(total),
			TotalPages:  (int(total) + req.Limit - 1) / req.Limit,
		},
	}, nil
}

func (s *Service) FindCatalog(ctx context.Context, req FindCatalogRequest) (*types.WebResponse[*[]CatalogResponse], error) {
	product := s.query.Product
	productDiscount := s.query.ProductDiscount

	offset := (req.Page - 1) * req.Limit

	products := product.WithContext(ctx).
		Where(product.IsActive.Is(true))

	if req.Query != nil {
		pattern := "%" + strings.ToLower(*req.Query) + "%"

		products = products.Where(
			field.Or(
				product.Name.Lower().Like(pattern),
				product.Description.Lower().Like(pattern),
			),
		)
	}

	if req.Cat != nil {
		categoryIDs := strings.Split(*req.Cat, ",")

		products = products.Where(
			product.CategoryID.In(categoryIDs...),
		)
	}

	if req.Min != nil {
		products = products.Where(
			product.Price.Gte(*req.Min),
		)
	}

	if req.Max != nil {
		products = products.Where(
			product.Price.Lte(*req.Max),
		)
	}

	total, err := products.Count()
	if err != nil {
		s.log.WithError(err).Error("Failed to count catalog products")
		return nil, exceptions.InternalServerError()
	}

	if total == 0 && req.Page == 1 {
		return nil, exceptions.NotFound(resourceName)
	}

	orderBy := product.CreatedAt.Desc()

	if req.Sort != nil {
		switch *req.Sort {
		case sortAscending:
			orderBy = product.Price.Asc()
		case sortDescending:
			orderBy = product.Price.Desc()
		}
	}

	data, err := products.
		Preload(product.Reviews).
		Preload(
			product.ProductDiscounts.On(
				productDiscount.IsActive.Is(true),
			),
		).
		Preload(product.ProductDiscounts.Discounts).
		Order(orderBy).
		Offset(offset).
		Limit(req.Limit).
		Find()

	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve product catalog")
		return nil, exceptions.InternalServerError()
	}

	catalog := make([]CatalogResponse, 0, len(data))

	for _, item := range data {
		result := CatalogResponse{
			ID:    item.ID,
			Name:  item.Name,
			Price: item.Price,
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

		catalog = append(catalog, result)
	}

	return &types.WebResponse[*[]CatalogResponse]{
		Message: utils.GetMessage(resourceName),
		Data:    &catalog,
		Paging: &types.Paging{
			CurrentPage: req.Page,
			PageSize:    req.Limit,
			TotalItems:  int(total),
			TotalPages:  (int(total) + req.Limit - 1) / req.Limit,
		},
	}, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) (*UpdateResponse, error) {
	var oldImageURL *string
	var newImageURL *string
	var productID string
	var updateSucceeded bool

	err := s.query.Transaction(func(tx *query.Query) error {
		product := tx.Product

		existing, err := product.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(product.ID.Eq(id)).
			First()

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return exceptions.NotFound(resourceName)
		}

		if err != nil {
			s.log.WithError(err).Error("Failed to retrieve product")
			return exceptions.InternalServerError()
		}

		productID = existing.ID

		updates := make([]field.AssignExpr, 0, 8)
		unchanged := make([]string, 0, 7)

		if req.CategoryID != nil {
			if *req.CategoryID == existing.CategoryID {
				unchanged = append(unchanged, "categoryId")
			} else {
				updates = append(updates, product.CategoryID.Value(*req.CategoryID))
			}
		}

		if req.DiscountID != nil {
			if existing.DiscountID != nil && *req.DiscountID == *existing.DiscountID {
				unchanged = append(unchanged, "discountId")
			} else {
				updates = append(updates, product.DiscountID.Value(*req.DiscountID))
			}
		}

		if req.Name != nil {
			if *req.Name == existing.Name {
				unchanged = append(unchanged, "name")
			} else {
				updates = append(updates, product.Name.Value(*req.Name))
			}
		}

		if req.Description != nil {
			if existing.Description != nil && *req.Description == *existing.Description {
				unchanged = append(unchanged, "description")
			} else {
				updates = append(updates, product.Description.Value(*req.Description))
			}
		}

		if req.Price != nil {
			if *req.Price == existing.Price {
				unchanged = append(unchanged, "price")
			} else {
				updates = append(updates, product.Price.Value(*req.Price))
			}
		}

		if req.IsActive != nil {
			if *req.IsActive == existing.IsActive {
				unchanged = append(unchanged, "isActive")
			} else {
				updates = append(updates, product.IsActive.Value(*req.IsActive))
			}
		}

		changed := len(updates)

		if req.ImageKey != nil {
			changed++
		}

		if err := utils.ValidateUpdate(changed, unchanged); err != nil {
			return exceptions.BadRequest(err.Error())
		}

		if req.ImageKey != nil {
			imageURL, err := s.publishImage(ctx, productID, *req.ImageKey)
			if err != nil {
				return err
			}

			oldImageURL = existing.ImageURL
			newImageURL = &imageURL

			updates = append(
				updates,
				product.ImageURL.Value(imageURL),
			)
		}

		updates = append(
			updates,
			product.UpdatedAt.Value(time.Now()),
		)

		result, err := product.WithContext(ctx).
			Where(product.ID.Eq(id)).
			UpdateSimple(updates...)

		if err != nil {
			s.log.WithError(err).Error("Failed to update product")
			return exceptions.InternalServerError()
		}

		if result.RowsAffected == 0 {
			return exceptions.NotFound(resourceName)
		}

		updateSucceeded = true

		return nil
	})

	if err != nil {
		if newImageURL != nil && !updateSucceeded {
			s.removeImage(ctx, productID, *newImageURL)
		}

		var httpError *fiber.Error

		if errors.As(err, &httpError) {
			return nil, err
		}

		s.log.WithError(err).Error("Failed to commit product update")

		return nil, exceptions.InternalServerError()
	}

	if req.ImageKey != nil {
		s.removeObject(ctx, *req.ImageKey)
	}

	if oldImageURL != nil {
		s.removeImage(ctx, productID, *oldImageURL)
	}

	if newImageURL != nil {
		return &UpdateResponse{
			ImageURL: *newImageURL,
		}, nil
	}

	return nil, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	var imageURL *string
	var productID string

	err := s.query.Transaction(func(tx *query.Query) error {
		product := tx.Product

		existing, err := product.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(product.ID.Eq(id)).
			First()

		if errors.Is(err, gorm.ErrRecordNotFound) {
			return exceptions.NotFound(resourceName)
		}

		if err != nil {
			s.log.WithError(err).Error("Failed to retrieve product")
			return exceptions.InternalServerError()
		}

		result, err := product.WithContext(ctx).
			Where(product.ID.Eq(id)).
			Delete()

		if err != nil {
			s.log.WithError(err).Error("Failed to delete product")
			return exceptions.InternalServerError()
		}

		if result.RowsAffected == 0 {
			return exceptions.NotFound(resourceName)
		}

		productID = existing.ID
		imageURL = existing.ImageURL

		return nil
	})

	if err != nil {
		var httpError *fiber.Error

		if errors.As(err, &httpError) {
			return err
		}

		s.log.WithError(err).Error("Failed to commit product deletion")

		return exceptions.InternalServerError()
	}

	if imageURL != nil {
		s.removeImage(ctx, productID, *imageURL)
	}

	return nil
}
