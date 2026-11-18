package discounts

import (
	"context"
	"errors"
	"time"

	"dam-creamy-cream-go/internal/dal/model"
	"dam-creamy-cream-go/internal/dal/query"
	"dam-creamy-cream-go/internal/exceptions"
	"dam-creamy-cream-go/internal/types"
	"dam-creamy-cream-go/internal/utils"

	"github.com/sirupsen/logrus"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

type Service struct {
	query *query.Query
	log   *logrus.Entry
}

func NewService(db *gorm.DB, log *logrus.Entry) *Service {
	return &Service{
		query: query.Use(db),
		log:   log.WithField("module", "discounts"),
	}
}

func (s *Service) ValidateDiscount(ctx context.Context, id string) (*model.Discount, error) {
	return s.FindByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, req CreateRequest) error {
	discount := &model.Discount{
		Name:          req.Name,
		DiscountType:  req.DiscountType,
		Value:         req.Value,
		MaxApplied:    *req.MaxApplied,
		MaxRedeemed:   *req.MaxRedeemed,
		RedeemedCount: *req.RedeemedCount,
		Description:   req.Description,
		ProductID:     req.ProductID,
		Quantity:      req.Quantity,
		StartDate:     req.StartDate.UTC().Truncate(time.Millisecond),
		EndDate:       req.EndDate.UTC().Truncate(time.Millisecond),
	}

	if err := s.query.Discount.
		WithContext(ctx).
		Create(discount); err != nil {
		s.log.WithError(err).Error("Failed to create discount")
		return exceptions.InternalServerError()
	}

	return nil
}

func (s *Service) FindAll(ctx context.Context, req FindAllRequest) (*types.WebResponse[*[]*model.Discount], error) {
	discount := s.query.Discount
	offset := (req.Page - 1) * req.Limit

	data, err := discount.WithContext(ctx).
		Offset(offset).
		Limit(req.Limit).
		Find()
	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve discounts")
		return nil, exceptions.InternalServerError()
	}

	total, err := discount.WithContext(ctx).Count()
	if err != nil {
		s.log.WithError(err).Error("Failed to count discounts")
		return nil, exceptions.InternalServerError()
	}

	if total == 0 {
		return nil, exceptions.NotFound(resourceName)
	}

	if data == nil {
		data = []*model.Discount{}
	}

	return &types.WebResponse[*[]*model.Discount]{
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

func (s *Service) FindByID(ctx context.Context, id string) (*model.Discount, error) {
	discount := s.query.Discount

	result, err := discount.WithContext(ctx).
		Where(discount.ID.Eq(id)).
		First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, exceptions.NotFound(resourceName)
	}

	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve discount")
		return nil, exceptions.InternalServerError()
	}

	return result, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) error {
	existing, err := s.ValidateDiscount(ctx, id)
	if err != nil {
		return err
	}

	discount := s.query.Discount

	updates := make([]field.AssignExpr, 0, 12)
	unchanged := make([]string, 0, 11)

	if req.Name != nil {
		if *req.Name == existing.Name {
			unchanged = append(unchanged, "name")
		} else {
			updates = append(updates, discount.Name.Value(*req.Name))
		}
	}

	if req.DiscountType != nil {
		if *req.DiscountType == existing.DiscountType {
			unchanged = append(unchanged, "discountType")
		} else {
			updates = append(updates, discount.DiscountType.Value(string(*req.DiscountType)))
		}
	}

	if req.Value != nil {
		if existing.Value != nil && *req.Value == *existing.Value {
			unchanged = append(unchanged, "value")
		} else {
			updates = append(updates, discount.Value.Value(*req.Value))
		}
	}

	if req.MaxApplied != nil {
		if *req.MaxApplied == existing.MaxApplied {
			unchanged = append(unchanged, "maxApplied")
		} else {
			updates = append(updates, discount.MaxApplied.Value(*req.MaxApplied))
		}
	}

	if req.MaxRedeemed != nil {
		if *req.MaxRedeemed == existing.MaxRedeemed {
			unchanged = append(unchanged, "maxRedeemed")
		} else {
			updates = append(updates, discount.MaxRedeemed.Value(*req.MaxRedeemed))
		}
	}

	if req.RedeemedCount != nil {
		if *req.RedeemedCount == existing.RedeemedCount {
			unchanged = append(unchanged, "redeemedCount")
		} else {
			updates = append(updates, discount.RedeemedCount.Value(*req.RedeemedCount))
		}
	}

	if req.Description != nil {
		if existing.Description != nil && *req.Description == *existing.Description {
			unchanged = append(unchanged, "description")
		} else {
			updates = append(updates, discount.Description.Value(*req.Description))
		}
	}

	if req.ProductID != nil {
		if existing.ProductID != nil && *req.ProductID == *existing.ProductID {
			unchanged = append(unchanged, "productId")
		} else {
			updates = append(updates, discount.ProductID.Value(*req.ProductID))
		}
	}

	if req.Quantity != nil {
		if existing.Quantity != nil && *req.Quantity == *existing.Quantity {
			unchanged = append(unchanged, "quantity")
		} else {
			updates = append(updates, discount.Quantity.Value(*req.Quantity))
		}
	}

	if req.StartDate != nil {
		startDate := req.StartDate.UTC().Truncate(time.Millisecond)

		if startDate.Equal(existing.StartDate) {
			unchanged = append(unchanged, "startDate")
		} else {
			updates = append(updates, discount.StartDate.Value(startDate))
		}
	}

	if req.EndDate != nil {
		endDate := req.EndDate.UTC().Truncate(time.Millisecond)

		if endDate.Equal(existing.EndDate) {
			unchanged = append(unchanged, "endDate")
		} else {
			updates = append(updates, discount.EndDate.Value(endDate))
		}
	}

	if err := utils.ValidateUpdate(len(updates), unchanged); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	updates = append(updates, discount.UpdatedAt.Value(time.Now()))

	result, err := discount.WithContext(ctx).
		Where(discount.ID.Eq(id)).
		UpdateSimple(updates...)
	if err != nil {
		s.log.WithError(err).Error("Failed to update discount")
		return exceptions.InternalServerError()
	}

	if result.RowsAffected == 0 {
		return exceptions.NotFound(resourceName)
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	discount := s.query.Discount

	result, err := discount.WithContext(ctx).
		Where(discount.ID.Eq(id)).
		Delete()
	if err != nil {
		s.log.WithError(err).Error("Failed to delete discount")
		return exceptions.InternalServerError()
	}

	if result.RowsAffected == 0 {
		return exceptions.NotFound(resourceName)
	}

	return nil
}
