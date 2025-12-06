package user

import (
	"errors"

	"gorm.io/gorm"
)

func SeedPermissions(db *gorm.DB) {
	permissions := map[string]string{
		// User Management
		"user.create": "Membuat pengguna baru",
		"user.read":   "Melihat daftar pengguna",
		"user.update": "Mengubah data pengguna",
		"user.delete": "Menghapus pengguna",
		// Role Management
		"role.create": "Membuat peran baru",
		"role.read":   "Melihat daftar peran",
		"role.update": "Mengubah peran dan izin",
		"role.delete": "Menghapus peran",
		// Permission Management
		"permission.manage": "Mengelola izin (Super Admin)",

		// Employee Data (Admin)
		"pegawai.create":      "Membuat data pegawai",
		"pegawai.read":        "Melihat daftar pegawai",
		"pegawai.bpjs_upload": "Upload file BPJS Pegawai",

		// Slip Gaji (Admin)
		"slipgaji.upload":   "Upload slip gaji",
		"slipgaji.read_all": "Melihat semua slip gaji",
		"slipgaji.delete":   "Menghapus slip gaji",

		// Employee Self Access (Pegawai)
		"pegawai.read_own":  "Melihat data diri & BPJS sendiri",
		"slipgaji.read_own": "Melihat slip gaji sendiri",
	}

	var definedPermissions []string
	for permName, desc := range permissions {
		definedPermissions = append(definedPermissions, permName)
		var p Permission
		if err := db.Where("name = ?", permName).First(&p).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				db.Create(&Permission{Name: permName, Description: desc})
			}
		} else {
			// Update description if missing or changed
			if p.Description != desc {
				p.Description = desc
				db.Save(&p)
			}
		}
	}

	// Cleanup: Delete permissions not in the list
	if len(definedPermissions) > 0 {
		db.Where("name NOT IN ?", definedPermissions).Delete(&Permission{})
	}

	// Assign all permissions to super_admin
	var allPerms []string
	for p := range permissions {
		allPerms = append(allPerms, p)
	}
	assignPermissionsToRole(db, "super_admin", allPerms)

	// Assign permissions to pegawai
	pegawaiPermissions := []string{
		"pegawai.read_own",
		"slipgaji.read_own",
	}
	assignPermissionsToRole(db, "pegawai", pegawaiPermissions)

	// Assign permissions to admin_hcgs
	adminHcgsPermissions := []string{
		"user.read",
		"user.create",
		"user.update",
		"pegawai.create",
		"pegawai.read",
		"pegawai.bpjs_upload",
		"slipgaji.upload",
		"slipgaji.read_all",
		"slipgaji.delete",
	}
	assignPermissionsToRole(db, "admin_hcgs", adminHcgsPermissions)
}

func assignPermissionsToRole(db *gorm.DB, roleName string, permNames []string) {
	var role Role
	if err := db.Where("name = ?", roleName).First(&role).Error; err == nil {
		var perms []Permission
		db.Where("name IN ?", permNames).Find(&perms)
		db.Model(&role).Association("Permissions").Replace(perms)
	}
}

// Helper to seed roles if not exist
func SeedRoles(db *gorm.DB) {
	roles := map[string]string{
		"super_admin": "dashboard,users,roles,pegawai,slip_gaji_list",
		"admin_hcgs":  "dashboard,users,pegawai,slip_gaji_list",
		"pegawai":     "dashboard", // Frontend handles specific "My Data" links
	}

	for roleName, menus := range roles {
		var role Role
		if err := db.Where("name = ?", roleName).First(&role).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				db.Create(&Role{Name: roleName, Menus: menus})
			}
		} else {
			if role.Menus != menus {
				role.Menus = menus
				db.Save(&role)
			}
		}
	}
}
