package user

import (
	"time"

	"github.com/google/uuid"
)

const (
	DefaultStatus = "active"

	RoleCustomer = "customer"
	RoleOwner    = "owner"
)

type User struct {
	ID        uuid.UUID  `gorm:"type:uuid;primaryKey"`
	FirstName string     `gorm:"size:255;not null"`
	LastName  string     `gorm:"size:255;not null"`
	Email     string     `gorm:"size:255;unique;not null"`
	Password  string     `gorm:"not null"`
	Role      string     `gorm:"type:user_role_enum;default:'customer'"`
	Status    string     `gorm:"type:user_status_enum;default:'active'"`
	LoggedAt  *time.Time `gorm:"column:logged_at;type:timestamptz;default:null"`
	CreatedAt time.Time  `gorm:"type:timestamptz;autoCreateTime"`
	UpdatedAt *time.Time `gorm:"type:timestamptz;autoUpdateTime;default:null"`
}

func (User) TableName() string {
	return "users"
}
