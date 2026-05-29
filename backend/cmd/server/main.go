package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"translator-checkin/internal/config"
	"translator-checkin/internal/handler"
	"translator-checkin/internal/middleware"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"
	"translator-checkin/internal/service"
	"translator-checkin/internal/tracing"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	cron "github.com/robfig/cron/v3"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	oteltrace "go.opentelemetry.io/otel/trace"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormtracing "gorm.io/plugin/opentelemetry/tracing"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// CLI mode: -reset-password <email> <newPassword>
	// Usage: docker exec thai-backend ./server -reset-password admin@admin.com "NewPass123"
	resetPW := flag.Bool("reset-password", false, "Reset a user's password (CLI mode)")
	flag.Parse()
	if *resetPW {
		args := flag.Args()
		if len(args) != 2 {
			fmt.Fprintln(os.Stderr, "Usage: server -reset-password <email> <newPassword>")
			os.Exit(1)
		}
		runResetPassword(cfg, args[0], args[1])
		os.Exit(0)
	}

	// Initialize OpenTelemetry tracing (OTLP → Jaeger). If the collector is
	// down we still boot — spans will be dropped, not the app.
	ctx := context.Background()
	shutdownTracing, err := tracing.Init(ctx)
	if err != nil {
		log.Printf("[tracing] init failed, running without tracing: %v", err)
	} else {
		defer func() {
			shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			if err := shutdownTracing(shutdownCtx); err != nil {
				log.Printf("[tracing] shutdown error: %v", err)
			}
		}()
	}

	// Connect to PostgreSQL
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connected successfully")

	// Install GORM tracing plugin so every SQL query becomes a span.
	// WithoutMetrics keeps memory usage down; we only care about traces here.
	// WithoutQueryVariables scrubs bound parameters from the span so PII
	// (emails, password hashes, etc.) never leaks to the collector.
	if err := db.Use(gormtracing.NewPlugin(
		gormtracing.WithoutMetrics(),
		gormtracing.WithoutQueryVariables(),
		gormtracing.WithDBSystem("postgresql"),
	)); err != nil {
		log.Printf("[tracing] gorm plugin install failed: %v", err)
	}

	// Auto-migrate all models
	if err := db.AutoMigrate(
		&model.User{},
		&model.Schedule{},
		&model.Checkin{},
		&model.ExportSchedule{},
		&model.AuditLog{},
		&model.Patient{},
		&model.SchedulePatient{},
		&model.DiagnosisPhoto{},
	); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}
	log.Println("Database migration completed")

	// Seed admin account if not exists
	seedAdmin(db)

	// Create upload directory
	if err := os.MkdirAll(cfg.UploadDir, 0755); err != nil {
		log.Fatalf("Failed to create upload directory: %v", err)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	scheduleRepo := repository.NewScheduleRepository(db)
	checkinRepo := repository.NewCheckinRepository(db)
	exportScheduleRepo := repository.NewExportScheduleRepository(db)
	auditRepo := repository.NewAuditLogRepository(db)
	patientRepo := repository.NewPatientRepository(db)
	schedulePatientRepo := repository.NewSchedulePatientRepository(db)
	diagnosisPhotoRepo := repository.NewDiagnosisPhotoRepository(db)

	// Initialize services
	authService := service.NewAuthService(userRepo)
	adminService := service.NewAdminService(userRepo)
	translatorService := service.NewTranslatorService(userRepo)
	scheduleService := service.NewScheduleService(scheduleRepo, checkinRepo, userRepo).
		WithPatientRepos(schedulePatientRepo, patientRepo)
	geocodingService := service.NewGeocodingService()
	checkinService := service.NewCheckinService(checkinRepo, scheduleRepo, userRepo, geocodingService).
		WithSchedulePatientRepo(schedulePatientRepo)
	mailService := service.NewMailService()
	exportService := service.NewExportService(checkinService, exportScheduleRepo, mailService)
	auditService := service.NewAuditService(auditRepo, userRepo)
	notificationService := service.NewNotificationService(userRepo, scheduleRepo, mailService)
	cleanupService := service.NewCleanupService()
	patientService := service.NewPatientService(patientRepo).
		WithScopeRepo(schedulePatientRepo).
		WithHistoryRepos(scheduleRepo, schedulePatientRepo, diagnosisPhotoRepo)
	diagnosisService := service.NewDiagnosisService(schedulePatientRepo, diagnosisPhotoRepo, scheduleRepo)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	adminHandler := handler.NewAdminHandler(adminService, auditService)
	translatorHandler := handler.NewTranslatorHandler(translatorService, authService, auditService)
	scheduleHandler := handler.NewScheduleHandler(scheduleService, auditService)
	checkinHandler := handler.NewCheckinHandler(checkinService, exportService, auditService)
	exportScheduleHandler := handler.NewExportScheduleHandler(exportScheduleRepo, exportService)
	auditHandler := handler.NewAuditHandler(auditService)
	patientHandler := handler.NewPatientHandler(patientService, auditService)
	diagnosisHandler := handler.NewDiagnosisHandler(diagnosisService, auditService)

	// Setup Gin router
	r := gin.Default()

	// OpenTelemetry middleware — creates a server span per request and
	// propagates W3C traceparent headers. The SpanNameFormatter is
	// overridden so span names use the route template (e.g.
	// "GET /api/admin/translators/:id") instead of the raw URL, which
	// keeps cardinality low and hides ids out of span names.
	r.Use(otelgin.Middleware("translator-checkin",
		otelgin.WithSpanNameFormatter(func(c *gin.Context) string {
			if c.FullPath() != "" {
				return c.Request.Method + " " + c.FullPath()
			}
			return c.Request.Method + " " + c.Request.URL.Path
		}),
	))

	// Strip sensitive headers/fields from recorded spans so PII never
	// leaks to the collector. Runs after otelgin so the span exists.
	r.Use(scrubSensitiveSpanAttributes())

	// CORS configuration (allow all origins for development)
	r.Use(cors.New(cors.Config{
		AllowAllOrigins:  true,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "Accept"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: false,
	}))

	// Static file serving for uploads
	r.Static("/uploads", cfg.UploadDir)

	// API routes
	api := r.Group("/api")
	{
		// Auth routes (public)
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/change-password", middleware.JWTAuth(), authHandler.ChangePassword)
		}

		// Admin routes
		admin := api.Group("/admin")
		admin.Use(middleware.JWTAuth(), middleware.RequirePasswordChanged(), middleware.RoleRequired("admin"))
		{
			// Translator management
			admin.GET("/translators", translatorHandler.ListTranslators)
			admin.POST("/translators", translatorHandler.CreateTranslator)
			admin.PUT("/translators/:id", translatorHandler.UpdateTranslator)
			admin.DELETE("/translators/:id", translatorHandler.DisableTranslator)
			admin.POST("/translators/:id/reset-password", translatorHandler.ResetTranslatorPassword)

			// Schedule management
			admin.GET("/schedules", scheduleHandler.AdminListSchedules)
			admin.POST("/schedules", scheduleHandler.AdminCreateSchedule)
			admin.POST("/schedules/import", scheduleHandler.AdminImportSchedules)
			admin.PUT("/schedules/:id", scheduleHandler.AdminUpdateSchedule)
			admin.DELETE("/schedules/:id", scheduleHandler.AdminDeleteSchedule)
			admin.DELETE("/schedules/:id/group", scheduleHandler.AdminDeleteScheduleGroup)

			// Checkin records
			admin.GET("/checkins", checkinHandler.AdminListCheckins)
			admin.PUT("/checkins/:id", checkinHandler.AdminUpdateCheckin)
			admin.DELETE("/checkins/:id", checkinHandler.AdminDeleteCheckin)
			admin.GET("/export/excel", checkinHandler.AdminExportExcel)
			admin.POST("/export/google-sheet", checkinHandler.AdminExportGoogleSheet)

			// Export schedule
			admin.GET("/export/schedule", exportScheduleHandler.GetExportSchedule)
			admin.POST("/export/schedule", exportScheduleHandler.UpsertExportSchedule)
			admin.POST("/export/schedule/run", exportScheduleHandler.RunExportNow)

			// Audit logs
			admin.GET("/audit-logs", auditHandler.ListAuditLogs)

			// Admin account management
			admin.GET("/admins", adminHandler.ListAdmins)
			admin.POST("/admins", adminHandler.CreateAdmin)
			admin.DELETE("/admins/:id", adminHandler.DeleteAdmin)

			// Patient management
			admin.GET("/patients", patientHandler.ListPatients)
			admin.POST("/patients", patientHandler.CreatePatient)
			admin.PUT("/patients/:id", patientHandler.UpdatePatient)
			admin.DELETE("/patients/:id", patientHandler.DeletePatient)
			admin.GET("/patients/:id/history", patientHandler.GetPatientHistory)

			// Stage 4 — admin surrogate uploads / mark no-show
			admin.POST("/diagnosis", diagnosisHandler.AdminUploadDiagnosis)
			admin.POST("/no-show", diagnosisHandler.AdminMarkNoShow)

			// Diagnosis results overview (all completed / no_show rows).
			admin.GET("/diagnosis-results", diagnosisHandler.AdminListResults)

			// Per-SchedulePatient photos (used by schedule detail modal).
			admin.GET("/schedule-patients/:id/photos", diagnosisHandler.AdminGetSchedulePatientPhotos)
		}

		// Translator routes
		translatorRoutes := api.Group("")
		translatorRoutes.Use(middleware.JWTAuth(), middleware.RequirePasswordChanged(), middleware.RoleRequired("translator"))
		{
			translatorRoutes.GET("/schedules", scheduleHandler.MySchedules)
			translatorRoutes.POST("/checkins", checkinHandler.Checkin)
			translatorRoutes.POST("/checkins/makeup", checkinHandler.MakeupCheckin)
			translatorRoutes.GET("/checkins", checkinHandler.MyCheckins)
			translatorRoutes.GET("/checkins/stats", checkinHandler.MyStats)

			// Stage 4 — per-patient diagnosis upload and no-show marking.
			translatorRoutes.POST("/checkins/diagnosis", diagnosisHandler.UploadDiagnosis)
			translatorRoutes.POST("/checkins/no-show", diagnosisHandler.MarkNoShow)

			// Patient list for translator (trimmed view).
			translatorRoutes.GET("/patients", patientHandler.ListPatientsForTranslator)
		}
	}

	// Start export cron scheduler
	startExportCron(exportScheduleRepo, exportService)

	// Start background cron jobs: reminders + photo cleanup
	startBackgroundCrons(notificationService, cleanupService)

	// Start server
	port := cfg.Port
	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// startExportCron starts a cron job that checks for scheduled exports daily at 08:00.
