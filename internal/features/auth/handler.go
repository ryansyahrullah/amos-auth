package auth

import (
	"amk-api-go/internal/core/audit"
	"amk-api-go/internal/core/platform"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

type Handler struct {
	repo         *Repository
	auditService *audit.Service
}

func NewHandler(repo *Repository, auditService *audit.Service) *Handler {
	return &Handler{repo: repo, auditService: auditService}
}

type LoginRequest struct {
	Identifier   string `json:"identifier" binding:"required"` // NRP or Email
	Password     string `json:"password" binding:"required"`
	CaptchaToken string `json:"captcha_token"` // Optional for now
}

func (h *Handler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ip := c.ClientIP()
	userAgent := c.Request.UserAgent()

	// Verify Captcha
	isMobile := c.GetHeader("X-App-Source") == "mobile"
	if !isMobile {
		if err := platform.VerifyCaptcha(req.CaptchaToken); err != nil {
			fmt.Printf("Captcha verification failed: %v\n", err)
			h.auditService.LogActivity(nil, "LOGIN_FAILED_CAPTCHA", fmt.Sprintf("Identifier: %s, Error: %v", req.Identifier, err), ip, userAgent)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Captcha verification failed. Please try again."})
			return
		}
	}

	user, err := h.repo.FindUserByEmailOrNRP(req.Identifier)
	if err != nil {
		fmt.Printf("Login error for %s: %v\n", req.Identifier, err)
		h.auditService.LogActivity(nil, "LOGIN_FAILED_USER_NOT_FOUND", fmt.Sprintf("Identifier: %s", req.Identifier), ip, userAgent)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Kombinasi Email/NRP dan Password salah"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		h.auditService.LogActivity(&user.ID, "LOGIN_FAILED_PASSWORD", "Invalid password", ip, userAgent)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Kombinasi Email/NRP dan Password salah"})
		return
	}

	// 1. Generate Short-Lived Access Token (15m)
	permissions := make([]string, len(user.Role.Permissions))
	for i, p := range user.Role.Permissions {
		permissions[i] = p.Name
	}

	token, err := platform.GenerateToken(user.ID, user.Role.Name, permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses login"})
		return
	}

	// 2. Generate Long-Lived Refresh Token (Random String)
	refreshToken, err := platform.GenerateRandomToken(64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal membuat session"})
		return
	}

	// 3. Save Refresh Token to DB (Expires in 7 days)
	if err := h.repo.SaveRefreshToken(user.ID, refreshToken, time.Now().Add(7*24*time.Hour)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal menyimpan session"})
		return
	}

	// 4. Set HttpOnly Cookie for Refresh Token
	c.SetSameSite(http.SameSiteNoneMode)
	isSecure := true
	if os.Getenv("APP_ENV") == "development" {
		isSecure = false
		c.SetSameSite(http.SameSiteLaxMode)
	}
	// Cookie: refresh_token, Value: token, MaxAge: 7 days
	c.SetCookie("refresh_token", refreshToken, 3600*24*7, "/", "", isSecure, true)

	// Log Successful Login
	h.auditService.LogActivity(&user.ID, "LOGIN_SUCCESS", "User logged in successfully", ip, userAgent)

	menus := []string{}
	if user.Role.Menus != "" {
		menus = strings.Split(user.Role.Menus, ",")
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login berhasil",
		"token":   token, // Access Token (Short-lived)
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"nrp":   user.NRP,
			"role":  user.Role.Name,
			"menus": menus,
		},
	})
}

func (h *Handler) RefreshToken(c *gin.Context) {
	// Get Refresh Token from Cookie
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil || refreshToken == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session expired, please login again"})
		return
	}

	// Verify in DB
	rt, err := h.repo.FindRefreshToken(refreshToken)
	if err != nil {
		// Invalid or Revoked -> Clear cookie
		c.SetCookie("refresh_token", "", -1, "/", "", false, true)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Session invalid"})
		return
	}

	// Rotation: Revoke old, Issue new
	h.repo.RevokeRefreshToken(refreshToken)

	newRefreshToken, err := platform.GenerateRandomToken(64)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server error"})
		return
	}

	if err := h.repo.SaveRefreshToken(rt.UserID, newRefreshToken, time.Now().Add(7*24*time.Hour)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server error"})
		return
	}

	// Generate new Access Token
	permissions := make([]string, len(rt.User.Role.Permissions))
	for i, p := range rt.User.Role.Permissions {
		permissions[i] = p.Name
	}
	newAccessToken, err := platform.GenerateToken(rt.UserID, rt.User.Role.Name, permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Server error"})
		return
	}

	// Set new Cookie
	c.SetSameSite(http.SameSiteNoneMode)
	isSecure := true
	if os.Getenv("APP_ENV") == "development" {
		isSecure = false
		c.SetSameSite(http.SameSiteLaxMode)
	}
	c.SetCookie("refresh_token", newRefreshToken, 3600*24*7, "/", "", isSecure, true)

	c.JSON(http.StatusOK, gin.H{
		"token": newAccessToken,
	})
}

