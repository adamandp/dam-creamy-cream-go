package unit_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/adamandp/dam-creamy-cream-go/internal/httpx"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/payments"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/sessions"
	"github.com/adamandp/dam-creamy-cream-go/internal/modules/shipments"
	"github.com/gofiber/fiber/v3"
)

func TestRajaOngkirFormContractAndResponse(t *testing.T) {
	var calls int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/api/v1/calculate/domestic-cost" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Method != http.MethodPost || r.Header.Get("key") != "secret-shipping-key" {
			t.Errorf("unexpected upstream auth/method %s / %q", r.Method, r.Header.Get("key"))
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("upstream content type = %q", r.Header.Get("Content-Type"))
		}
		if err := r.ParseForm(); err != nil {
			t.Errorf("parse form: %v", err)
		}
		if r.FormValue("origin") != "123" || r.FormValue("destination") != "456" || r.FormValue("weight") != "2500" {
			t.Errorf("input form = %v", r.Form)
		}
		if !strings.Contains(r.FormValue("courier"), "jne:sicepat:ide") || !strings.Contains(r.FormValue("courier"), ":dse") {
			t.Errorf("courier list missing: %q", r.FormValue("courier"))
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"meta":{"message":"Success","code":200,"status":"success"},"data":[{"name":"JNE","code":"jne","service":"REG","description":"Regular","cost":12000,"etd":"2-3 days"}]}`))
	}))
	defer upstream.Close()
	client, err := shipments.NewRajaOngkir(upstream.URL+"/api/v1", "secret-shipping-key", upstream.Client())
	if err != nil {
		t.Fatal(err)
	}
	got, err := client.Calculate(context.Background(), shipments.CostInput{Origin: 123, Destination: 456, Weight: 2500})
	if err != nil {
		t.Fatal(err)
	}
	if calls != 1 || got.Status != 200 || got.Message != "Success" || len(got.Data) != 1 || got.Data[0].Cost != 12000 {
		t.Fatalf("unexpected quote: calls=%d quote=%+v", calls, got)
	}
}

func TestRajaOngkirFailurePaths(t *testing.T) {
	for _, v := range []struct{ base, key string }{
		{"https://api.example.com", ""},
		{"", "secret"},
		{"https://user:password@example.com/api/v1", "secret"},
		{"file:///etc/passwd", "secret"},
	} {
		if _, err := shipments.NewRajaOngkir(v.base, v.key, nil); err == nil {
			t.Errorf("accepted invalid provider settings: %+v", v)
		}
	}
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "upstream refused", 503)
	}))
	defer upstream.Close()
	client, err := shipments.NewRajaOngkir(upstream.URL, "secret", upstream.Client())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.Calculate(context.Background(), shipments.CostInput{Origin: 1, Destination: 2, Weight: 3}); err == nil {
		t.Fatal("upstream HTTP error must not become a valid quote")
	}
}

type quoteFixture struct {
	calls int
}

func (q *quoteFixture) Calculate(_ context.Context, in shipments.CostInput) (shipments.ShippingQuote, error) {
	q.calls++
	return shipments.ShippingQuote{
		Message: "Priced", Status: 200,
		Data: []shipments.ShippingData{{Code: "jne", Cost: in.Weight}},
	}, nil
}

type quotaFixture struct {
	allowed bool
	err     error
	calls   int
}

func (q *quotaFixture) Allow(_ context.Context, _ string) (bool, error) {
	q.calls++
	return q.allowed, q.err
}

func TestShippingCostEndpointValidationAndLegacyPaymentScaffold(t *testing.T) {
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler})
	quote := &quoteFixture{}
	quota := &quotaFixture{allowed: true}
	ses := sessions.NewService(nil, "unused-access", "unused-refresh")
	shipments.NewService(nil, ses, quote, quota).Routes(app)
	payments.NewService().Routes(app)
	call := func(method, path, body string, status int) string {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if body != "" {
			req.Header.Set("Content-Type", "application/json")
		}
		response, err := app.Test(req)
		if err != nil {
			t.Fatal(err)
		}
		defer response.Body.Close()
		payload, _ := io.ReadAll(response.Body)
		if response.StatusCode != status {
			t.Fatalf("%s %s returned %d, expected %d: %s", method, path, response.StatusCode, status, payload)
		}
		return string(payload)
	}
	for _, body := range []string{
		`{"origin":1,"destination":2}`,
		`{"origin":0,"destination":2,"weight":100}`,
		`{"origin":1,"destination":2,"weight":-1}`,
		`{"origin":1,"destination":2,"weight":100,"unknown":true}`,
		`{"origin":"not a number","destination":2,"weight":100}`,
		`{"origin":1.5,"destination":2,"weight":100}`,
		`{"origin":1,"destination":"2.25","weight":100}`,
		`{"origin":1,"destination":2,"weight":0.5}`,
		`{"origin":1,"destination":2,"weight":2147483648}`,
	} {
		call("POST", "/shipments/cost", body, 400)
	}
	if quote.calls != 0 || quota.calls != 0 {
		t.Fatal("invalid quote input reached rate limiter or provider")
	}
	out := call("POST", "/shipments/cost", `{"origin":"1","destination":2,"weight":100}`, 201)
	var response struct {
		Status int `json:"status"`
		Data   []struct {
			Cost float64 `json:"cost"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(out), &response); err != nil || response.Status != 200 || len(response.Data) != 1 || response.Data[0].Cost != 100 {
		t.Fatalf("unexpected quote response: %v %s", err, out)
	}
	if quote.calls != 1 {
		t.Fatalf("want exactly one provider call, got %d", quote.calls)
	}
	quota.allowed = false
	out = call("POST", "/shipments/cost", `{"origin":1,"destination":2,"weight":100}`, 429)
	if !strings.Contains(out, "Too Many Requests") || quote.calls != 1 {
		t.Fatalf("blocked rate limit must not call provider: %s, calls=%d", out, quote.calls)
	}
	quota.err = errors.New("redis unavailable")
	out = call("POST", "/shipments/cost", `{"origin":1,"destination":2,"weight":100}`, 503)
	if !strings.Contains(out, "Service Unavailable") || quote.calls != 1 {
		t.Fatalf("rate limiter failure must fail closed: %s, calls=%d", out, quote.calls)
	}
	if v := call("POST", "/payments", `{}`, 201); v != "This action adds a new payment" {
		t.Fatalf("Nest placeholder mismatch: %q", v)
	}
	if v := call("GET", "/payments/42", "", 200); v != "This action returns a #42 payment" {
		t.Fatalf("Nest placeholder mismatch: %q", v)
	}
	if v := call("PATCH", "/payments/42", `{}`, 200); v != "This action updates a #42 payment" {
		t.Fatalf("Nest placeholder mismatch: %q", v)
	}
	if v := call("DELETE", "/payments/42", "", 200); v != "This action removes a #42 payment" {
		t.Fatalf("Nest placeholder mismatch: %q", v)
	}
	if v := call("GET", "/payments", "", 200); v != "This action returns all payments" {
		t.Fatalf("Nest placeholder mismatch: %q", v)
	}
}
