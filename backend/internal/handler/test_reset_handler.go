//go:build e2e

package handler

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"translator-checkin/internal/model"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// E2E seed credentials. Documented here as the single source of truth — the
// Playwright fixtures reference these constants in TypeScript.
const (
	E2ESeedPassword       = "Test1234!"
	E2EAdminEmail         = "admin@admin.com"
	E2ETranslatorActive   = "alice@translator.com"
	E2ETranslatorDisabled = "bob@translator.com"
)

// RegisterTestResetRoutes registers POST /api/test/reset when built with
// `-tags e2e`. Three layers of protection prevent this from running in
// production:
//
//  1. Build tag: this file only compiles with `-tags e2e`. The default
//     production build calls the stub in test_reset_stub.go which is a no-op.
//  2. Env flag: ENABLE_TEST_RESET must be exactly "true".
//  3. GIN_MODE: must not be "release". The endpoint refuses to register if
//     either condition fails.
//
// If both env checks pass, the endpoint truncates all tables, wipes the
// upload directory, then seeds a deterministic dataset for E2E tests.
func RegisterTestResetRoutes(r *gin.Engine, db *gorm.DB, uploadDir string) {
	if os.Getenv("ENABLE_TEST_RESET") != "true" {
		log.Println("[test-reset] ENABLE_TEST_RESET != true, endpoint disabled")
		return
	}
	if os.Getenv("GIN_MODE") == "release" {
		log.Println("[test-reset] refusing to enable in GIN_MODE=release")
		return
	}

	log.Println("[test-reset] WARNING: /api/test/reset is enabled — this build is for E2E testing only")

	r.POST("/api/test/reset", func(c *gin.Context) {
		if err := resetAndSeed(db, uploadDir); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"status":     "ok",
			"adminEmail": E2EAdminEmail,
			"password":   E2ESeedPassword,
			"activeTranslator":   E2ETranslatorActive,
			"disabledTranslator": E2ETranslatorDisabled,
		})
	})
}

// resetAndSeed truncates every business table (FK-safe order) then seeds a
// deterministic dataset. Exported through the package only for the build-tag
// test file in test_reset_handler_test.go.
func resetAndSeed(db *gorm.DB, uploadDir string) error {
	// Truncate in FK-safe order. CASCADE handles edge cases where new tables
	// reference these in the future. RESTART IDENTITY resets auto-increment
	// so seeded rows have stable, predictable IDs.
	tables := []string{
		"diagnosis_photos",
		"schedule_patients",
		"schedules",
		"checkins",
		"export_schedules",
		"audit_logs",
		"patients",
		"users",
	}
	for _, t := range tables {
		if err := db.Exec("TRUNCATE TABLE " + t + " RESTART IDENTITY CASCADE").Error; err != nil {
			return err
		}
	}

	// Wipe upload directory CONTENTS so old test photos don't leak between
	// runs. We deliberately do NOT remove the directory itself — in docker
	// it is a volume mountpoint, and unlinking a mountpoint fails with
	// EBUSY ("device or resource busy"). Removing the children is enough.
	if uploadDir != "" {
		entries, err := os.ReadDir(uploadDir)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		for _, e := range entries {
			if err := os.RemoveAll(filepath.Join(uploadDir, e.Name())); err != nil {
				return err
			}
		}
		// Make sure it exists in case the directory itself was never created
		// (e.g. fresh container where UPLOAD_DIR points somewhere new).
		if err := os.MkdirAll(uploadDir, 0755); err != nil {
			return err
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(E2ESeedPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	hashStr := string(hash)

	// Users: 1 admin (mustChangePW=false so login lands straight on dashboard),
	// 1 active translator, 1 disabled translator.
	users := []model.User{
		{
			Email: E2EAdminEmail, PasswordHash: hashStr, Name: "E2E Admin",
			Role: "admin", Status: "active", MustChangePW: false,
		},
		{
			Email: E2ETranslatorActive, PasswordHash: hashStr, Name: "Alice (active)",
			Phone: "0900-000-001", Role: "translator", Status: "active", MustChangePW: false,
		},
		{
			Email: E2ETranslatorDisabled, PasswordHash: hashStr, Name: "Bob (disabled)",
			Phone: "0900-000-002", Role: "translator", Status: "disabled", MustChangePW: false,
		},
	}
	if err := db.Create(&users).Error; err != nil {
		return err
	}

	// Patients: one of each ID type so the patient selector / display covers
	// the full enum surface during E2E.
	patients := []model.Patient{
		{Name: "Patient Passport", Phone: "0911-000-001", IDType: "passport", IDNumber: "A123456"},
		{Name: "Patient HN", Phone: "0911-000-002", IDType: "hn", IDNumber: "HN001"},
		{Name: "Patient Unid", Phone: "0911-000-003", IDType: "unid", IDNumber: "UN-XYZ"},
	}
	if err := db.Create(&patients).Error; err != nil {
		return err
	}

	// One historical schedule on yesterday's date for Alice, with two
	// patients: the first completed (and has a fake photo URL), the second
	// still pending. This gives diagnosis-results / patient-history pages
	// something to render without each test having to set it up.
	yesterday := time.Now().AddDate(0, 0, -1)
	sched := model.Schedule{
		TranslatorID: users[1].ID, // alice
		Date:         yesterday,
		StartTime:    "09:00",
		EndTime:      "12:00",
		Location:     "E2E Clinic, Bangkok",
		Note:         "Seeded historical schedule",
	}
	if err := db.Create(&sched).Error; err != nil {
		return err
	}

	sps := []model.SchedulePatient{
		{
			ScheduleID: sched.ID, PatientID: patients[0].ID,
			StartTime: "09:00", EndTime: "10:00", OrderIdx: 0,
			Status: model.SchedulePatientStatusCompleted,
		},
		{
			ScheduleID: sched.ID, PatientID: patients[1].ID,
			StartTime: "10:00", EndTime: "11:00", OrderIdx: 1,
			Status: model.SchedulePatientStatusPending,
		},
	}
	if err := db.Create(&sps).Error; err != nil {
		return err
	}

	// Fake photo so the patient-history page renders the image gallery.
	// The file may not exist on disk — UI tests that need a real image
	// should upload through the actual flow instead of relying on the seed.
	photo := model.DiagnosisPhoto{
		SchedulePatientID: sps[0].ID,
		PhotoURL:          filepath.Join("/uploads", "e2e-seed.jpg"),
		UploadedAt:        yesterday,
	}
	if err := db.Create(&photo).Error; err != nil {
		return err
	}

	return nil
}