func (h *Handler) Logout(c *gin.Context) {
	// 1. Get Refresh Token to revoke it
	refreshToken, err := c.Cookie("refresh_token")
	if err == nil && refreshToken != "" {
		h.repo.RevokeRefreshToken(refreshToken)
	}

	// Log Logout
	userID, exists := c.Get("userID")
	if exists {
		uid, ok := userID.(uint)
		if ok {
			ip := c.ClientIP()
			userAgent := c.Request.UserAgent()
			h.auditService.LogActivity(&uid, "LOGOUT", "User logged out", ip, userAgent)
		}
	}

	// Clear the refresh_token cookie
	c.SetSameSite(http.SameSiteNoneMode)
	isSecure := true
	if os.Getenv("APP_ENV") == "development" {
		isSecure = false
		c.SetSameSite(http.SameSiteLaxMode)
	}

	c.SetCookie("refresh_token", "", -1, "/", "", isSecure, true)
	c.SetCookie("token", "", -1, "/", "", isSecure, true) // Clear old legacy cookie if any
	c.JSON(http.StatusOK, gin.H{"message": "Logout berhasil"})
}

type ForgotPasswordRequest struct {
	Email        string `json:"email" binding:"required,email"`
	CaptchaToken string `json:"captcha_token"`
}

func (h *Handler) ForgotPassword(c *gin.Context) {
	var req ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify Captcha
	isMobile := c.GetHeader("X-App-Source") == "mobile"
	if !isMobile {
		if err := platform.VerifyCaptcha(req.CaptchaToken); err != nil {
			fmt.Printf("Captcha verification failed: %v\n", err)
			c.JSON(http.StatusBadRequest, gin.H{"error": "Captcha verification failed. Please try again."})
			return
		}
	}

	req.Email = strings.TrimSpace(req.Email)

	user, err := h.repo.FindUserByEmailOrNRP(req.Email)
	if err != nil {
		// User not found: fake success for security
		c.JSON(http.StatusOK, gin.H{"message": "Jika email terdaftar, OTP akan dikirimkan ke email Anda."})
		return
	}

	// Generate OTP
	otp, err := platform.GenerateOTP()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal memproses permintaan"})
		return
	}

	// Save OTP
	if err := h.repo.SaveOTP(user.Email, otp); err != nil {
		c.JSON(http.StatusTooManyRequests, gin.H{"error": "Terlalu banyak permintaan. Silakan tunggu beberapa saat."})
		return
	}

	// Send Email
	baseURL := os.Getenv("APP_URL")
	if baseURL == "" {
		baseURL = "http://localhost:9090"
	}

	emailData := struct {
		OTP     string
		BaseURL string
	}{
		OTP:     otp,
		BaseURL: baseURL,
	}

	err = platform.SendEmail(user.Email, "Password Reset OTP", "otp.html", emailData)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Gagal mengirim email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Jika email terdaftar, OTP akan dikirimkan ke email Anda."})
}

type VerifyOTPRequest struct {
	Email string `json:"email" binding:"required,email"`
	OTP   string `json:"otp" binding:"required"`
}

func (h *Handler) VerifyOTP(c *gin.Context) {
	var req VerifyOTPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.OTP = strings.TrimSpace(req.OTP)

	if err := h.repo.VerifyOTP(req.Email, req.OTP); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "OTP verified"})
}

type ResetPasswordRequest struct {
	Email       string `json:"email" binding:"required,email"`
	OTP         string `json:"otp" binding:"required"`
	NewPassword string `json:"new_password" binding:"required"`
}

func (h *Handler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	req.Email = strings.TrimSpace(req.Email)
	req.OTP = strings.TrimSpace(req.OTP)

	if err := h.repo.VerifyOTP(req.Email, req.OTP); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired OTP"})
		return
	}

	user, err := h.repo.FindUserByEmailOrNRP(req.Email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	user.Password = string(hashedPassword)

	if err := h.repo.UpdateUser(user); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	h.repo.DeleteOTP(req.Email)
	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}
