package shipments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const couriers = "jne:sicepat:ide:sap:jnt:ninja:tiki:lion:anteraja:pos:ncs:rex:rpx:sentral:star:wahana:dse"
const maxQuoteResponseBytes int64 = 1 << 20

type ShippingData struct {
	Name        string  `json:"name"`
	Code        string  `json:"code"`
	Service     string  `json:"service"`
	Description string  `json:"description"`
	Cost        float64 `json:"cost"`
	ETD         string  `json:"etd"`
}

type ShippingQuote struct {
	Message string
	Status  int
	Data    []ShippingData
}

type RajaOngkir struct {
	endpoint string
	apiKey   string
	client   *http.Client
}

// Missing settings keep the service available for local development, but
// POST /shipments/cost returns 503 instead of silently using fake prices.
func NewRajaOngkir(baseURL, apiKey string, client *http.Client) (*RajaOngkir, error) {
	if baseURL == "" && apiKey == "" {
		return nil, nil
	}
	if baseURL == "" || apiKey == "" {
		return nil, errors.New("RAJAONGKIR_BASE_URL and RAJAONGKIR_API_KEY must be configured together")
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") ||
		parsed.Host == "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return nil, errors.New("RAJAONGKIR_BASE_URL must be an absolute HTTP(S) URL without credentials or query")
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + "/calculate/domestic-cost"
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	// The provider credential is sent in a custom "key" header. Go copies that
	// header across redirects, including redirects to other hosts. Do not follow
	// any redirects: a 3xx response is an upstream failure, not a new destination.
	// Copy the client so callers' shared HTTP client configuration is untouched.
	safeClient := *client
	safeClient.CheckRedirect = func(_ *http.Request, _ []*http.Request) error {
		return http.ErrUseLastResponse
	}
	return &RajaOngkir{endpoint: parsed.String(), apiKey: apiKey, client: &safeClient}, nil
}

func (r *RajaOngkir) Calculate(ctx context.Context, input CostInput) (ShippingQuote, error) {
	form := url.Values{
		"origin":      {strconv.FormatFloat(input.Origin, 'f', -1, 64)},
		"destination": {strconv.FormatFloat(input.Destination, 'f', -1, 64)},
		"weight":      {strconv.FormatFloat(input.Weight, 'f', -1, 64)},
		"courier":     {couriers},
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, r.endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return ShippingQuote{}, err
	}
	request.Header.Set("key", r.apiKey)
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response, err := r.client.Do(request)
	if err != nil {
		return ShippingQuote{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return ShippingQuote{}, fmt.Errorf("rajaongkir returned HTTP %d", response.StatusCode)
	}
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxQuoteResponseBytes+1))
	if err != nil {
		return ShippingQuote{}, err
	}
	if int64(len(payload)) > maxQuoteResponseBytes {
		return ShippingQuote{}, errors.New("rajaongkir response exceeds size limit")
	}
	var wire struct {
		Meta struct {
			Message string `json:"message"`
			Code    int    `json:"code"`
		} `json:"meta"`
		Data []ShippingData `json:"data"`
	}
	if err := json.Unmarshal(payload, &wire); err != nil {
		return ShippingQuote{}, fmt.Errorf("invalid rajaongkir response: %w", err)
	}
	if wire.Meta.Code == 0 || wire.Meta.Message == "" {
		return ShippingQuote{}, errors.New("invalid rajaongkir response metadata")
	}
	return ShippingQuote{Message: wire.Meta.Message, Status: wire.Meta.Code, Data: wire.Data}, nil
}