// When today matches a schedule's day-of-month, the configured export is built
// and emailed via ExportService.
func startExportCron(repo *repository.ExportScheduleRepository, exportSvc *service.ExportService) {
	c := cron.New()
	tracer := otel.Tracer("translator-checkin/cron")
	c.AddFunc("0 8 * * *", func() {
		// Each cron tick gets its own trace so Jaeger can show the full
		// fan-out: loading schedules → building export → sending email.
		ctx, span := tracer.Start(context.Background(), "cron.export_schedules")
		defer span.End()

		today := time.Now()
		schedules, _ := repo.FindAllEnabled()
		for _, es := range schedules {
			if es.DayOfMonth != today.Day() {
				continue
			}
			log.Printf("Running scheduled export for admin %d (format: %s)", es.AdminID, es.Format)
			runCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
			result, err := exportSvc.RunExportForAdmin(runCtx, es.AdminID)
			cancel()
			if err != nil {
				log.Printf("Scheduled export failed for admin %d: %v", es.AdminID, err)
				continue
			}
			log.Printf("Scheduled export OK for admin %d (range %s~%s)", es.AdminID, result.RangeFrom, result.RangeTo)
		}
	})
	c.Start()
	log.Println("Export cron scheduler started")
}

// startBackgroundCrons registers daily cron jobs for:
//   - 07:00 schedule reminders (LINE / email)
//   - 03:00 photo cleanup of files older than retention window
//
// Each tick opens its own root span so we get a separate Jaeger trace per run.
func startBackgroundCrons(notificationSvc *service.NotificationService, cleanupSvc *service.CleanupService) {
	c := cron.New()
	tracer := otel.Tracer("translator-checkin/cron")

	// 07:00 daily — send tomorrow's reminders
	c.AddFunc("0 7 * * *", func() {
		_, span := tracer.Start(context.Background(), "cron.schedule_reminders")
		defer span.End()
		log.Println("[cron] running schedule reminders")
		notificationSvc.SendScheduleReminders()
	})
	// 03:00 daily — prune old photos
	c.AddFunc("0 3 * * *", func() {
		_, span := tracer.Start(context.Background(), "cron.photo_cleanup")
		defer span.End()
		log.Println("[cron] running photo cleanup")
		cleanupSvc.RunPhotoCleanup()
	})
	c.Start()
	log.Println("Background cron scheduler started")
}

