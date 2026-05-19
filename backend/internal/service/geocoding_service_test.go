package service

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGeocodingService_ReverseGeocode_Success(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/reverse", r.URL.Path)
		assert.Contains(t, r.URL.RawQuery, "lat=25.000000")
		assert.NotEmpty(t, r.Header.Get("User-Agent"), "Nominatim requires User-Agent")
		fmt.Fprintln(w, `{"display_name":"Taipei, Taiwan"}`)
	}))
	defer srv.Close()

	svc := NewGeocodingService()
	svc.SetBaseURL(srv.URL)

	got, err := svc.ReverseGeocode(context.Background(), 25.0, 121.5)
	require.NoError(t, err)
	assert.Equal(t, "Taipei, Taiwan", got)
}

func TestGeocodingService_ReverseGeocode_Non200(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rate limit", http.StatusTooManyRequests)
	}))
	defer srv.Close()

	svc := NewGeocodingService()
	svc.SetBaseURL(srv.URL)

	_, err := svc.ReverseGeocode(context.Background(), 25.0, 121.5)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "status 429")
}

func TestGeocodingService_ReverseGeocode_BadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, `not json`)
	}))
	defer srv.Close()

	svc := NewGeocodingService()
	svc.SetBaseURL(srv.URL)

	_, err := svc.ReverseGeocode(context.Background(), 25.0, 121.5)
	require.Error(t, err)
}
