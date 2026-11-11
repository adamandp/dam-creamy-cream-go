package integration_test

import (
	"context"
	"testing"
	"time"

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

func TestExpiredRefreshCookieDoesNotBindLoginToPreviousUser(t *testing.T) {
	db := connectPrismaFixture(t)
	tx := db.Begin()
	if tx.Error != nil {
		t.Fatal(tx.Error)
	}
	defer tx.Rollback()
	role := domain.Role{Name: "SharedBrowserFixture"}
	if err := tx.Create(&role).Error; err != nil {
		t.Fatal(err)
	}
	alice := domain.User{RoleID: role.ID.ID, Username: "OldBrowserUser", FullName: "Old Browser", Password: "fixture", IsActive: true}
	bob := domain.User{RoleID: role.ID.ID, Username: "NewBrowserUser", FullName: "New Browser", Password: "fixture", IsActive: true}
	for _, user := range []*domain.User{&alice, &bob} {
		if err := tx.Create(user).Error; err != nil {
			t.Fatal(err)
		}
	}
	ses := sessions.NewService(tx, "browser-access", "browser-refresh")
	old, err := ses.NewTokens(context.Background(), alice.ID.ID, role.ID.ID, "", "Browser A", "127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.Model(&domain.UserToken{}).Where("token = ?", old.RefreshToken).
		Update("expiredAt", time.Now().UTC().Add(-time.Minute)).Error; err != nil {
		t.Fatal(err)
	}
	newTokens, err := ses.NewTokens(context.Background(), bob.ID.ID, role.ID.ID, old.RefreshToken, "Browser B", "127.0.0.2")
	if err != nil {
		t.Fatalf("logging in with an expired cookie should work: %v", err)
	}
	claims, err := ses.ActiveRefresh(context.Background(), newTokens.RefreshToken)
	if err != nil || claims.Subject != bob.ID.ID {
		t.Fatalf("refreshed session must belong to new user: claims=%+v err=%v", claims, err)
	}
	var stored domain.UserToken
	if err := tx.Where("token = ?", newTokens.RefreshToken).First(&stored).Error; err != nil || stored.UserID != bob.ID.ID {
		t.Fatalf("cookie reissue retained previous owner: %+v err=%v", stored, err)
	}
}
