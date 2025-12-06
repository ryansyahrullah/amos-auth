package auth

import (
	"amk-api-go/configs"
	"amk-api-go/internal/features/user"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) FindUserByEmailOrNRP(identifier string) (*user.User, error) {
	var u user.User
	// Check if identifier is email or NRP
	err := r.db.Preload("Role.Permissions").Preload("Role").Where("email = ? OR nrp = ?", identifier, identifier).First(&u).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) CreateUser(u *user.User) error {
	return r.db.Create(u).Error
}

func (r *Repository) UpdateUser(u *user.User) error {
	return r.db.Save(u).Error
}

func (r *Repository) SaveOTP(email string, otp string) error {
	var existingOTP PasswordReset
	err := r.db.Where("email = ?", email).First(&existingOTP).Error
	if err == nil {
		// Check cooldown (90 seconds)
		if time.Since(existingOTP.CreatedAt) < 90*time.Second {
			return errors.New("please wait 90 seconds before requesting a new OTP")
		}
		// Update existing
		existingOTP.Token = otp
		existingOTP.ExpiresAt = time.Now().Add(10 * time.Minute)
		existingOTP.CreatedAt = time.Now() // Reset CreatedAt for cooldown
		return r.db.Save(&existingOTP).Error
	}

	// Create new
	newOTP := PasswordReset{
		Email:     email,
		Token:     otp,
		ExpiresAt: time.Now().Add(10 * time.Minute),
		CreatedAt: time.Now(),
	}
	return r.db.Create(&newOTP).Error
}

func (r *Repository) VerifyOTP(email string, otp string) error {
	var record PasswordReset
	err := r.db.Where("email = ? AND token = ?", email, otp).First(&record).Error
	if err != nil {
		return errors.New("invalid OTP")
	}

	if time.Now().After(record.ExpiresAt) {
		return errors.New("OTP expired")
	}
	return nil
}

func (r *Repository) DeleteOTP(email string) error {
	return r.db.Where("email = ?", email).Delete(&PasswordReset{}).Error
}

// Helper to seed roles if not exist
func SeedRoles() {
	roles := map[string]string{
		"super_admin":  "dashboard,users,roles,pegawai,master_data,pengumuman,slip_gaji_input,slip_gaji_list,mcu",
		"direktur":     "dashboard,pengumuman",
		"admin_hcgs":   "dashboard,users,pegawai,master_data,pengumuman,slip_gaji_list,mcu",
		"admin_fat":    "dashboard,slip_gaji_input,slip_gaji_list",
		"pegawai":      "dashboard,pengumuman,mcu",
		"admin_she":    "dashboard,pengumuman,mcu",
		"admin_klinik": "dashboard,pengumuman,mcu",
	}

	for roleName, menus := range roles {
		var role user.Role
		if err := configs.DB.Where("name = ?", roleName).First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				configs.DB.Create(&user.Role{Name: roleName, Menus: menus})
			}
		} else {
			// Update menus if they are empty or different (optional, but good for dev)
			if role.Menus != menus {
				role.Menus = menus
				configs.DB.Save(&role)
			}
		}
	}
}

func SeedDefaultAdmin() {
	fmt.Println("Seeding Default Admin...")
	var adminRole user.Role
	if err := configs.DB.Where("name = ?", "super_admin").First(&adminRole).Error; err != nil {
		fmt.Printf("Error finding super_admin role: %v\n", err)
		return // Should not happen if SeedRoles is called first
	}

	var u user.User
	if err := configs.DB.Where("email = ?", "syahrullahryan@gmail.com").First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			fmt.Println("Creating default admin user...")
			hashedPassword, _ := bcrypt.GenerateFromPassword([]byte("@Ryan0852"), bcrypt.DefaultCost)
			admin := user.User{
				Email:    "syahrullahryan@gmail.com",
				Password: string(hashedPassword),
				RoleID:   &adminRole.ID,
				NRP:      "ADMIN001", // Default NRP
			}
			if err := configs.DB.Create(&admin).Error; err != nil {
				fmt.Printf("Error creating admin: %v\n", err)
			} else {
				fmt.Println("Default admin created successfully")
			}
		} else {
			fmt.Printf("Error finding admin user: %v\n", err)
		}
	} else {
		fmt.Println("Default admin already exists")
	}
}

// --- Refresh Token Methods ---

func (r *Repository) SaveRefreshToken(userID uint, token string, expiresAt time.Time) error {
	rt := user.RefreshToken{
		UserID:    userID,
		Token:     token,
		ExpiresAt: expiresAt,
	}
	return r.db.Create(&rt).Error
}

func (r *Repository) FindRefreshToken(token string) (*user.RefreshToken, error) {
	var rt user.RefreshToken
	err := r.db.Preload("User").Preload("User.Role").Preload("User.Role.Permissions").Where("token = ? AND revoked = ?", token, false).First(&rt).Error
	if err != nil {
		return nil, err
	}
	if time.Now().After(rt.ExpiresAt) {
		return nil, errors.New("refresh token expired")
	}
	return &rt, nil
}

func (r *Repository) RevokeRefreshToken(token string) error {
	return r.db.Model(&user.RefreshToken{}).Where("token = ?", token).Update("revoked", true).Error
}

func (r *Repository) RevokeAllUserTokens(userID uint) error {
	return r.db.Model(&user.RefreshToken{}).Where("user_id = ?", userID).Update("revoked", true).Error
}
