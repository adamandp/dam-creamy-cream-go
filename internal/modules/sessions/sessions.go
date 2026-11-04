package sessions

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/mileusna/useragent"
	"strings"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/gofiber/fiber/v3"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

type Service struct {
	db            *gorm.DB
	accessSecret  []byte
	refreshSecret []byte
}

func NewService(db *gorm.DB, accessSecret, refreshSecret string) *Service {
	return &Service{db: db, accessSecret: []byte(accessSecret), refreshSecret: []byte(refreshSecret)}
}

type Tokens struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}
type Claims struct {
	Role string `json:"role"`
	jwt.RegisteredClaims
}

func (s *Service) sign(userID, roleID string, ttl time.Duration, secret []byte) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{Role: roleID, RegisteredClaims: jwt.RegisteredClaims{Subject: userID, IssuedAt: jwt.NewNumericDate(time.Now()), ExpiresAt: jwt.NewNumericDate(time.Now().Add(ttl))}})
	return token.SignedString(secret)
}
func (s *Service) parse(token string, secret []byte) (*Claims, error) {
	if token == "" {
		return nil, httpx.Unauthorized()
	}
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(t *jwt.Token) (any, error) { return secret, nil }, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}), jwt.WithExpirationRequired())
	if err != nil || !parsed.Valid {
		return nil, httpx.Unauthorized()
	}
	claims, ok := parsed.Claims.(*Claims)
	if !ok || claims.Subject == "" {
		return nil, httpx.Unauthorized()
	}
	return claims, nil
}
func (s *Service) VerifyAccess(token string) (*Claims, error) { return s.parse(token, s.accessSecret) }
func (s *Service) VerifyRefresh(token string) (*Claims, error) {
	return s.parse(token, s.refreshSecret)
}
func (s *Service) AccessFor(c fiber.Ctx) (*Claims, error) {
	header := c.Get("Authorization")
	if strings.HasPrefix(header, "Bearer ") {
		return s.VerifyAccess(strings.TrimPrefix(header, "Bearer "))
	}
	return nil, httpx.Unauthorized()
}
func (s *Service) CookieUser(c fiber.Ctx) (*Claims, error) {
	return s.VerifyRefresh(c.Cookies("refreshToken"))
}
func (s *Service) NewTokens(ctx context.Context, userID, roleID, existing, agent, ip string) (Tokens, error) {
	access, err := s.sign(userID, roleID, 15*time.Minute, s.accessSecret)
	if err != nil {
		return Tokens{}, err
	}
	refresh, err := s.sign(userID, roleID, 7*24*time.Hour, s.refreshSecret)
	if err != nil {
		return Tokens{}, err
	}
	now := time.Now()
	info, _ := json.Marshal(parseBrowserInfo(ip, "dummy-"+agent))
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if existing != "" {
			var old domain.UserToken
			if err := tx.Where("token = ?", existing).First(&old).Error; err != nil {
				return err
			}
			if old.ExpiredAt.After(now) {
				return fiber.NewError(403, "✅ You are already logged in. No need to log in again! 🎉")
			}
			return tx.Model(&old).Updates(map[string]any{"token": refresh, "expiredAt": now.Add(7 * 24 * time.Hour)}).Error
		}
		return tx.Create(&domain.UserToken{UserID: userID, Token: refresh, BrowserInfo: string(info), ExpiredAt: now.Add(7 * 24 * time.Hour)}).Error
	})
	if err != nil {
		return Tokens{}, err
	}
	return Tokens{AccessToken: access, RefreshToken: refresh}, nil
}
func (s *Service) Refresh(ctx context.Context, token string) (string, error) {
	claims, err := s.VerifyRefresh(token)
	if err != nil {
		return "", err
	}
	var count int64
	if err := s.db.WithContext(ctx).Model(&domain.UserToken{}).Where("token = ? AND \"expiredAt\" > ?", token, time.Now()).Count(&count).Error; err != nil {
		return "", err
	}
	if count == 0 {
		return "", httpx.Unauthorized()
	}
	return s.sign(claims.Subject, claims.Role, 15*time.Minute, s.accessSecret)
}
func (s *Service) Logout(ctx context.Context, token string) error {
	if _, err := s.VerifyRefresh(token); err != nil {
		return err
	}
	res := s.db.WithContext(ctx).Where("token = ?", token).Delete(&domain.UserToken{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return httpx.NotFound("Token")
	}
	return nil
}
func (s *Service) LogoutAll(ctx context.Context, token string) error {
	claims, err := s.VerifyRefresh(token)
	if err != nil {
		return err
	}
	res := s.db.WithContext(ctx).Where("user_id = ?", claims.Subject).Delete(&domain.UserToken{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return httpx.NotFound("Token")
	}
	return nil
}
func (s *Service) Routes(app fiber.Router) { app.Post("/sessions/refresh", s.refreshHandler) }
func (s *Service) refreshHandler(c fiber.Ctx) error {
	token, err := s.Refresh(context.Background(), c.Cookies("refreshToken"))
	if err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Data: fiber.Map{"accessToken": token}})
}

func parseBrowserInfo(ip, agent string) map[string]string {
	ua := useragent.Parse(agent)
	browser := fmt.Sprintf("%s %s", unknown(ua.Name), unknown(ua.Version))
	osInfo := fmt.Sprintf("%s %s", unknown(ua.OS), unknown(ua.OSVersion))
	device := "Unknown"
	if ua.Tablet {
		device = "tablet"
	} else if ua.Mobile {
		device = "mobile"
	}
	return map[string]string{"ip": ip, "browser": browser, "os": osInfo, "device": device}
}
func unknown(v string) string {
	if v == "" {
		return "Unknown"
	}
	return v
}
