package categories

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"dam-creamy-cream-go/internal/dal/model"
	"dam-creamy-cream-go/internal/dal/query"
	"dam-creamy-cream-go/internal/exceptions"
	"dam-creamy-cream-go/internal/types"
	"dam-creamy-cream-go/internal/utils"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gen/field"
	"gorm.io/gorm"
)

type Service struct {
	query *query.Query
	redis *redis.Client
	log   *logrus.Entry
}

func NewService(db *gorm.DB, redis *redis.Client, log *logrus.Entry) *Service {
	return &Service{
		query: query.Use(db),
		redis: redis,
		log:   log.WithField("module", "categories"),
	}
}

func (s *Service) Create(ctx context.Context, req CreateRequest) error {
	category := &model.Category{
		Name:        req.Name,
		Description: req.Description,
		IsActive:    req.IsActive,
	}

	if err := s.query.Category.
		WithContext(ctx).
		Create(category); err != nil {
		var sqlError interface {
			SQLState() string
		}

		if errors.Is(err, gorm.ErrDuplicatedKey) ||
			(errors.As(err, &sqlError) && sqlError.SQLState() == "23505") {
			return exceptions.Conflict(resourceName)
		}

		s.log.WithError(err).Error("Failed to create category")

		return exceptions.InternalServerError()
	}

	return nil
}

func (s *Service) FindAll(ctx context.Context, req FindAllRequest) (*types.WebResponse[*[]*model.Category], error) {
	cacheKey := fmt.Sprintf("%s:%d:%d", cachePrefix, req.Page, req.Limit)

	// Read from Redis.
	cached, err := s.redis.Get(ctx, cacheKey).Bytes()
	if err == nil && len(cached) > 0 {
		var response *types.WebResponse[*[]*model.Category]

		if err := json.Unmarshal(cached, &response); err != nil {
			s.log.WithError(err).Error("Failed to decode category cache")
			return nil, exceptions.InternalServerError()
		}

		if response != nil {
			return response, nil
		}
	}

	if err != nil && !errors.Is(err, redis.Nil) {
		s.log.WithError(err).Error("Failed to read category cache")
		return nil, exceptions.InternalServerError()
	}

	// Query PostgreSQL.
	category := s.query.Category
	offset := (req.Page - 1) * req.Limit

	data, err := category.WithContext(ctx).
		Offset(offset).
		Limit(req.Limit).
		Find()
	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve categories")
		return nil, exceptions.InternalServerError()
	}

	total, err := category.WithContext(ctx).Count()
	if err != nil {
		s.log.WithError(err).Error("Failed to count categories")
		return nil, exceptions.InternalServerError()
	}

	if total == 0 {
		return nil, exceptions.NotFound(resourceName)
	}

	if data == nil {
		data = []*model.Category{}
	}

	response := &types.WebResponse[*[]*model.Category]{
		Message: utils.GetMessage(resourceName),
		Data:    &data,
		Paging: &types.Paging{
			CurrentPage: req.Page,
			PageSize:    req.Limit,
			TotalItems:  int(total),
			TotalPages:  (int(total) + req.Limit - 1) / req.Limit,
		},
	}

	// Store response in Redis.
	payload, err := json.Marshal(response)
	if err != nil {
		s.log.WithError(err).Error("Failed to encode category cache")
		return nil, exceptions.InternalServerError()
	}

	if err := s.redis.Set(ctx, cacheKey, payload, cacheTTL).Err(); err != nil {
		s.log.WithError(err).Error("Failed to write category cache")
		return nil, exceptions.InternalServerError()
	}

	return response, nil
}

func (s *Service) FindByID(ctx context.Context, id string) (*model.Category, error) {
	category := s.query.Category

	result, err := category.WithContext(ctx).
		Where(category.ID.Eq(id)).
		First()
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, exceptions.NotFound(resourceName)
	}

	if err != nil {
		s.log.WithError(err).Error("Failed to retrieve category")
		return nil, exceptions.InternalServerError()
	}

	return result, nil
}

func (s *Service) ValidateCategory(ctx context.Context, id string) (*model.Category, error) {
	return s.FindByID(ctx, id)
}

func (s *Service) Update(ctx context.Context, id string, req UpdateRequest) error {
	existing, err := s.FindByID(ctx, id)
	if err != nil {
		return err
	}

	category := s.query.Category

	updates := make([]field.AssignExpr, 0, 4)
	unchanged := make([]string, 0, 3)

	if req.Name != nil {
		if *req.Name == existing.Name {
			unchanged = append(unchanged, "name")
		} else {
			updates = append(updates, category.Name.Value(*req.Name))
		}
	}

	if req.Description != nil {
		if existing.Description != nil && *req.Description == *existing.Description {
			unchanged = append(unchanged, "description")
		} else {
			updates = append(updates, category.Description.Value(*req.Description))
		}
	}

	if req.IsActive != nil {
		if existing.IsActive != nil && *req.IsActive == *existing.IsActive {
			unchanged = append(unchanged, "isActive")
		} else {
			updates = append(updates, category.IsActive.Value(*req.IsActive))
		}
	}

	if err := utils.ValidateUpdate(len(updates), unchanged); err != nil {
		return exceptions.BadRequest(err.Error())
	}

	updates = append(updates, category.UpdatedAt.Value(time.Now()))

	result, err := category.WithContext(ctx).
		Where(category.ID.Eq(id)).
		UpdateSimple(updates...)
	if err != nil {
		var sqlError interface {
			SQLState() string
		}

		if errors.Is(err, gorm.ErrDuplicatedKey) ||
			(errors.As(err, &sqlError) && sqlError.SQLState() == "23505") {
			return exceptions.Conflict(resourceName)
		}

		s.log.WithError(err).Error("Failed to update category")
		return exceptions.InternalServerError()
	}

	if result.RowsAffected == 0 {
		return exceptions.NotFound(resourceName)
	}

	return nil
}

func (s *Service) Delete(ctx context.Context, id string) error {
	category := s.query.Category

	result, err := category.WithContext(ctx).
		Where(category.ID.Eq(id)).
		Delete()
	if err != nil {
		s.log.WithError(err).Error("Failed to delete category")
		return exceptions.InternalServerError()
	}

	if result.RowsAffected == 0 {
		return exceptions.NotFound(resourceName)
	}

	return nil
}