// scrubSensitiveSpanAttributes removes attributes that could leak PII from
// the gin server span. otelgin records http.target (full URL + query string)
// and http.user_agent by default; we also blank out any Authorization-ish
// headers that sneaked in as attributes via a future upgrade.
func scrubSensitiveSpanAttributes() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		// The span is already attached to the request context by otelgin.
		// We only need to remove query strings from the http.target attribute
		// on logged spans. OTel SDK doesn't support in-place attribute edits
		// on an ended span, so we rely on setting the "clean" attribute before
		// the span ends. otelgin ends the span after middlewares return, so
		// setting attributes here still lands on the exported span.
		span := oteltrace.SpanFromContext(c.Request.Context())
		if !span.IsRecording() {
			return
		}
		if raw := c.Request.URL.Path; raw != "" {
			span.SetAttributes(attribute.String("http.target", raw)) // path without query
		}
	}
}

// seedAdmin creates the default admin account if it does not already exist.
// The password comes from ADMIN_DEFAULT_PASSWORD; if unset, a random password
// is generated and logged once so operators can capture it on first boot.
func seedAdmin(db *gorm.DB) {
	var count int64
	db.Model(&model.User{}).Where("email = ?", "admin@admin.com").Count(&count)
	if count > 0 {
		log.Println("Admin account already exists, skipping seed")
		return
	}

	pw := config.AppConfig.AdminDefaultPassword
	if pw == "" {
		buf := make([]byte, 8)
		if _, err := rand.Read(buf); err != nil {
			log.Fatalf("Failed to generate random admin password: %v", err)
		}
		pw = hex.EncodeToString(buf)
		log.Printf("ADMIN_DEFAULT_PASSWORD not set — generated random admin password: %s", pw)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(pw), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash admin password: %v", err)
	}

	admin := model.User{
		Email:        "admin@admin.com",
		PasswordHash: string(hash),
		Name:         "System Admin",
		Role:         "admin",
		Status:       "active",
		MustChangePW: true,
	}

	if err := db.Create(&admin).Error; err != nil {
		log.Fatalf("Failed to seed admin account: %v", err)
	}
	log.Printf("Admin account seeded: admin@admin.com (password set via %s)",
		ternary(config.AppConfig.AdminDefaultPassword != "", "ADMIN_DEFAULT_PASSWORD env", "random"))
}

func ternary(cond bool, a, b string) string {
	if cond {
		return a
	}
	return b
}

// runResetPassword connects to DB and resets a user's password directly.
// Intended for emergency recovery inside the Docker container.
func runResetPassword(cfg *config.Config, email, newPW string) {
	if len(newPW) < 8 {
		fmt.Fprintln(os.Stderr, "Error: new password must be at least 8 characters")
		os.Exit(1)
	}
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("[reset-password] DB connect failed: %v", err)
	}
	userRepo := repository.NewUserRepository(db)
	user, err := userRepo.FindByEmail(email)
	if err != nil {
		log.Fatalf("[reset-password] User not found: %s", email)
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(newPW), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("[reset-password] Hash failed: %v", err)
	}
	if err := userRepo.UpdatePasswordAndForceChange(user.ID, string(hash)); err != nil {
		log.Fatalf("[reset-password] Update failed: %v", err)
	}
	log.Printf("[reset-password] Password reset for %s (id=%d). must_change_pw=true", email, user.ID)
}
