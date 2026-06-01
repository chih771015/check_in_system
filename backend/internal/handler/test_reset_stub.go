//go:build !e2e

package handler

import (
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// RegisterTestResetRoutes is a no-op in production builds.
//
// The real implementation lives in test_reset_handler.go and is gated by the
// `e2e` build tag. Production binaries (built without `-tags e2e`) do not
// include the reset endpoint at all — the code that wipes the database
// physically does not exist in the compiled binary.
//
// This file makes the call site in main.go compile unconditionally so the
// route registration stays simple.
func RegisterTestResetRoutes(_ *gin.Engine, _ *gorm.DB, _ string) {
	// intentionally empty
}
