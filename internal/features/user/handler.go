package user

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"amk-api-go/internal/core/platform"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
)

func uintPtr(n uint) *uint {
	return &n
}

type Handler struct {
	repo *Repository
}

func NewHandler(repo *Repository, _ interface{}) *Handler { // Keep signature compatible or change it? Main.go passes nil.
	return &Handler{repo: repo}
}

type CreateUserRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
	Role     string `json:"role" binding:"required"`
	NRP      string `json:"nrp" binding:"required"`
	FullName string `json:"full_name" binding:"required"`
	// HCGS fields ignored for now
	JabatanID           uint   `json:"jabatan_id"`
	DepartemenID        uint   `json:"departemen_id"`
	JobSiteID           uint   `json:"job_site_id"`
	KontrakID           uint   `json:"kontrak_id"`
	TanggalAwalKontrak  string `json:"tanggal_awal_kontrak"`
	TanggalAkhirKontrak string `json:"tanggal_akhir_kontrak"`
	TanggalBergabung    string `json:"tanggal_bergabung"`
}

func (h *Handler) CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check permissions
	currentUserRole := c.GetString("role")
	if currentUserRole != "super_admin" {
		if currentUserRole == "admin_hcgs" && req.Role != "pegawai" {
			c.JSON(http.StatusForbidden, gin.H{"error": "Admin HCGS can only create Pegawai"})
			return
		}
	}

	role, err := h.repo.FindRoleByName(req.Role)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	u := User{
		Email:         req.Email,
		Password:      string(hashedPassword),
		PlainPassword: req.Password,
		RoleID:        uintPtr(role.ID),
		NRP:           req.NRP,
		Name:          req.FullName,
	}

	if err := h.repo.CreateUser(&u); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": fmt.Sprintf("Failed to create user: %v", err)})
		return
	}

	// NOTE: HCGS Pegawai creation is no longer handled here.
	// It should be handled by calling the HCGS service directly.

	c.JSON(http.StatusCreated, gin.H{"message": "User created successfully", "user": u})
}

func (h *Handler) GetAllUsers(c *gin.Context) {
	users, err := h.repo.FindAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch users"})
		return
	}
	currentUserRole := c.GetString("role")
	if currentUserRole != "super_admin" {
		for i := range users {
			users[i].PlainPassword = ""
		}
	}
	c.JSON(http.StatusOK, users)
}

func (h *Handler) GetMe(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}
	uid, ok := userID.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user ID"})
		return
	}

	user, err := h.repo.FindUserByID(uid)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Prepare response with menus
	menus := []string{}
	if user.Role.Menus != "" {
		menus = strings.Split(user.Role.Menus, ",")
	}

	// Prepare permissions list
	permissions := []string{}
	for _, p := range user.Role.Permissions {
		permissions = append(permissions, p.Name)
	}

	c.JSON(http.StatusOK, gin.H{
		"id":           user.ID,
		"email":        user.Email,
		"nrp":          user.NRP,
		"nama_lengkap": user.Name, // Mapped to nama_lengkap for mobile app compatibility
		"role":         user.Role.Name,
		"menus":        menus,
		"permissions":  permissions,
	})
}

func (h *Handler) GetUserByID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	user, err := h.repo.FindUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}
	currentUserRole := c.GetString("role")
	if currentUserRole != "super_admin" {
		user.PlainPassword = ""
	}
	c.JSON(http.StatusOK, user)
}

