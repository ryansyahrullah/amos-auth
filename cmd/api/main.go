package main

import (
	"amk-api-go/configs"
	"amk-api-go/internal/core/audit"
	"amk-api-go/internal/core/platform"
	"amk-api-go/internal/features/auth"
	"amk-api-go/internal/features/user"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env file
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Connect to Database
	configs.ConnectDB()

	// Auto Migrate
	err := configs.DB.AutoMigrate(
		&user.Role{},
		&user.Permission{},
		&user.User{},
		&user.RefreshToken{},
		&auth.PasswordReset{},
		&audit.AuditLog{}, // Audit Logs
	)
	if err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	// --- SEEDING DATA ---
	fmt.Println("Running Data Seeding...")
	user.SeedRoles(configs.DB)
	auth.SeedDefaultAdmin()
	user.SeedPermissions(configs.DB)
	// user.SeedUser(configs.DB) // Disabled as per user request (Super Admin only)
	fmt.Println("Data Seeding Completed.")
	// --------------------
	userRepo := user.NewRepository(configs.DB)
	auditRepo := audit.NewRepository(configs.DB)
	authRepo := auth.NewRepository(configs.DB)

	auditService := audit.NewService(auditRepo)

	authHandler := auth.NewHandler(authRepo, auditService)
	userHandler := user.NewHandler(userRepo, nil)

	// Start Audit Log Cleanup Worker
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			// Automated Audit Log Cleanup (Monthly on the 1st)
			if time.Now().Day() == 1 {
				// Keep logs for 30 days
				if err := auditRepo.DeleteOldLogs(30 * 24 * time.Hour); err != nil {
					fmt.Printf("Error cleaning up old audit logs: %v\n", err)
				} else {
					fmt.Println("[INFO] Audit logs cleanup executed (monthly)")
				}
			}
		}
	}()

	// Initialize Router
	r := gin.Default()
	r.SetFuncMap(template.FuncMap{
		"formatRupiah": func(v interface{}) string {
			var amount float64
			switch val := v.(type) {
			case float64:
				amount = val
			case int:
				amount = float64(val)
			default:
				return fmt.Sprintf("%v", v)
			}
			s := fmt.Sprintf("%.0f", amount)
			n := len(s)
			if n <= 3 {
				return s
			}
			var result []byte
			for i, c := range s {
				if i > 0 && (n-i)%3 == 0 {
					result = append(result, '.')
				}
				result = append(result, byte(c))
			}
			return string(result)
		},
		"formatQty": func(v interface{}) string {
			var amount float64
			switch val := v.(type) {
			case float64:
				amount = val
			case int:
				amount = float64(val)
			default:
				return fmt.Sprintf("%v", v)
			}
			return strconv.FormatFloat(amount, 'f', -1, 64)
		},
	})
	r.LoadHTMLGlob("internal/core/templates/*")
	// r.Static("/assets", "./asset") // Removed asset static serve as asset folder is gone

	// --- CORS CONFIGURATION ---
	config := cors.DefaultConfig()
	config.AllowOrigins = []string{
		"http://localhost:5173",
		"http://127.0.0.1:5173",
		"https://keyareya.com",
		"https://www.keyareya.com",
	}
	config.AllowMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	config.AllowCredentials = true
	r.Use(cors.New(config))

	// Security Headers
	r.Use(platform.SecurityHeadersMiddleware())

	// Health Check
	r.GET("/sukablyat", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"service": "amos-auth",
			"message": "Health check passed",
		})
	})

	// Public Fix Permissions (Temporary for Dev)
	r.POST("/fix-permissions", platform.RateLimitMiddleware(1, 5), userHandler.FixPermissions)

	// Ping Endpoint
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"message": "pong",
		})
	})

	// Auth Routes
	authRoutes := r.Group("/auth")
	{
		authRoutes.POST("/login", platform.RateLimitMiddleware(1, 5), authHandler.Login)
		authRoutes.POST("/logout", platform.RateLimitMiddleware(1, 5), authHandler.Logout)
		authRoutes.POST("/forgot-password", platform.RateLimitMiddleware(0.05, 3), authHandler.ForgotPassword)
		authRoutes.POST("/verify-otp", platform.RateLimitMiddleware(1, 5), authHandler.VerifyOTP)
		authRoutes.POST("/reset-password", platform.RateLimitMiddleware(1, 5), authHandler.ResetPassword)
		authRoutes.POST("/refresh-token", platform.RateLimitMiddleware(1, 5), authHandler.RefreshToken)
	}

	// User Routes
	userRoutes := r.Group("/users")
	userRoutes.Use(platform.AuthMiddleware())
	{
		// User Management
		userRoutes.GET("/me", userHandler.GetMe)
		userRoutes.POST("", platform.PermissionMiddleware("user.create"), userHandler.CreateUser)
		userRoutes.GET("", platform.PermissionMiddleware("user.read"), userHandler.GetAllUsers)
		userRoutes.PUT("/:id", platform.PermissionMiddleware("user.update"), userHandler.UpdateUser)
		userRoutes.DELETE("/:id", platform.PermissionMiddleware("user.delete"), userHandler.DeleteUser)

		// Permission Management
		permissions := userRoutes.Group("/permissions")
		permissions.Use(platform.PermissionMiddleware("permission.manage"))
		{
			permissions.POST("", userHandler.CreatePermission)
			permissions.GET("", userHandler.GetAllPermissions)
			permissions.PUT("/:id", userHandler.UpdatePermission)
			permissions.DELETE("/:id", userHandler.DeletePermission)
			permissions.POST("/assign", userHandler.AssignPermission)
			permissions.POST("/revoke", userHandler.RevokePermission)
		}

		// Role Management
		roles := userRoutes.Group("/roles")
		{
			roles.POST("", platform.PermissionMiddleware("role.create"), userHandler.CreateRole)
			roles.GET("", userHandler.GetAllRoles)
			roles.PUT("/:id", platform.PermissionMiddleware("role.update"), userHandler.UpdateRole)
			roles.DELETE("/:id", platform.PermissionMiddleware("role.delete"), userHandler.DeleteRole)
		}
	}

	// HCGS Proxy
	hcgsTarget := os.Getenv("HCGS_SERVICE_URL")
	if hcgsTarget == "" {
		hcgsTarget = "http://localhost:8081"
	}
	hcgsURL, err := url.Parse(hcgsTarget)
	if err != nil {
		log.Fatalf("Invalid HCGS_SERVICE_URL: %v", err)
	}

	proxy := httputil.NewSingleHostReverseProxy(hcgsURL)

	// Custom Director to ensure headers are passed correctly
	originalDirector := proxy.Director
	proxy.Director = func(req *http.Request) {
		originalDirector(req)
		// Ensure Host header matches target
		req.Host = hcgsURL.Host
		// Preserve Authorization header (already inherited from original request)
		// Preserve cookies for token-based auth
		// The original request headers including Authorization are automatically copied
	}

	proxy.ModifyResponse = func(resp *http.Response) error {
		// Strip CORS headers from the backend response to avoid duplication
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Allow-Methods")
		resp.Header.Del("Access-Control-Allow-Headers")
		resp.Header.Del("Access-Control-Allow-Credentials")
		return nil
	}

	r.Any("/hcgs/*path", func(c *gin.Context) {
		proxy.ServeHTTP(c.Writer, c.Request)
	})

	// Start Server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}
