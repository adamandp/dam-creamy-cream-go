package users

import (
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"
)

func TestPasswordMatchesJavaScriptLengthAndNodeBcryptBoundary(t *testing.T) {
	password := "Aa1!" + strings.Repeat("x", 68)
	withSuffix := password + strings.Repeat("🍦", 10)
	if err := validatePassword(withSuffix); err != nil {
		t.Fatalf("valid password longer than 72 bytes must match Nest validation: %v", err)
	}
	if len(bcryptInput(withSuffix)) != 72 {
		t.Fatalf("bcrypt input must be 72 bytes, got %d", len(bcryptInput(withSuffix)))
	}
	hash, err := bcrypt.GenerateFromPassword(bcryptInput(withSuffix), bcrypt.MinCost)
	if err != nil {
		t.Fatalf("bcrypt rejected previously valid password: %v", err)
	}
	if err := bcrypt.CompareHashAndPassword(hash, bcryptInput(password)); err != nil {
		t.Fatalf("Node and Go must agree on the bcrypt 72-byte boundary: %v", err)
	}
	if err := validatePassword("Aa1!" + strings.Repeat("🍦", 126)); err == nil {
		t.Fatal("password over 255 JavaScript UTF-16 units must be rejected")
	}
}
