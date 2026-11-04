package uservouchers

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"strings"
)

type Service struct {
	db       *gorm.DB
	sessions *sessions.Service
}

func NewService(db *gorm.DB, sessions *sessions.Service) *Service {
	return &Service{db: db, sessions: sessions}
}

// An actor is resolved from a signed JWT and the CURRENT database role.
type actor struct {
	UserID string
	Admin  bool
}

func (s *Service) requireActor(c fiber.Ctx) error {
	claims, err := s.sessions.AccessFor(c)
	if err != nil {
		return httpx.Unauthorized()
	}
	var user domain.User
	if err := s.db.First(&user, "id = ?", claims.Subject).Error; err != nil || !user.IsActive || user.RoleID != claims.Role {
		return httpx.Unauthorized()
	}
	var role domain.Role
	if err := s.db.First(&role, "id = ?", user.RoleID).Error; err != nil {
		return httpx.Unauthorized()
	}
	name := strings.ToLower(strings.TrimSpace(role.Name))
	isAdmin := name == "admin" || name == "administrator" || name == "super admin" || name == "superadmin"
	c.Locals("userVoucherActor", actor{UserID: user.ID.ID, Admin: isAdmin})
	return c.Next()
}
func currentActor(c fiber.Ctx) actor {
	return c.Locals("userVoucherActor").(actor)
}
func requireAdmin(c fiber.Ctx) error {
	if !currentActor(c).Admin {
		return httpx.Forbidden()
	}
	return nil
}

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/user-vouchers")
	r.Use(s.requireActor)
	r.Post("/", s.create)
	r.Get("/", s.findAll)
	r.Get("/user", s.byUser)
	r.Get("/voucher/:id", s.byVoucher)
	r.Patch("/:id", s.use)
	r.Get("/:id", s.byID)
}

type CreateInput struct {
	UserID    string             `json:"userId"`
	VoucherID string             `json:"voucherId"`
	ClaimedAt promotionutil.Date `json:"claimedAt"`
	ExpiredAt promotionutil.Date `json:"expiredAt"`
}

func (s *Service) create(c fiber.Ctx) error {
	if err := promotionutil.ValidateFields(c, promotionutil.UserVoucherFields(), false); err != nil {
		return err
	}
	var body CreateInput
	if err := httpx.Bind(c, &body); err != nil {
		return err
	}
	if err := promotionutil.Required(c, "userId", "voucherId", "claimedAt", "expiredAt"); err != nil {
		return err
	}
	who := currentActor(c)
	if !who.Admin && who.UserID != body.UserID {
		return httpx.Forbidden()
	}
	var user domain.User
	if err := s.db.First(&user, "id = ?", body.UserID).Error; err != nil {
		return err
	}
	var voucher domain.Voucher
	if err := s.db.First(&voucher, "id = ?", body.VoucherID).Error; err != nil {
		return err
	}
	// Status defaults to CLAIMED; keep user-provided claimedAt and expiredAt.
	m := domain.UserVoucher{
		UserID: body.UserID, VoucherID: body.VoucherID, Status: "CLAIMED",
		ClaimedAt: body.ClaimedAt.Time, ExpiredAt: body.ExpiredAt.Time,
	}
	if err := s.db.Create(&m).Error; err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Message: httpx.Message("create", "User Voucher")})
}

func (s *Service) findAll(c fiber.Ctx) error {
	if err := requireAdmin(c); err != nil {
		return err
	}
	page, limit, err := httpx.Page(c)
	if err != nil {
		return err
	}
	var data []domain.UserVoucher
	var total int64
	if err := s.db.Model(&domain.UserVoucher{}).Count(&total).Error; err != nil {
		return err
	}
	if total == 0 {
		return httpx.NotFound("User Voucher")
	}
	if err := s.db.Offset((page - 1) * limit).Limit(limit).Find(&data).Error; err != nil {
		return err
	}
	mapped, err := promotionutil.AsViews(data)
	if err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "User Voucher"), Data: mapped, Paging: httpx.Pagination(page, limit, total)})
}

func (s *Service) byID(c fiber.Ctx) error {
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var m domain.UserVoucher
	if err := s.db.First(&m, "id = ?", id).Error; err != nil {
		return err
	}
	who := currentActor(c)
	if !who.Admin && who.UserID != m.UserID {
		return httpx.Forbidden()
	}
	view, err := promotionutil.AsView(m)
	if err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "User Voucher"), Data: view})
}

func (s *Service) byVoucher(c fiber.Ctx) error {
	if err := requireAdmin(c); err != nil {
		return err
	}
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var data []domain.UserVoucher
	if err := s.db.Where("voucher_id = ?", id).Find(&data).Error; err != nil {
		return err
	}
	mapped, err := promotionutil.AsViews(data)
	if err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "User Voucher"), Data: mapped})
}

func (s *Service) byUser(c fiber.Ctx) error {
	who := currentActor(c)
	var assignments []domain.UserVoucher
	if err := s.db.Where("user_id = ?", who.UserID).Find(&assignments).Error; err != nil {
		return err
	}
	// Avoid changing the default UserVoucher JSON shape on the other routes.
	// The /user endpoint alone includes the related voucher under "vouchers".
	ids := make([]string, 0, len(assignments))
	for _, assignment := range assignments {
		ids = append(ids, assignment.VoucherID)
	}
	voucherByID := map[string]map[string]any{}
	if len(ids) > 0 {
		var related []domain.Voucher
		if err := s.db.Where("id IN ?", ids).Find(&related).Error; err != nil {
			return err
		}
		for _, voucher := range related {
			view, err := promotionutil.AsView(voucher)
			if err != nil {
				return err
			}
			voucherByID[voucher.ID.ID] = view
		}
	}
	data := make([]map[string]any, 0, len(assignments))
	for _, assignment := range assignments {
		view, err := promotionutil.AsView(assignment)
		if err != nil {
			return err
		}
		view["vouchers"] = voucherByID[assignment.VoucherID]
		data = append(data, view)
	}
	return c.JSON(httpx.Response{Message: httpx.Message("get", "User Voucher"), Data: data})
}

func (s *Service) use(c fiber.Ctx) error {
	// Customers must not mark claims USED arbitrarily: checkout (when ported)
	// or authorized administrators perform the state transition.
	if err := requireAdmin(c); err != nil {
		return err
	}
	id := c.Params("id")
	if err := httpx.UUID(id); err != nil {
		return err
	}
	var m domain.UserVoucher
	if err := s.db.First(&m, "id = ?", id).Error; err != nil {
		return err
	}
	if err := s.db.Model(&m).Update("status", "USED").Error; err != nil {
		return err
	}
	return c.JSON(httpx.Response{Message: httpx.Message("update", "User Voucher")})
}
