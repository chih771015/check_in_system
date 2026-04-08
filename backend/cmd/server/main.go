package main

import (
	"log"
	"os"
	"time"

	"translator-checkin/internal/config"
	"translator-checkin/internal/handler"
	"translator-checkin/internal/middleware"
	"translator-checkin/internal/model"
	"translator-checkin/internal/repository"
	"translator-checkin/internal/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	cron "github.com/robfig/cron/v3"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Connect to PostgreSQL
	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	log.Println("Database connected successfully")

	// Auto-migrate all models
	if err := db.AutoMigrate(&model.User{}, &model.Schedule{}, &model.Checkin{}, &model.ExportSchedule{}); err != nil {
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

	// Initialize services
	authService := service.NewAuthService(userRepo)
	translatorService := service.NewTranslatorService(userRepo)
	scheduleService := service.NewScheduleService(scheduleRepo, checkinRepo, userRepo)
	checkinService := service.NewCheckinService(checkinRepo, scheduleRepo, userRepo)

	// Initialize handlers
	authHandler := handler.NewAuthHandler(authService)
	translatorHandler := handler.NewTranslatorHandler(translatorService)
	scheduleHandler := handler.NewScheduleHandler(scheduleService)
	checkinHandler := handler.NewCheckinHandler(checkinService)
	exportScheduleHandler := handler.NewExportScheduleHandler(exportScheduleRepo)

	// Setup Gin router
	r := gin.Default()

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
		admin.Use(middleware.JWTAuth(), middleware.RoleRequired("admin"))
		{
			// Translator management
			admin.GET("/translators", translatorHandler.ListTranslators)
			admin.POST("/translators", translatorHandler.CreateTranslator)
			admin.PUT("/translators/:id", translatorHandler.UpdateTranslator)
			admin.DELETE("/translators/:id", translatorHandler.DisableTranslator)

			// Schedule management
			admin.GET("/schedules", scheduleHandler.AdminListSchedules)
			admin.POST("/schedules", scheduleHandler.AdminCreateSchedule)
			admin.PUT("/schedules/:id", scheduleHandler.AdminUpdateSchedule)
			admin.DELETE("/schedules/:id", scheduleHandler.AdminDeleteSchedule)

			// Checkin records
			admin.GET("/checkins", checkinHandler.AdminListCheckins)
			admin.GET("/export/excel", checkinHandler.AdminExportExcel)
			admin.POST("/export/google-sheet", checkinHandler.AdminExportGoogleSheet)

			// Export schedule
			admin.GET("/export/schedule", exportScheduleHandler.GetExportSchedule)
			admin.POST("/export/schedule", exportScheduleHandler.UpsertExportSchedule)
		}

		// Translator routes
		translatorRoutes := api.Group("")
		translatorRoutes.Use(middleware.JWTAuth(), middleware.RoleRequired("translator"))
		{
			translatorRoutes.GET("/schedules", scheduleHandler.MySchedules)
			translatorRoutes.POST("/checkins", checkinHandler.Checkin)
			translatorRoutes.POST("/checkins/makeup", checkinHandler.MakeupCheckin)
		}
	}

	// Start export cron scheduler
	startExportCron(exportScheduleRepo, checkinService)

	// Start server
	port := cfg.Port
	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

// startExportCron starts a cron job that checks for scheduled exports daily at 08:00.
func startExportCron(repo *repository.ExportScheduleRepository, svc *service.CheckinService) {
	c := cron.New()
	// Run daily at 08:00 to check if today matches any schedule
	c.AddFunc("0 8 * * *", func() {
		today := time.Now()
		schedules, _ := repo.FindAllEnabled()
		for _, es := range schedules {
			if es.DayOfMonth == today.Day() {
				log.Printf("Running scheduled export for admin %d (format: %s)", es.AdminID, es.Format)
				repo.UpdateLastRun(es.ID, today)
				// Actual export execution would happen here
				// For now we just log; full email/drive integration is Phase 4
			}
		}
	})
	c.Start()
	log.Println("Export cron scheduler started")
}

// seedAdmin creates the default admin account if it does not already exist.
func seedAdmin(db *gorm.DB) {
	var count int64
	db.Model(&model.User{}).Where("email = ?", "admin@admin.com").Count(&count)
	if count > 0 {
		log.Println("Admin account already exists, skipping seed")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
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
	log.Println("Admin account seeded: admin@admin.com / admin123")
}
