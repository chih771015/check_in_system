package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"translator-checkin/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestConfig() {
	if config.AppConfig == nil {
		config.AppConfig = &config.Config{
			JWTSecret:    "test-secret-key-at-least-32-characters-long-xx",
			JWTExpiryHrs: 24,
		}
	}
}

func TestGenerateAndParseToken_Roundtrip(t *testing.T) {
	setupTestConfig()
	token, err := GenerateToken(42, "admin", true)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	uid, role, mustChange, err := ParseToken(token)
	require.NoError(t, err)
	assert.Equal(t, uint(42), uid)
	assert.Equal(t, "admin", role)
	assert.True(t, mustChange)
}

func TestParseToken_RejectsBogusToken(t *testing.T) {
	setupTestConfig()
	_, _, _, err := ParseToken("not.a.jwt")
	assert.Error(t, err)
}

func TestParseToken_RejectsWrongSecret(t *testing.T) {
	setupTestConfig()
	// Sign with one secret
	token, err := GenerateToken(1, "admin", false)
	require.NoError(t, err)

	// Swap secret and re-parse should fail
	prev := config.AppConfig.JWTSecret
	config.AppConfig.JWTSecret = "different-secret-also-32-chars-long-xxxxxx"
	defer func() { config.AppConfig.JWTSecret = prev }()

	_, _, _, err = ParseToken(token)
	assert.Error(t, err)
}

// runMiddleware helper: build a tiny gin engine, mount middleware, hit it.
func runMiddleware(t *testing.T, mw gin.HandlerFunc, header string, beforeNext gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if beforeNext != nil {
		r.Use(beforeNext)
	}
	r.Use(mw)
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest("GET", "/ping", nil)
	if header != "" {
		req.Header.Set("Authorization", header)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestJWTAuth_MissingHeader(t *testing.T) {
	setupTestConfig()
	w := runMiddleware(t, JWTAuth(), "", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_BadFormat(t *testing.T) {
	setupTestConfig()
	w := runMiddleware(t, JWTAuth(), "Token abc", nil)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestJWTAuth_ValidTokenPasses(t *testing.T) {
	setupTestConfig()
	token, _ := GenerateToken(1, "admin", false)
	w := runMiddleware(t, JWTAuth(), "Bearer "+token, nil)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRequirePasswordChanged_BlocksWhenFlagged(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("mustChangePW", true)
		c.Next()
	})
	r.Use(RequirePasswordChanged())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
	assert.Contains(t, w.Body.String(), "PASSWORD_CHANGE_REQUIRED")
}

func TestRequirePasswordChanged_AllowsWhenFalse(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("mustChangePW", false)
		c.Next()
	})
	r.Use(RequirePasswordChanged())
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRoleRequired_AllowsMatchingRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userRole", "admin")
		c.Next()
	})
	r.Use(RoleRequired("admin"))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRoleRequired_RejectsOtherRole(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("userRole", "translator")
		c.Next()
	})
	r.Use(RoleRequired("admin"))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestRoleRequired_MissingRoleInContext(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(RoleRequired("admin"))
	r.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	req := httptest.NewRequest("GET", "/ping", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}
