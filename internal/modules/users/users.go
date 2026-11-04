package users

import (
	"context"
	"io"
	"regexp"
	"strings"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/infrastructure"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	db       *gorm.DB
	sessions *sessions.Service
	upload   *infrastructure.Cloudinary
}

func NewService(db *gorm.DB, s *sessions.Service, upload *infrastructure.Cloudinary) *Service {
	return &Service{db: db, sessions: s, upload: upload}
}
func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/users")
	r.Get("/", s.findAll)
	r.Get("/profile", s.profile)
	r.Get("/:id", s.findOne)
	r.Patch("/:id", s.update)
	r.Put("/:id/password", s.updatePassword)
	r.Put("/:id/image", s.updateImage)
	r.Delete("/:id", s.remove)
}
func (s *Service) findAll(c fiber.Ctx) error {
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	var data []domain.User
	var total int64
	if err := s.db.Model(&domain.User{}).Count(&total).Error; err != nil {
		return err
	}
	if total == 0 {
		return httpx.NotFound("User")
	}
	if err := s.db.Offset((page - 1) * limit).Limit(limit).Find(&data).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "User"), Data: data, Paging: httpx.Pagination(page, limit, total)})
}
func (s *Service) profile(c fiber.Ctx) error {
	claims, err := s.sessions.AccessFor(c)
	if err != nil {
		return err
	}
	var u domain.User
	if err := s.db.First(&u, "id = ?", claims.Subject).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "User"), Data: fiber.Map{"username": u.Username, "email": u.Email, "phoneNumber": u.PhoneNumber, "fullName": u.FullName, "birthDate": u.BirthDate, "imageUrl": u.ImageURL, "gender": u.Gender}})
}
func (s *Service) findOne(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var u domain.User
	if err := s.db.First(&u, "id = ?", id).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "User"), Data: u})
}

type UpdateInput struct {
	RoleID      *string `json:"roleId"`
	Username    *string `json:"username"`
	Email       *string `json:"email"`
	PhoneNumber *string `json:"phoneNumber"`
	Password    *string `json:"password"`
	FullName    *string `json:"fullName"`
	Gender      *string `json:"gender"`
	IsActive    *bool   `json:"isActive"`
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

var lowercase = regexp.MustCompile(`[a-z]`)
var uppercase = regexp.MustCompile(`[A-Z]`)
var digit = regexp.MustCompile(`[0-9]`)
var special = regexp.MustCompile(`[!@#$%^&*]`)

func validatePassword(password string) error {
	if len(password) < 8 || len(password) > 255 || !lowercase.MatchString(password) || !uppercase.MatchString(password) || !digit.MatchString(password) || !special.MatchString(password) {
		return fiber.NewError(400, "Password must be 8-255 chars with uppercase, lowercase, number and special character")
	}
	return nil
}
func (s *Service) update(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var u domain.User
	if err := s.db.First(&u, "id = ?", id).Error; err != nil {
		return err
	}
	var body UpdateInput
	if err := httpx.Bind(c, &body); err != nil {
		return err
	}
	updates := map[string]any{}
	if body.RoleID != nil {
		if err := httpx.UUID(*body.RoleID); err != nil {
			return err
		}
		var role domain.Role
		if err := s.db.First(&role, "id = ?", *body.RoleID).Error; err != nil {
			return err
		}
		updates["role_id"] = *body.RoleID
	}
	if err := s.hasDuplicate(body.Username, body.Email, body.PhoneNumber); err != nil {
		return err
	}
	if body.Username != nil {
		if err := httpx.NonBlank(*body.Username, 5, 50, "username"); err != nil {
			return err
		}
		updates["username"] = *body.Username
	}
	if body.Email != nil {
		if err := httpx.NonBlank(*body.Email, 4, 100, "email"); err != nil {
			return err
		}
		updates["email"] = *body.Email
	}
	if body.PhoneNumber != nil {
		if err := httpx.NonBlank(*body.PhoneNumber, 4, 100, "phoneNumber"); err != nil {
			return err
		}
		updates["phone_number"] = *body.PhoneNumber
	}
	if body.FullName != nil {
		if err := httpx.NonBlank(*body.FullName, 1, 100, "fullName"); err != nil {
			return err
		}
		updates["full_name"] = *body.FullName
	}
	if body.Gender != nil {
		if *body.Gender != "MAN" && *body.Gender != "WOMAN" && *body.Gender != "OTHER" {
			return fiber.NewError(400, "Invalid gender")
		}
		updates["gender"] = *body.Gender
	}
	if body.IsActive != nil {
		updates["is_active"] = *body.IsActive
	}
	if body.Password != nil {
		if err := validatePassword(*body.Password); err != nil {
			return err
		}
		hash, err := bcrypt.GenerateFromPassword([]byte(*body.Password), 10)
		if err != nil {
			return err
		}
		updates["password"] = string(hash)
	}
	if len(updates) > 0 {
		if err := s.db.Model(&u).Updates(updates).Error; err != nil {
			return err
		}
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "User")})
}

type PasswordInput struct {
	ExistingPassword string `json:"exsitingPassword"`
	NewPassword      string `json:"newPassword"`
}

func (s *Service) updatePassword(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var body PasswordInput
	if err := httpx.Bind(c, &body); err != nil {
		return err
	}
	if err := validatePassword(body.ExistingPassword); err != nil {
		return err
	}
	if err := validatePassword(body.NewPassword); err != nil {
		return err
	}
	var u domain.User
	if err := s.db.First(&u, "id = ?", id).Error; err != nil {
		return err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(body.ExistingPassword)) != nil {
		return fiber.NewError(400, "🔐 Existing password is incorrect. Please try again with the correct one! 🙅‍♂️")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(body.NewPassword)) == nil {
		return fiber.NewError(400, "🔁 New password can't be the same as your current password. Try something different! 😉")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(body.NewPassword), 10)
	if err != nil {
		return err
	}
	if err := s.db.Model(&u).Update("password", string(hash)).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "User Password")})
}
func (s *Service) updateImage(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	file, err := c.FormFile("file")
	if err != nil {
		return fiber.NewError(400, "file is required")
	}
	if file.Size > 2*1024*1024 {
		return fiber.NewError(400, "file too large")
	}
	mime := file.Header.Get("Content-Type")
	if mime != "image/jpg" && mime != "image/jpeg" && mime != "image/png" && mime != "image/webp" {
		return fiber.NewError(400, "Only image files are allowed!")
	}
	f, err := file.Open()
	if err != nil {
		return err
	}
	defer f.Close()
	bytes, err := io.ReadAll(io.LimitReader(f, 2*1024*1024+1))
	if err != nil {
		return err
	}
	if len(bytes) > 2*1024*1024 {
		return fiber.NewError(400, "file too large")
	}
	var u domain.User
	if err := s.db.First(&u, "id = ?", id).Error; err != nil {
		return err
	}
	if u.ImageURL != nil {
		if err := s.upload.Delete(context.Background(), id); err != nil {
			return fiber.NewError(400, "Failed to delete existing image")
		}
	}
	imageURL, err := s.upload.Upload(context.Background(), id, file.Filename, bytes)
	if err != nil {
		return fiber.NewError(400, "Failed to upload user image")
	}
	if err := s.db.Model(&u).Update("image_url", imageURL).Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "User Image")})
}
func (s *Service) remove(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	r := s.db.Delete(&domain.User{}, "id = ?", id)
	if r.Error != nil {
		return r.Error
	}
	if r.RowsAffected == 0 {
		return httpx.NotFound("User")
	}
	return c.JSON(httpx.Response{Message: httpx.Message("delete", "User")})
}
