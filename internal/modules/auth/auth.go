package auth

import (
	"context"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Service struct {
	db         *gorm.DB
	sessions   *sessions.Service
	production bool
}

func NewService(db *gorm.DB, s *sessions.Service, production bool) *Service {
	return &Service{db: db, sessions: s, production: production}
}
func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/auth")
	r.Post("/dummy-customer-login", s.dummyLogin)
	r.Post("/logout", s.logout)
	r.Post("/logout-all", s.logoutAll)
}
func (s *Service) dummyLogin(c fiber.Ctx) error {
	var u domain.User
	if err := s.db.Where("username = ? OR email = ? OR phone_number = ?", "DummyUser", "DummyUser", "DummyUser").First(&u).Error; err != nil {
		return err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte("@Password123")); err != nil {
		return fiber.NewError(401, "🚫 Login failed. Please check your identifier and password, then try again! 🔍")
	}
	tokens, err := s.sessions.NewTokens(context.Background(), u.ID.ID, u.RoleID, c.Cookies("refreshToken"), c.Get("User-Agent"), c.IP())
	if err != nil {
		return err
	}
	c.Cookie(&fiber.Cookie{Name: "refreshToken", Value: tokens.RefreshToken, HTTPOnly: true, Secure: s.production, SameSite: "Lax", Path: "/", Expires: time.Now().Add(7 * 24 * time.Hour)})
	return c.Status(201).JSON(httpx.Response{Message: "🔓 Welcome back, " + u.Username + "! You have logged in successfully! 🎉", Data: fiber.Map{"accessToken": tokens.AccessToken}})
}
func (s *Service) clearCookie(c fiber.Ctx) {
	c.Cookie(&fiber.Cookie{Name: "refreshToken", Value: "", HTTPOnly: true, Secure: s.production, SameSite: "Lax", Path: "/", Expires: time.Unix(0, 0)})
}
func (s *Service) logout(c fiber.Ctx) error {
	// Clear the browser cookie even when token revocation fails.
	s.clearCookie(c)
	if err := s.sessions.Logout(context.Background(), c.Cookies("refreshToken")); err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Message: "🎉 You have logged out successfully! see u lateer"})
}
func (s *Service) logoutAll(c fiber.Ctx) error {
	s.clearCookie(c)
	if err := s.sessions.LogoutAll(context.Background(), c.Cookies("refreshToken")); err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Message: "🎉 You have logged out all sessions successfully! see u lateer"})
}
