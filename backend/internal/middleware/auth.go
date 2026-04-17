package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"translator-checkin/internal/config"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT claims payload.
type Claims struct {
	UserID       uint   `json:"user_id"`
	Role         string `json:"role"`
	MustChangePW bool   `json:"must_change_pw"`
	jwt.RegisteredClaims
}

// GenerateToken creates a signed JWT for the given user.
func GenerateToken(userID uint, role string, mustChangePW bool) (string, error) {
	cfg := config.AppConfig
	expiryDuration := time.Duration(cfg.JWTExpiryHrs) * time.Hour

	claims := &Claims{
		UserID:       userID,
		Role:         role,
		MustChangePW: mustChangePW,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(expiryDuration)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(cfg.JWTSecret))
}

// ParseToken validates a JWT string and extracts user ID, role, and the
// "must change password" flag.
func ParseToken(tokenString string) (uint, string, bool, error) {
	cfg := config.AppConfig

	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(cfg.JWTSecret), nil
	})
	if err != nil {
		return 0, "", false, err
	}
	if !token.Valid {
		return 0, "", false, errors.New("invalid token")
	}

	return claims.UserID, claims.Role, claims.MustChangePW, nil
}

// JWTAuth is a Gin middleware that validates the Bearer token and sets
// "userID" and "userRole" in the request context.
func JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must be Bearer {token}"})
			c.Abort()
			return
		}

		userID, role, mustChangePW, err := ParseToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		c.Set("userID", userID)
		c.Set("userRole", role)
		c.Set("mustChangePW", mustChangePW)
		c.Next()
	}
}

// RequirePasswordChanged blocks any request whose token still carries the
// must_change_pw flag. Apply this to every protected route group except
// the change-password endpoint itself.
func RequirePasswordChanged() gin.HandlerFunc {
	return func(c *gin.Context) {
		v, exists := c.Get("mustChangePW")
		if !exists {
			c.Next()
			return
		}
		if must, ok := v.(bool); ok && must {
			c.JSON(http.StatusForbidden, gin.H{
				"code":  "PASSWORD_CHANGE_REQUIRED",
				"error": "password change required",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// RoleRequired is a Gin middleware that checks if the user's role is in the allowed list.
func RoleRequired(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("userRole")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found in context"})
			c.Abort()
			return
		}

		roleStr, ok := userRole.(string)
		if !ok {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid role type in context"})
			c.Abort()
			return
		}

		for _, r := range roles {
			if roleStr == r {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
		c.Abort()
	}
}