func (h *Handler) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
		Role     string `json:"role"`
		NRP      string `json:"nrp"`
		Name     string `json:"name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	u, err := h.repo.FindUserByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Check permissions
	// Check permissions
	currentUserRole := c.GetString("role")
	if currentUserRole != "super_admin" {
		if currentUserRole == "admin_hcgs" {
			if u.Role.Name != "pegawai" {
				c.JSON(http.StatusForbidden, gin.H{"error": "Admin HCGS can only update Pegawai users"})
				return
			}
			if req.Role != "" && req.Role != "pegawai" {
				c.JSON(http.StatusForbidden, gin.H{"error": "Admin HCGS cannot change role to non-Pegawai"})
				return
			}
		}
	}

	if req.Email != "" {
		u.Email = req.Email
	}
	if req.NRP != "" {
		u.NRP = req.NRP
	}
	if req.Name != "" {
		u.Name = req.Name
	}
	if req.Password != "" {
		hashedPassword, _ := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
		u.Password = string(hashedPassword)
		u.PlainPassword = req.Password
	}
	if req.Role != "" {
		role, err := h.repo.FindRoleByName(req.Role)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role: " + req.Role})
			return
		}
		u.RoleID = uintPtr(role.ID)
		u.Role = Role{} // Clear preloaded role to ensure RoleID update takes effect
		log.Printf("[DEBUG] UpdateUser: Changing role to %s (ID: %d)\n", role.Name, role.ID)
	}
	log.Printf("[DEBUG] UpdateUser: Saving user with RoleID: %d\n", *u.RoleID)

	if err := h.repo.UpdateUser(u); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user"})
		return
	}
	c.JSON(http.StatusOK, u)
}

func (h *Handler) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}

	skipSync := c.Query("skip_sync") == "true"
	if !skipSync {
		// Call HCGS Service to delete linked pegawai
		hcgsURL := os.Getenv("HCGS_SERVICE_URL")
		if hcgsURL == "" {
			hcgsURL = "http://localhost:8081"
		}
		systemToken, _ := platform.GenerateToken(0, "super_admin", nil)

		client := &http.Client{Timeout: 5 * time.Second}
		req, _ := http.NewRequest("DELETE", fmt.Sprintf("%s/hcgs/pegawai/user/%d", hcgsURL, id), nil)
		req.Header.Set("Authorization", "Bearer "+systemToken)

		resp, err := client.Do(req)
		if err != nil {
			log.Printf("[ERROR] Failed to sync delete pegawai: %v\n", err)
		} else {
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				log.Printf("[ERROR] HCGS service returned %d during sync delete\n", resp.StatusCode)
			}
		}
	}

	if err := h.repo.DeleteUser(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete user"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "User deleted successfully"})
}

// --- Permission Handlers ---

func (h *Handler) CreatePermission(c *gin.Context) {
	var p Permission
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.CreatePermission(&p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create permission"})
		return
	}
	c.JSON(http.StatusCreated, p)
}

func (h *Handler) GetAllPermissions(c *gin.Context) {
	permissions, err := h.repo.GetAllPermissions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch permissions"})
		return
	}
	fmt.Printf("[DEBUG] GetAllPermissions: Found %d permissions\n", len(permissions))
	c.JSON(http.StatusOK, permissions)
}

type AssignPermissionRequest struct {
	RoleID       uint `json:"role_id" binding:"required"`
	PermissionID uint `json:"permission_id" binding:"required"`
}

func (h *Handler) AssignPermission(c *gin.Context) {
	var req AssignPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.AddPermissionToRole(req.RoleID, req.PermissionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to assign permission"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Permission assigned successfully"})
}

func (h *Handler) RevokePermission(c *gin.Context) {
	var req AssignPermissionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.repo.RemovePermissionFromRole(req.RoleID, req.PermissionID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to revoke permission"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Permission revoked successfully"})
}

func (h *Handler) UpdatePermission(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	var p Permission
	if err := c.ShouldBindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	p.ID = uint(id)
	if err := h.repo.UpdatePermission(&p); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update permission"})
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *Handler) DeletePermission(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	if err := h.repo.DeletePermission(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete permission"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Permission deleted successfully"})
}

// --- Role Handlers ---

func (h *Handler) CreateRole(c *gin.Context) {
	var r Role
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := h.repo.CreateRole(&r); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create role"})
		return
	}
	c.JSON(http.StatusCreated, r)
}

func (h *Handler) GetAllRoles(c *gin.Context) {
	roles, err := h.repo.FindAllRoles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch roles"})
		return
	}
	c.JSON(http.StatusOK, roles)
}

func (h *Handler) UpdateRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	var r Role
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	r.ID = uint(id)
	if err := h.repo.UpdateRole(&r); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update role"})
		return
	}
	c.JSON(http.StatusOK, r)
}

func (h *Handler) DeleteRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid ID"})
		return
	}
	if err := h.repo.DeleteRole(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete role"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Role deleted successfully"})
}

func (h *Handler) FixPermissions(c *gin.Context) {
	SeedPermissions(h.repo.db)
	c.JSON(http.StatusOK, gin.H{
		"message": "Permissions fixed successfully via SeedPermissions",
	})
}
