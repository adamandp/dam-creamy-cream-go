package unit_test

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/adamandp/dam-creamy-cream-go/internal/domain"
)

// Prisma emits millisecond-precision UTC strings, including .000Z. Native
// time.Time would omit all three digits for the deterministic seed records.
func TestRawModelJSONPreservesNestDateTimeContract(t *testing.T) {
	utc := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	stamp := domain.Stamp{CreatedAt: utc, UpdatedAt: utc}
	date := utc
	cases := []struct {
		name   string
		model  any
		fields map[string]any
	}{
		{"Category", domain.Category{Name: "Ice Cream", Stamp: stamp}, map[string]any{
			"createdAt": "2026-01-01T00:00:00.000Z", "updatedAt": "2026-01-01T00:00:00.000Z", "name": "Ice Cream",
		}},
		{"User", domain.User{Username: "test", Password: "bcrypt-must-be-private", BirthDate: &date, Stamp: stamp}, map[string]any{
			"createdAt": "2026-01-01T00:00:00.000Z", "birthDate": "2026-01-01T00:00:00.000Z", "username": "test",
		}},
		{"Shipment", domain.Shipment{Estimated: utc.Add(24 * time.Hour), Stamp: stamp}, map[string]any{
			"createdAt": "2026-01-01T00:00:00.000Z", "estimated": "2026-01-02T00:00:00.000Z", "deliveredAt": nil,
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(tc.model)
			if err != nil {
				t.Fatal(err)
			}
			var data map[string]any
			if err := json.Unmarshal(raw, &data); err != nil {
				t.Fatal(err)
			}
			if _, leaked := data["password"]; leaked {
				t.Fatal("model serialization disclosed an authentication password")
			}
			for field, want := range tc.fields {
				if data[field] != want {
					t.Fatalf("%s: got %#v, expected %#v", field, data[field], want)
				}
			}
		})
	}
}
