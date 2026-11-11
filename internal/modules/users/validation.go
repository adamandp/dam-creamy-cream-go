package users

import (
	"fmt"
	"regexp"
	"time"
	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/gofiber/fiber/v3"
)

var lowercase = regexp.MustCompile(`[a-z]`)

var uppercase = regexp.MustCompile(`[A-Z]`)

var digit = regexp.MustCompile(`[0-9]`)

var special = regexp.MustCompile(`[!@#$%^&*]`)

func validatePassword(password string) error {
	if httpx.JSStringLength(password) < 8 || httpx.JSStringLength(password) > 255 || !lowercase.MatchString(password) || !uppercase.MatchString(password) || !digit.MatchString(password) || !special.MatchString(password) {
		return fiber.NewError(400, "Password must be 8-255 chars with uppercase, lowercase, number and special character")
	}
	return nil
}

// Node's bcrypt implementation hashes at most the first 72 password bytes.
// The Go implementation rejects inputs longer than 72 bytes instead. Apply
// the same truncation for both verification and hashing to keep existing Nest
// password hashes usable and prevent a previously valid password returning 500.

func parseBirthDate(v string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, "2006-01-02"} {
		d, err := time.Parse(layout, v)
		if err == nil {
			return d, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid date")
}
