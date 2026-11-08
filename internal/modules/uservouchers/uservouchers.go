package uservouchers

import (
	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/promotionutil"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/gofiber/fiber/v3"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"time"
)

type Service struct {
	db       *gorm.DB
	sessions *sessions.Service
}

func NewService(db *gorm.DB, sessions *sessions.Service) *Service {
	return &Service{db: db, sessions: sessions}
}

// The same current-database authorization is shared with Roles, Users and
// Promotions, so renaming a role cannot silently bypass a separate guard.
func (s *Service) ensureAdmin(c fiber.Ctx) error {
	who, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	if !who.Admin {
		return httpx.Forbidden()
	}
	return nil
}

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/user-vouchers")
	r.Use(s.sessions.RequireActor)
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
	if err := promotionutil.ValidateFields(c, promotionutil.UserVoucherFields(), true); err != nil {
		return err
	}
	var body CreateInput
	if err := httpx.Bind(c, &body); err != nil {
		return err
	}
	if err := promotionutil.Required(c, "voucherId"); err != nil {
		return err
	}
	who, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
	userID := who.UserID
	if body.UserID != "" {
		if !who.Admin && body.UserID != who.UserID {
			return httpx.Forbidden()
		}
		userID = body.UserID
	}
	// Serialize claim attempts for a voucher. PostgreSQL row locks prevent
	// concurrent requests from both passing capacity and duplicate checks.
	err = s.db.Transaction(func(tx *gorm.DB) error {
		var voucher domain.Voucher
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&voucher, "id = ?", body.VoucherID).Error; err != nil {
			return err
		}
		now := time.Now().UTC()
		if !voucher.IsActive {
			return fiber.NewError(409, "Voucher is inactive")
		}
		if now.Before(voucher.StartDate) {
			return fiber.NewError(409, "Voucher is not yet valid")
		}
		if !now.Before(voucher.EndDate) {
			return fiber.NewError(409, "Voucher has expired")
		}
		var user domain.User
		if err := tx.First(&user, "id = ?", userID).Error; err != nil {
			return err
		}
		if !user.IsActive {
			return fiber.NewError(409, "User account is inactive")
		}
		var duplicate int64
		if err := tx.Model(&domain.UserVoucher{}).Where("user_id = ? AND voucher_id = ?", userID, voucher.ID.ID).Count(&duplicate).Error; err != nil {
			return err
		}
		if duplicate > 0 {
			return fiber.NewError(409, "Voucher already claimed by this user")
		}
		var totalClaims int64
		if err := tx.Model(&domain.UserVoucher{}).Where("voucher_id = ?", voucher.ID.ID).Count(&totalClaims).Error; err != nil {
			return err
		}
		usedOrReserved := totalClaims
		if int64(voucher.RedeemedCount) > usedOrReserved {
			usedOrReserved = int64(voucher.RedeemedCount)
		}
		if voucher.MaxRedeemed <= 0 || usedOrReserved >= int64(voucher.MaxRedeemed) {
			return fiber.NewError(409, "Voucher redemption limit reached")
		}
		// Caller-supplied claimedAt/expiredAt are accepted for legacy request
		// compatibility but never persisted. Server controls both timestamps.
		m := domain.UserVoucher{UserID: userID, VoucherID: voucher.ID.ID, Status: "CLAIMED", ClaimedAt: now, ExpiredAt: voucher.EndDate.UTC()}
		return tx.Create(&m).Error
	})
	if err != nil {
		return err
	}
	return c.Status(201).JSON(httpx.Response{Message: httpx.Message("create", "User Voucher")})
}

func (s *Service) findAll(c fiber.Ctx) error {
	if err := s.ensureAdmin(c); err != nil {
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
		if err == gorm.ErrRecordNotFound {
			return httpx.NotFound("User Voucher")
		}
		return err
	}
	who, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
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
	if err := s.ensureAdmin(c); err != nil {
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
	who, err := s.sessions.Actor(c)
	if err != nil {
		return err
	}
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
	if err := s.ensureAdmin(c); err != nil {
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
