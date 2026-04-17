package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
)

// GeocodingService resolves latitude/longitude into human-readable addresses
// via the free OpenStreetMap Nominatim endpoint.
type GeocodingService struct {
	client *http.Client
}

// NewGeocodingService creates a new GeocodingService.
// The HTTP client is wrapped with otelhttp so every Nominatim lookup becomes
// a client span in Jaeger.
func NewGeocodingService() *GeocodingService {
	return &GeocodingService{
		client: &http.Client{
			Timeout:   5 * time.Second,
			Transport: otelhttp.NewTransport(http.DefaultTransport),
		},
	}
}

// ReverseGeocode returns a display name for the given coordinates. Errors are
// returned as-is so callers can choose to ignore or propagate them.
// Takes a context so the outbound span nests under the caller's trace.
func (g *GeocodingService) ReverseGeocode(ctx context.Context, lat, lng float64) (string, error) {
	url := fmt.Sprintf(
		"https://nominatim.openstreetmap.org/reverse?format=json&lat=%f&lon=%f&accept-language=zh-TW",
		lat, lng)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	// Nominatim requires a distinctive User-Agent per their usage policy.
	req.Header.Set("User-Agent", "translator-checkin/1.0")
	resp, err := g.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("nominatim returned status %d", resp.StatusCode)
	}
	var payload struct {
		DisplayName string `json:"display_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	return payload.DisplayName, nil
}
