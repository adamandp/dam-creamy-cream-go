package integration_test

import (
	"context"
	"testing"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
)

func TestRefreshJWTUniqueJTIWithinSecond(t *testing.T) {
	db := connectPrismaFixture(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	role := domain.Role{Name: "JWTCustomer"}
	if err := tx.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	user := domain.User{RoleID: role.ID.ID, Username: "JTIUniqueUser", FullName: "JWT User", Password: "fixture", IsActive: true}
	if err := tx.Create(&user).Error; err != nil {
		t.Fatal(err)
	}
	ses := sessions.NewService(tx, "jti-access", "jti-refresh")
	seen := map[string]bool{}
	var previous string
	for i := 0; i < 12; i++ {
		tokens, err := ses.NewTokens(context.Background(), user.ID.ID, role.ID.ID, "", "Test", "127.0.0.1")
		if err != nil {
			t.Fatalf("login %d: %v", i, err)
		}
		if len(tokens.RefreshToken) > 255 {
			t.Fatalf("refresh JWT exceeds VARCHAR(255): %d", len(tokens.RefreshToken))
		}
		claims, err := ses.VerifyRefresh(tokens.RefreshToken)
		if err != nil {
			t.Fatal(err)
		}
		if claims.ID == "" || seen[claims.ID] || tokens.RefreshToken == previous {
			t.Fatalf("duplicate or missing refresh JWT jti on login %d: %q", i, claims.ID)
		}
		if claims.Subject != user.ID.ID || claims.Role != role.ID.ID {
			t.Fatal("refresh claims changed")
		}
		if _, err := ses.ActiveRefresh(context.Background(), tokens.RefreshToken); err != nil {
			t.Fatalf("session not persisted: %v", err)
		}
		seen[claims.ID] = true
		previous = tokens.RefreshToken
	}
	var rows int64
	if err := tx.Model(&domain.UserToken{}).Where("user_id = ?", user.ID.ID).Count(&rows).Error; err != nil {
		t.Fatal(err)
	}
	if rows != 12 {
		t.Fatalf("expected 12 distinct persisted sessions, got %d", rows)
	}
}
