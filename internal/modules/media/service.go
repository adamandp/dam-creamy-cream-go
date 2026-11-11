package media

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/infrastructure"
)

type Service struct {
	store infrastructure.ImageStore
}

func NewService(store infrastructure.ImageStore) *Service {
	return &Service{store: store}
}
