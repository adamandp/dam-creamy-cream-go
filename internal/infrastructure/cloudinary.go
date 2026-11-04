package infrastructure

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type Cloudinary struct {
	CloudName string
	APIKey    string
	APISecret string
	client    *http.Client
}

func NewCloudinary(cloudName, apiKey, apiSecret string) *Cloudinary {
	return &Cloudinary{CloudName: cloudName, APIKey: apiKey, APISecret: apiSecret, client: &http.Client{Timeout: 25 * time.Second}}
}
func (c *Cloudinary) ready() error {
	if c.CloudName == "" || c.APIKey == "" || c.APISecret == "" {
		return errors.New("cloudinary credentials are missing")
	}
	return nil
}
func (c *Cloudinary) signature(parameters string) string {
	b := sha1.Sum([]byte(parameters + c.APISecret))
	return hex.EncodeToString(b[:])
}
func (c *Cloudinary) Upload(ctx context.Context, publicID, filename string, data []byte) (string, error) {
	if err := c.ready(); err != nil {
		return "", err
	}
	now := strconv.FormatInt(time.Now().Unix(), 10)
	body := new(bytes.Buffer)
	w := multipart.NewWriter(body)
	file, err := w.CreateFormFile("file", filename)
	if err != nil {
		return "", err
	}
	if _, err := file.Write(data); err != nil {
		return "", err
	}
	fields := map[string]string{"public_id": publicID, "timestamp": now, "api_key": c.APIKey, "signature": c.signature("public_id=" + publicID + "&timestamp=" + now)}
	for key, value := range fields {
		if err := w.WriteField(key, value); err != nil {
			return "", err
		}
	}
	if err := w.Close(); err != nil {
		return "", err
	}
	endpoint := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/upload", url.PathEscape(c.CloudName))
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", w.FormDataContentType())
	res, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return "", err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return "", fmt.Errorf("cloudinary upload: status=%d response=%s", res.StatusCode, string(raw))
	}
	var response struct {
		SecureURL string `json:"secure_url"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return "", err
	}
	if response.SecureURL == "" {
		return "", errors.New("cloudinary returned no secure_url")
	}
	return response.SecureURL, nil
}
func (c *Cloudinary) Delete(ctx context.Context, publicID string) error {
	if err := c.ready(); err != nil {
		return err
	}
	now := strconv.FormatInt(time.Now().Unix(), 10)
	values := url.Values{}
	values.Set("public_id", publicID)
	values.Set("timestamp", now)
	values.Set("api_key", c.APIKey)
	values.Set("signature", c.signature("public_id="+publicID+"&timestamp="+now))
	endpoint := fmt.Sprintf("https://api.cloudinary.com/v1_1/%s/image/destroy", url.PathEscape(c.CloudName))
	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	res, err := c.client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if err != nil {
		return err
	}
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("cloudinary destroy: status=%d response=%s", res.StatusCode, string(raw))
	}
	var response struct {
		Result string `json:"result"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		return err
	}
	if response.Result != "ok" && response.Result != "not found" {
		return fmt.Errorf("cloudinary destroy result: %s", response.Result)
	}
	return nil
}
