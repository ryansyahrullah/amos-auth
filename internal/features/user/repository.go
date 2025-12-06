package user

import (
	"gorm.io/gorm"
)

type Repository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) CreateUser(u *User) error {
	return r.db.Create(u).Error
}

func (r *Repository) FindAllUsers() ([]User, error) {
	var users []User
	err := r.db.Preload("Role").Find(&users).Error
	return users, err
}

func (r *Repository) FindUserByID(id uint) (*User, error) {
	var u User
	err := r.db.Preload("Role").Preload("Role.Permissions").First(&u, id).Error
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func (r *Repository) UpdateUser(u *User) error {
	return r.db.Model(u).Updates(map[string]interface{}{
		"email":          u.Email,
		"nrp":            u.NRP,
		"password":       u.Password,
		"plain_password": u.PlainPassword,
		"role_id":        u.RoleID,
	}).Error
}

func (r *Repository) DeleteUser(id uint) error {
	return r.db.Delete(&User{}, id).Error
}

func (r *Repository) FindRoleByName(name string) (*Role, error) {
	var role Role
	err := r.db.Preload("Permissions").Where("name = ?", name).First(&role).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *Repository) CreatePermission(p *Permission) error {
	return r.db.Create(p).Error
}

func (r *Repository) GetAllPermissions() ([]Permission, error) {
	var permissions []Permission
	err := r.db.Find(&permissions).Error
	return permissions, err
}

func (r *Repository) AddPermissionToRole(roleID uint, permissionID uint) error {
	var role Role
	if err := r.db.First(&role, roleID).Error; err != nil {
		return err
	}
	var permission Permission
	if err := r.db.First(&permission, permissionID).Error; err != nil {
		return err
	}
	return r.db.Model(&role).Association("Permissions").Append(&permission)
}

func (r *Repository) GetRolePermissions(roleID uint) ([]Permission, error) {
	var role Role
	if err := r.db.Preload("Permissions").First(&role, roleID).Error; err != nil {
		return nil, err
	}
	return role.Permissions, nil
}

// --- Role CRUD ---

func (r *Repository) CreateRole(role *Role) error {
	return r.db.Create(role).Error
}

func (r *Repository) FindAllRoles() ([]Role, error) {
	var roles []Role
	err := r.db.Preload("Permissions").Find(&roles).Error
	return roles, err
}

func (r *Repository) FindRoleByID(id uint) (*Role, error) {
	var role Role
	err := r.db.Preload("Permissions").First(&role, id).Error
	if err != nil {
		return nil, err
	}
	return &role, nil
}

func (r *Repository) UpdateRole(role *Role) error {
	return r.db.Save(role).Error
}

func (r *Repository) DeleteRole(id uint) error {
	return r.db.Delete(&Role{}, id).Error
}

// --- Permission CRUD ---

func (r *Repository) FindPermissionByID(id uint) (*Permission, error) {
	var p Permission
	err := r.db.First(&p, id).Error
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func (r *Repository) UpdatePermission(p *Permission) error {
	return r.db.Save(p).Error
}

func (r *Repository) DeletePermission(id uint) error {
	return r.db.Delete(&Permission{}, id).Error
}

func (r *Repository) RemovePermissionFromRole(roleID uint, permissionID uint) error {
	var role Role
	if err := r.db.First(&role, roleID).Error; err != nil {
		return err
	}
	var permission Permission
	if err := r.db.First(&permission, permissionID).Error; err != nil {
		return err
	}
	return r.db.Model(&role).Association("Permissions").Delete(&permission)
}
