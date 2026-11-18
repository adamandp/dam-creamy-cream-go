package productdiscounts

import (
	"context"
	"errors"
	"time"

	"dam-creamy-cream-go/internal/dal/model"
	"dam-creamy-cream-go/internal/dal/query"
	"dam-creamy-cream-go/internal/exceptions"
	"dam-creamy-cream-go/internal/modules/discounts"
	"dam-creamy-cream-go/internal/modules/products"
	"dam-creamy-cream-go/internal/types"
	"dam-creamy-cream-go/internal/utils"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	query     *query.Query
	products  *products.Service
	discounts *discounts.Service
	log       *logrus.Entry
}

func NewService(
	db *gorm.DB,
	productsService *products.Service,
	discountsService *discounts.Service,
	log *logrus.Entry,
) *Service {
	return &Service{
		query:     query.Use(db),
		products:  productsService,
		discounts: discountsService,
		log:       log.WithField("module", "product-discounts"),
	}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) error {
	if _, err := s.products.ValidateProduct(ctx, req.ProductID); err != nil {
		return err
	}

	if _, err := s.discounts.ValidateDiscount(ctx, req.DiscountID); err != nil {
		return err
	}

	isActive := true

	record := &model.ProductDiscount{
		ProductID:  req.ProductID,
		DiscountID: req.DiscountID,
		IsActive:   &isActive,
	}

	productDiscount := s.query.ProductDiscount

	result := productDiscount.WithContext(ctx).
		UnderlyingDB().
		Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "product_id"},
			},
			DoUpdates: clause.AssignmentColumns([]string{
				"discount_id",
				"is_active",
				"updated_at",
			}),
			Where: clause.Where{
				Exprs: []clause.Expression{
					clause.Eq{
						Column: clause.Column{
							Table: record.TableName(),
							Name:  "is_active",
						},
						Value: false,
					},
				},
			},
		}).
		Create(record)

	if result.Error != nil {
		s.log.WithError(result.Error).Error("Failed to assign product discount")
		return exceptions.InternalServerError()
	}

	if result.RowsAffected == 0 {
		return exceptions.Conflict(resourceName)
	}

	return nil
}

func (s *Service) FindAll(
	ctx context.Context,
	req FindAllRequest,
) (*types.WebResponse[*[]*model.ProductDiscount], error) {
	productDiscount := s.query.ProductDiscount
	offset := (req.Page - 1) * req.Limit

	data, err := productDiscount.WithContext(ctx).
		Offset(offset).
		Limit(req.Limit).
		Find()

	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve product discounts")
		return nil, exceptions.InternalServerError()
	}

	total, err := productDiscount.WithContext(ctx).Count()

	if err != nil {
		s.log.WithError(err).Error("Failed to count product discounts")
		return nil, exceptions.InternalServerError()
	}

	if total == 0 {
		return nil, exceptions.NotFound(resourceName)
	}

	if data == nil {
		data = []*model.ProductDiscount{}
	}

	return &types.WebResponse[*[]*model.ProductDiscount]{
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

func (s *Service) FindByID(ctx context.Context, id string) (*model.ProductDiscount, error) {
	productDiscount := s.query.ProductDiscount

	result, err := productDiscount.WithContext(ctx).
		Where(productDiscount.ID.Eq(id)).
		First()

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, exceptions.NotFound(resourceName)
	}

	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve product discount")
		return nil, exceptions.InternalServerError()
	}

	return result, nil
}

func (s *Service) ValidateProductDiscount(ctx context.Context, id string) (*model.ProductDiscount, error) {
	return s.FindByID(ctx, id)
}

func (s *Service) FindByProductID(ctx context.Context, productID string) ([]*model.ProductDiscount, error) {
	productDiscount := s.query.ProductDiscount

	data, err := productDiscount.WithContext(ctx).
		Where(productDiscount.ProductID.Eq(productID)).
		Find()

	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve discounts by product ID")
		return nil, exceptions.InternalServerError()
	}

	if data == nil {
		data = []*model.ProductDiscount{}
	}

	return data, nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	productDiscount := s.query.ProductDiscount

	result, err := productDiscount.WithContext(ctx).
		Where(productDiscount.ID.Eq(id)).
		UpdateSimple(
			productDiscount.IsActive.Value(false),
			productDiscount.UpdatedAt.Value(time.Now()),
		)

	if err != nil {
		s.log.WithError(err).Error("Failed to deactivate product discount")
		return exceptions.InternalServerError()
	}

	if result.RowsAffected == 0 {
		return exceptions.NotFound(resourceName)
	}

	return nil
}
