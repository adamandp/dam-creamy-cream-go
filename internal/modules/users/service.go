package users

import (
	"strings"
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/infrastructure"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	sessions *sessions.Service
	upload   infrastructure.ImageStore
}

func NewService(db *gorm.DB, s *sessions.Service, upload infrastructure.ImageStore) *Service {
	return &Service{db: db, sessions: s, upload: upload}
}

func (s *Service) hasDuplicate(username, email, phone *string) error {
	db := s.db.Model(&domain.User{})
	where := []string{}
	args := []any{}
	if username != nil {
		where = append(where, "username = ?")
		args = append(args, *username)
	}
	if email != nil {
		where = append(where, "email = ?")
		args = append(args, *email)
	}
	if phone != nil {
		where = append(where, "phone_number = ?")
		args = append(args, *phone)
	}
	if len(where) == 0 {
		return nil
	}
	var u domain.User
	err := db.Where(strings.Join(where, " OR "), args...).First(&u).Error
	if err == gorm.ErrRecordNotFound {
		return nil
	}
	if err != nil {
		return err
	}
	if username != nil && u.Username == *username {
		return httpx.Conflict("username")
	}
	if email != nil && u.Email != nil && *u.Email == *email {
		return httpx.Conflict("email")
	}
	if phone != nil && u.PhoneNumber != nil && *u.PhoneNumber == *phone {
		return httpx.Conflict("phoneNumber")
	}
	return nil
}
