package sessions

import (
	"strings"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/gofiber/fiber/v3"
)

// Actor is the verified identity at the time of the request, not an
// authorization decision cached inside an access token.
type Actor struct {
	UserID string
	RoleID string
	Admin  bool
}

const actorLocalKey = "verifiedSessionActor"

// Actor validates the access JWT and re-reads the current role and active
// state from PostgreSQL. A changed role immediately invalidates stale JWTs.
func (s *Service) Actor(c fiber.Ctx) (Actor, error) {
	if cached, ok := c.Locals(actorLocalKey).(Actor); ok {
		return cached, nil
	}
	claims, err := s.AccessFor(c)
	if err != nil {
		return Actor{}, httpx.Unauthorized()
	}
	var user domain.User
	if err := s.db.First(&user, "id = ?", claims.Subject).Error; err != nil {
		return Actor{}, httpx.Unauthorized()
	}
	if !user.IsActive || user.RoleID != claims.Role {
		return Actor{}, httpx.Unauthorized()
	}
	var role domain.Role
	if err := s.db.First(&role, "id = ?", user.RoleID).Error; err != nil {
		return Actor{}, httpx.Unauthorized()
	}
	name := strings.ToLower(strings.TrimSpace(role.Name))
	isAdmin := name == "admin" || name == "administrator" || name == "super admin" || name == "superadmin"
	a := Actor{UserID: user.ID.ID, RoleID: user.RoleID, Admin: isAdmin}
	c.Locals(actorLocalKey, a)
	return a, nil
}
func (s *Service) RequireActor(c fiber.Ctx) error {
	if _, err := s.Actor(c); err != nil {
		return err
	}
	return c.Next()
}
func (s *Service) RequireAdmin(c fiber.Ctx) error {
	actor, err := s.Actor(c)
	if err != nil {
		return err
	}
	if !actor.Admin {
		return httpx.Forbidden()
	}
	return c.Next()
}
func (s *Service) RequireSelfOrAdmin(c fiber.Ctx) error {
	actor, err := s.Actor(c)
	if err != nil {
		return err
	}
	if !actor.Admin && actor.UserID != c.Params("id") {
		return httpx.Forbidden()
	}
	return c.Next()
}
func (s *Service) RequireSelf(c fiber.Ctx) error {
	actor, err := s.Actor(c)
	if err != nil {
		return err
	}
	if actor.UserID != c.Params("id") {
		return httpx.Forbidden()
	}
	return c.Next()
}
