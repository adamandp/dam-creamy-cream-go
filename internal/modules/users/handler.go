package users

import (
	"context"
	"io"
	"log/slog"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

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
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound("User")
		}
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "User"), Data: u})
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
	if err := httpx.BindStrict(c, &body); err != nil {
		return err
	}
	if err := httpx.RejectNull(c, "roleId", "username", "fullName", "password", "isActive"); err != nil {
		return err
	}
	actor, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	// Password changes must use PUT /users/:id/password with the existing
	// password check. Neither users nor admins can bypass it via PATCH.
	if body.Password != nil {
		return httpx.Forbidden()
	}
	// Customers may edit their own safe profile fields, but may not change
	// role assignment, account status, or any other user's account.
	if !actor.Admin && (body.RoleID != nil || body.IsActive != nil) {
		return httpx.Forbidden()
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
		if err := httpx.Name(*body.Username, 5, 50, "username"); err != nil {
			return err
		}
		updates["username"] = *body.Username
	}
	if body.Email != nil {
		if err := httpx.Email(*body.Email); err != nil {
			return err
		}
		updates["email"] = *body.Email
	}
	if body.PhoneNumber != nil {
		if err := httpx.Phone(*body.PhoneNumber); err != nil {
			return err
		}
		updates["phone_number"] = *body.PhoneNumber
	}
	if body.BirthDate != nil {
		birthDate, err := parseBirthDate(*body.BirthDate)
		if err != nil {
			return fiber.NewError(400, "Invalid birthDate")
		}
		updates["birth_date"] = birthDate
	}
	if body.FullName != nil {
		if err := httpx.Name(*body.FullName, 1, 100, "fullName"); err != nil {
			return err
		}
		updates["full_name"] = *body.FullName
	}
	if body.Gender != nil {
		switch *body.Gender {
		case "MAN":
			updates["gender"] = "MALE"
		case "WOMAN":
			updates["gender"] = "FEMALE"
		case "OTHER":
			updates["gender"] = "OTHER"
		default:
			return fiber.NewError(400, "Invalid gender")
		}
	}
	if body.IsActive != nil {
		updates["is_active"] = *body.IsActive
	}
	if httpx.ExplicitNull(c, "email") {
		updates["email"] = nil
	}
	if httpx.ExplicitNull(c, "phoneNumber") {
		updates["phone_number"] = nil
	}
	if httpx.ExplicitNull(c, "gender") {
		updates["gender"] = nil
	}
	if httpx.ExplicitNull(c, "birthDate") {
		updates["birth_date"] = nil
	}
	if err := httpx.ValidateUpdate(map[string]any{"role_id": u.RoleID, "username": u.Username, "email": valueOrNil(u.Email), "phone_number": valueOrNil(u.PhoneNumber), "full_name": u.FullName, "gender": valueOrNil(u.Gender), "is_active": u.IsActive, "birth_date": dateOrNil(u.BirthDate)}, updates); err != nil {
		return err
	}
	if len(updates) > 0 {
		if err := s.db.Model(&u).Updates(updates).Error; err != nil {
			return err
		}
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "User")})
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
	if bcrypt.CompareHashAndPassword([]byte(u.Password), bcryptInput(body.ExistingPassword)) != nil {
		return fiber.NewError(400, "🔐 Existing password is incorrect. Please try again with the correct one! 🙅‍♂️")
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), bcryptInput(body.NewPassword)) == nil {
		return fiber.NewError(400, "🔁 New password can't be the same as your current password. Try something different! 😉")
	}
	hash, err := bcrypt.GenerateFromPassword(bcryptInput(body.NewPassword), 10)
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
	previousImageURL := ""
	if u.ImageURL != nil {
		previousImageURL = *u.ImageURL
	}
	ctx, cancel := context.WithTimeout(c.Context(), 30*time.Second)
	defer cancel()
	imageURL, err := s.upload.Upload(ctx, "users", id, bytes)
	if err != nil {
		return fiber.NewError(400, "Failed to upload user image")
	}
	if err := s.db.WithContext(ctx).Model(&u).Update("image_url", imageURL).Error; err != nil {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if cleanupErr := s.upload.Delete(cleanupCtx, imageURL); cleanupErr != nil {
			slog.Warn("failed to remove orphaned user image", "error", cleanupErr)
		}
		return err
	}
	if previousImageURL != "" {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cleanupCancel()
		if err := s.upload.Delete(cleanupCtx, previousImageURL); err != nil {
			slog.Warn("failed to remove previous user image", "error", err)
		}
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
