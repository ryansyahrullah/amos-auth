package platform

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"amk-api-go/configs"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		var tokenString string

		// 1. Check Cookie
		if cookie, err := c.Cookie("token"); err == nil {
			tokenString = cookie
		}

		// 2. Fallback to Header
		if tokenString == "" {
			authHeader := c.GetHeader("Authorization")
			if authHeader != "" {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}

		// 3. Fallback to Query Param
		if tokenString == "" {
			tokenString = c.Query("token")
		}

		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization required"})
			c.Abort()
			return
		}
		claims, err := ValidateToken(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)

		// Fetch user to get username (avoids import cycle in handlers)
		var u struct {
			Name string
		}
		if err := configs.DB.Table("users").Select("name").Where("id = ?", claims.UserID).First(&u).Error; err == nil {
			c.Set("username", u.Name)
		}

		c.Next()
	}
}

func RoleMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		roleStr := userRole.(string)
		for _, role := range allowedRoles {
			if role == roleStr {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden"})
		c.Abort()
	}
}

// PermissionMiddleware checks if the user's role has the required permission.
func PermissionMiddleware(requiredPermission string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
			c.Abort()
			return
		}

		// Super Admin Bypass
		// Super Admin Bypass
		userRole, _ := c.Get("role")
		roleStr, ok := userRole.(string)
		if ok {
			normalizedRole := strings.ToLower(strings.TrimSpace(roleStr))
			if normalizedRole == "super_admin" || normalizedRole == "super admin" || normalizedRole == "superadmin" {
				c.Next()
				return
			}
		}

		// Check if user has permission via their role
		// We need to join User -> Role -> Permissions
		var count int64
		err := configs.DB.Table("users").
			Joins("JOIN roles ON users.role_id = roles.id").
			Joins("JOIN role_permissions ON roles.id = role_permissions.role_id").
			Joins("JOIN permissions ON role_permissions.permission_id = permissions.id").
			Where("users.id = ? AND permissions.name = ?", userID, requiredPermission).
			Count(&count).Error

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check permissions"})
			c.Abort()
			return
		}

		if count > 0 {
			c.Next()
			return
		}

		c.JSON(http.StatusForbidden, gin.H{"error": "Forbidden: Missing permission " + requiredPermission})
		c.Abort()
	}
}

// RateLimitMiddleware limits requests based on IP address.
// r: rate (events per second), b: burst size
func RateLimitMiddleware(r float64, b int) gin.HandlerFunc {
	// Simple in-memory map for rate limiters.
	// In production, use Redis or similar for distributed systems.
	type client struct {
		limiter  *rate.Limiter
		lastSeen time.Time
	}
	var (
		mu      sync.Mutex
		clients = make(map[string]*client)
	)

	// Cleanup background routine
	go func() {
		for {
			time.Sleep(time.Minute)
			mu.Lock()
			for ip, client := range clients {
				if time.Since(client.lastSeen) > 3*time.Minute {
					delete(clients, ip)
				}
			}
			mu.Unlock()
		}
	}()

	return func(c *gin.Context) {
		ip := c.ClientIP()
		mu.Lock()
		if _, found := clients[ip]; !found {
			clients[ip] = &client{limiter: rate.NewLimiter(rate.Limit(r), b)}
		}
		clients[ip].lastSeen = time.Now()
		if !clients[ip].limiter.Allow() {
			mu.Unlock()
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "Too many requests"})
			c.Abort()
			return
		}
		mu.Unlock()
		c.Next()
	}
}

// SecurityHeadersMiddleware adds security-related headers to responses
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Prevent MIME type sniffing
		c.Header("X-Content-Type-Options", "nosniff")

		// Prevent clickjacking
		c.Header("X-Frame-Options", "DENY")

		// Enable XSS protection (browser filter)
		c.Header("X-XSS-Protection", "1; mode=block")

		// Strict Transport Security (HSTS) - 1 year
		// Only effective on HTTPS.
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")

		// Content Security Policy (CSP)
		// Restrict sources for scripts, styles, images, etc.
		// Adjust 'self' and other domains as needed.
		// Added 'unsafe-inline' and 'unsafe-eval' for now as Vue/Vite might need them in dev.
		// Added cloudflare.com for Turnstile (wildcard for all subdomains).
		csp := "default-src 'self'; " +
			"script-src 'self' 'unsafe-inline' 'unsafe-eval' https://*.cloudflare.com https://challenges.cloudflare.com; " +
			"style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: blob: https://*.cloudflare.com; " +
			"frame-src https://*.cloudflare.com https://challenges.cloudflare.com; " +
			"connect-src 'self' https://*.cloudflare.com https://challenges.cloudflare.com; " +
			"font-src 'self' data:;"
		c.Header("Content-Security-Policy", csp)

		// Referrer Policy
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")

		c.Next()
	}
}
