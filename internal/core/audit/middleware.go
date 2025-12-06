package audit

import (
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

func AuditMiddleware(service *Service) gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()

		// Process request
		c.Next()

		endTime := time.Now()
		latency := endTime.Sub(startTime)

		// Collect data
		method := c.Request.Method
		path := c.Request.URL.Path
		statusCode := c.Writer.Status()
		clientIP := c.ClientIP()
		userAgent := c.Request.UserAgent()

		// Get User ID if available (set by AuthMiddleware)
		var userID *uint
		if val, exists := c.Get("userID"); exists {
			if uid, ok := val.(uint); ok {
				userID = &uid
			}
		}

		// Construct Action and Details
		action := fmt.Sprintf("%s %s", method, path)
		details := fmt.Sprintf("Status: %d, Latency: %v", statusCode, latency)

		// Log to database (async to not block response?)
		// For now, sync is safer to ensure log is written.
		// We can make it a goroutine if performance becomes an issue, but need to handle error logging.
		go func() {
			_ = service.LogActivity(userID, action, details, clientIP, userAgent)
		}()
	}
}
