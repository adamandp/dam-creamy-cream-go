package inventory

import (
	"context"
	"errors"
	"time"

	"dam-creamy-cream-go/internal/dal/model"
	"dam-creamy-cream-go/internal/dal/query"
	"dam-creamy-cream-go/internal/exceptions"
	"dam-creamy-cream-go/internal/types"
	"dam-creamy-cream-go/internal/utils"

	"github.com/gofiber/fiber/v3"
	"github.com/sirupsen/logrus"
	"gorm.io/gen/field"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Service struct {
	query *query.Query
	log   *logrus.Entry
}

func NewService(db *gorm.DB, log *logrus.Entry) *Service {
	return &Service{
		query: query.Use(db),
		log:   log.WithField("module", "inventory"),
	}
}

func (s *Service) FindAll(ctx context.Context, req FindAllRequest) (*types.WebResponse[*[]*model.Inventory], error) {
	inventory := s.query.Inventory
	offset := (req.Page - 1) * req.Limit

	data, err := inventory.WithContext(ctx).
		Offset(offset).
		Limit(req.Limit).
		Find()
	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve inventories")
		return nil, exceptions.InternalServerError()
	}

	total, err := inventory.WithContext(ctx).Count()
	if err != nil {
		s.log.WithError(err).Error("Failed to count inventories")
		return nil, exceptions.InternalServerError()
	}

	if total == 0 {
		return nil, exceptions.NotFound(resourceName)
	}

	if data == nil {
		data = []*model.Inventory{}
	}

	return &types.WebResponse[*[]*model.Inventory]{
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

func (s *Service) FindByID(ctx context.Context, id string) (*model.Inventory, error) {
	inventory := s.query.Inventory

	result, err := inventory.WithContext(ctx).
		Where(inventory.ID.Eq(id)).
		First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, exceptions.NotFound(resourceName)
	}

	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve inventory")
		return nil, exceptions.InternalServerError()
	}

	return result, nil
}

func (s *Service) FindByProductID(ctx context.Context, productID string) (*model.Inventory, error) {
	inventory := s.query.Inventory

	result, err := inventory.WithContext(ctx).
		Where(inventory.ProductID.Eq(productID)).
		First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, exceptions.NotFound(resourceName)
	}

	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve inventory by product ID")
		return nil, exceptions.InternalServerError()
	}

	return result, nil
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) error {
	err := s.query.Transaction(func(tx *query.Query) error {
		inventory := tx.Inventory

		existing, err := inventory.WithContext(ctx).
			Clauses(clause.Locking{Strength: "UPDATE"}).
			Where(inventory.ID.Eq(id)).
			First()
		if err != nil {
			return err
		}

		updates := make([]field.AssignExpr, 0, 4)
		unchanged := make([]string, 0, 3)

		quantity := existing.Quantity
		reservedQuantity := existing.ReservedQuantity

		if req.Quantity != nil {
			quantity = *req.Quantity

			if quantity == existing.Quantity {
				unchanged = append(unchanged, "quantity")
			} else {
				updates = append(updates, inventory.Quantity.Value(quantity))
			}
		}

		if req.ReservedQuantity != nil {
			reservedQuantity = *req.ReservedQuantity

			if reservedQuantity == existing.ReservedQuantity {
				unchanged = append(unchanged, "reservedQuantity")
			} else {
				updates = append(updates, inventory.ReservedQuantity.Value(reservedQuantity))
			}
		}

		if req.IsActive != nil {
			if *req.IsActive == existing.IsActive {
				unchanged = append(unchanged, "isActive")
			} else {
				updates = append(updates, inventory.IsActive.Value(*req.IsActive))
			}
		}

		if err := utils.ValidateUpdate(len(updates), unchanged); err != nil {
			return exceptions.BadRequest(err.Error())
		}

		if req.Quantity != nil || req.ReservedQuantity != nil {
			if quantity < 0 || reservedQuantity < 0 {
				return exceptions.BadRequest("Inventory quantities cannot be negative")
			}

			if reservedQuantity > quantity {
				return exceptions.BadRequest("Reserved quantity cannot exceed total quantity")
			}
		}

		updates = append(updates, inventory.UpdatedAt.Value(time.Now()))

		result, err := inventory.WithContext(ctx).
			Where(inventory.ID.Eq(id)).
			UpdateSimple(updates...)
		if err != nil {
			return err
		}

		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}

		return nil
	})
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return exceptions.NotFound(resourceName)
	}

	if err != nil {
		var httpError *fiber.Error

		if errors.As(err, &httpError) {
			return httpError
		}

		s.log.WithError(err).Error("Failed to update inventory")
		return exceptions.InternalServerError()
	}

	return nil
}
