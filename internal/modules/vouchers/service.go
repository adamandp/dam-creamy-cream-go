package vouchers

import (
	"context"
	"errors"
	"time"

	"dam-creamy-cream-go/internal/dal/model"
	"dam-creamy-cream-go/internal/dal/query"
	"dam-creamy-cream-go/internal/exceptions"
	"dam-creamy-cream-go/internal/modules/products"
	"dam-creamy-cream-go/internal/types"
	"dam-creamy-cream-go/internal/utils"

	"github.com/sirupsen/logrus"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

type Service struct {
	query   *query.Query
	product *products.Service
	log     *logrus.Entry
}

func NewService(db *gorm.DB, productsService *products.Service, log *logrus.Entry) *Service {
	return &Service{
		query:   query.Use(db),
		product: productsService,
		log:     log.WithField("module", "vouchers"),
	}
}

func (s *Service) ValidateVoucher(ctx context.Context, id string) (*model.Voucher, error) {
	return s.FindByID(ctx, id)
}

func (s *Service) Create(ctx context.Context, req CreateRequest) error {
	voucher := &model.Voucher{
		Code:            req.Code,
		Name:            req.Name,
		Description:     req.Description,
		Type:            req.Type,
		Group:           req.Group,
		Value:           req.Value,
		ProductID:       req.ProductID,
		Quantity:        req.Quantity,
		MinimumPurchase: *req.MinimumPurchase,
		IsActive:        *req.IsActive,
		StartDate:       req.StartDate.UTC().Truncate(time.Millisecond),
		EndDate:         req.EndDate.UTC().Truncate(time.Millisecond),
		MaxRedeemed:     *req.MaxRedeemed,
		RedeemedCount:   *req.RedeemedCount,
	}

	if err := s.query.Voucher.WithContext(ctx).Create(voucher); err != nil {
		s.log.WithError(err).Error("Failed to create voucher")
		return exceptions.InternalServerError()
	}

	return nil
}

func (s *Service) FindAll(ctx context.Context, req FindAllRequest) (*types.WebResponse[*[]*model.Voucher], error) {
	voucher := s.query.Voucher
	offset := (req.Page - 1) * req.Limit

	data, err := voucher.WithContext(ctx).
		Offset(offset).
		Limit(req.Limit).
		Find()
	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve vouchers")
		return nil, exceptions.InternalServerError()
	}

	total, err := voucher.WithContext(ctx).Count()
	if err != nil {
		s.log.WithError(err).Error("Failed to count vouchers")
		return nil, exceptions.InternalServerError()
	}

	if total == 0 {
		return nil, exceptions.NotFound(resourceName)
	}

	if data == nil {
		data = []*model.Voucher{}
	}

	return &types.WebResponse[*[]*model.Voucher]{
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

func (s *Service) FindByID(ctx context.Context, id string) (*model.Voucher, error) {
	voucher := s.query.Voucher

	result, err := voucher.WithContext(ctx).
		Where(voucher.ID.Eq(id)).
		First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, exceptions.NotFound(resourceName)
	}

	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve voucher")
		return nil, exceptions.InternalServerError()
	}

	return result, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) error {
	existing, err := s.ValidateVoucher(ctx, id)
	if err != nil {
		return err
	}

	if req.ProductID != nil {
		if _, err := s.product.ValidateProduct(ctx, *req.ProductID); err != nil {
			return err
		}
	}

	voucher := s.query.Voucher

	updates := make([]field.AssignExpr, 0, 15)
	unchanged := make([]string, 0, 14)

	if req.Code != nil {
		if *req.Code == existing.Code {
			unchanged = append(unchanged, "code")
		} else {
			updates = append(updates, voucher.Code.Value(*req.Code))
		}
	}

	if req.Name != nil {
		if *req.Name == existing.Name {
			unchanged = append(unchanged, "name")
		} else {
			updates = append(updates, voucher.Name.Value(*req.Name))
		}
	}

	if req.Description != nil {
		if existing.Description != nil && *req.Description == *existing.Description {
			unchanged = append(unchanged, "description")
		} else {
			updates = append(updates, voucher.Description.Value(*req.Description))
		}
	}

	if req.Type != nil {
		if *req.Type == existing.Type {
			unchanged = append(unchanged, "type")
		} else {
			updates = append(updates, voucher.Type.Value(string(*req.Type)))
		}
	}

	if req.Group != nil {
		if *req.Group == existing.Group {
			unchanged = append(unchanged, "group")
		} else {
			updates = append(updates, voucher.Group.Value(string(*req.Group)))
		}
	}

	if req.Value != nil {
		if existing.Value != nil && *req.Value == *existing.Value {
			unchanged = append(unchanged, "value")
		} else {
			updates = append(updates, voucher.Value.Value(*req.Value))
		}
	}

	if req.ProductID != nil {
		if existing.ProductID != nil && *req.ProductID == *existing.ProductID {
			unchanged = append(unchanged, "productId")
		} else {
			updates = append(updates, voucher.ProductID.Value(*req.ProductID))
		}
	}

	if req.Quantity != nil {
		if existing.Quantity != nil && *req.Quantity == *existing.Quantity {
			unchanged = append(unchanged, "quantity")
		} else {
			updates = append(updates, voucher.Quantity.Value(*req.Quantity))
		}
	}

	if req.MinimumPurchase != nil {
		if *req.MinimumPurchase == existing.MinimumPurchase {
			unchanged = append(unchanged, "minimumPurchase")
		} else {
			updates = append(updates, voucher.MinimumPurchase.Value(*req.MinimumPurchase))
		}
	}

	if req.IsActive != nil {
		if *req.IsActive == existing.IsActive {
			unchanged = append(unchanged, "isActive")
		} else {
			updates = append(updates, voucher.IsActive.Value(*req.IsActive))
		}
	}

	if req.StartDate != nil {
		startDate := req.StartDate.UTC().Truncate(time.Millisecond)

		if startDate.Equal(existing.StartDate) {
			unchanged = append(unchanged, "startDate")
		} else {
			updates = append(updates, voucher.StartDate.Value(startDate))
		}
	}

	if req.EndDate != nil {
		endDate := req.EndDate.UTC().Truncate(time.Millisecond)

		if endDate.Equal(existing.EndDate) {
			unchanged = append(unchanged, "endDate")
		} else {
			updates = append(updates, voucher.EndDate.Value(endDate))
		}
	}

	if req.MaxRedeemed != nil {
		if *req.MaxRedeemed == existing.MaxRedeemed {
			unchanged = append(unchanged, "maxRedeemed")
		} else {
			updates = append(updates, voucher.MaxRedeemed.Value(*req.MaxRedeemed))
		}
	}

	if req.RedeemedCount != nil {
		if *req.RedeemedCount == existing.RedeemedCount {
			unchanged = append(unchanged, "redeemedCount")
		} else {
			updates = append(updates, voucher.RedeemedCount.Value(*req.RedeemedCount))
		}
	}

	if err := utils.ValidateUpdate(len(updates), unchanged); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	updates = append(updates, voucher.UpdatedAt.Value(time.Now()))

	result, err := voucher.WithContext(ctx).
		Where(voucher.ID.Eq(id)).
		UpdateSimple(updates...)
	if err != nil {
		s.log.WithError(err).Error("Failed to update voucher")
		return exceptions.InternalServerError()
	}

	if result.RowsAffected == 0 {
		return exceptions.NotFound(resourceName)
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	voucher := s.query.Voucher

	result, err := voucher.WithContext(ctx).
		Where(voucher.ID.Eq(id)).
		UpdateSimple(
			voucher.IsActive.Value(false),
			voucher.UpdatedAt.Value(time.Now()),
		)
	if err != nil {
		s.log.WithError(err).Error("Failed to deactivate voucher")
		return exceptions.InternalServerError()
	}

	if result.RowsAffected == 0 {
		return exceptions.NotFound(resourceName)
	}

	return nil
}
