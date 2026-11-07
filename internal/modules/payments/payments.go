// Package payments preserves the current NestJS PaymentsService HTTP behavior.
// The upstream service is a scaffold: these routes do not charge, update,
// or delete payments. Checkout creates real PENDING payment records.
package payments

import (
	"errors"
	"math"
	"math/big"
	"regexp"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
)

type Service struct{}

func NewService() *Service { return &Service{} }

func (s *Service) Routes(app fiber.Router) {
	r := app.Group("/payments")
	r.Post("/", func(c fiber.Ctx) error { return c.Status(201).SendString("This action adds a new payment") })
	r.Get("/", func(c fiber.Ctx) error { return c.SendString("This action returns all payments") })
	r.Get("/:id", func(c fiber.Ctx) error {
		return c.SendString("This action returns a #" + legacyNumericID(c.Params("id")) + " payment")
	})
	r.Patch("/:id", func(c fiber.Ctx) error {
		return c.SendString("This action updates a #" + legacyNumericID(c.Params("id")) + " payment")
	})
	r.Delete("/:id", func(c fiber.Ctx) error {
		return c.SendString("This action removes a #" + legacyNumericID(c.Params("id")) + " payment")
	})
}

var (
	jsDecimalNumber = regexp.MustCompile(`^[+-]?(?:[0-9]+(?:\.[0-9]*)?|\.[0-9]+)(?:[eE][+-]?[0-9]+)?$`)
	jsBinaryDigits  = regexp.MustCompile(`^[01]+$`)
	jsOctalDigits   = regexp.MustCompile(`^[0-7]+$`)
	jsHexDigits     = regexp.MustCompile(`^[0-9a-fA-F]+$`)
)

// NestJS uses the JavaScript unary + on @Param('id') before interpolating the
// number into a scaffold string. Handle Number()'s decimal, radix and infinity
// forms without treating Go-specific "Inf"/hex-float syntax as valid JS input.
func legacyNumericID(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "0"
	}
	switch s {
	case "Infinity", "+Infinity":
		return "Infinity"
	case "-Infinity":
		return "-Infinity"
	}
	if len(s) > 2 && s[0] == '0' {
		base := 0
		switch s[1] {
		case 'x', 'X':
			base = 16
		case 'b', 'B':
			base = 2
		case 'o', 'O':
			base = 8
		}
		if base != 0 {
			// JavaScript's Number() accepts only unsigned digits after a radix
			// prefix. big.Int.SetString also accepts a sign (e.g. "0x-1"
			// becomes -1 after removing the prefix), which would break parity.
			digits := s[2:]
			if (base == 2 && !jsBinaryDigits.MatchString(digits)) ||
				(base == 8 && !jsOctalDigits.MatchString(digits)) ||
				(base == 16 && !jsHexDigits.MatchString(digits)) {
				return "NaN"
			}
			integer, valid := new(big.Int).SetString(digits, base)
			if !valid {
				return "NaN"
			}
			number, _ := new(big.Float).SetInt(integer).Float64()
			return formatJSNumber(number)
		}
	}
	if !jsDecimalNumber.MatchString(s) {
		return "NaN"
	}
	number, err := strconv.ParseFloat(s, 64)
	if err != nil && !errors.Is(err, strconv.ErrRange) {
		return "NaN"
	}
	return formatJSNumber(number)
}

func formatJSNumber(number float64) string {
	if math.IsNaN(number) {
		return "NaN"
	}
	if math.IsInf(number, 1) {
		return "Infinity"
	}
	if math.IsInf(number, -1) {
		return "-Infinity"
	}
	if number == 0 {
		return "0"
	}
	absolute := math.Abs(number)
	if absolute >= 1e-6 && absolute < 1e21 {
		return strconv.FormatFloat(number, 'f', -1, 64)
	}
	exponential := strconv.FormatFloat(number, 'e', -1, 64)
	parts := strings.SplitN(exponential, "e", 2)
	if len(parts) != 2 {
		return exponential
	}
	exponent, err := strconv.Atoi(parts[1])
	if err != nil {
		return exponential
	}
	if exponent >= 0 {
		return parts[0] + "e+" + strconv.Itoa(exponent)
	}
	return parts[0] + "e" + strconv.Itoa(exponent)
}
