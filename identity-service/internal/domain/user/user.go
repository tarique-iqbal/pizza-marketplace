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
	ID        uuid.UUID     `gorm:"type:uuid;primaryKey"`
	FirstName string        `gorm:"size:255;not null"`
	LastName  string        `gorm:"size:255;not null"`
	Email     string        `gorm:"size:255;unique;not null"`
	Password  string        `gorm:"not null"`
	Role      string        `gorm:"type:user_role_enum;default:'customer'"`
	Status    string        `gorm:"type:user_status_enum;default:'active'"`
	LoggedAt  *time.Time    `gorm:"column:logged_at;type:timestamptz;default:null"`
	CreatedAt time.Time     `gorm:"type:timestamptz;autoCreateTime"`
	UpdatedAt *time.Time    `gorm:"type:timestamptz;autoUpdateTime;default:null"`
	events    []DomainEvent `gorm:"-"`
}

func (User) TableName() string {
	return "users"
}

func (u *User) MarkRegistered() {
	u.events = append(u.events, UserRegistered{
		UserID:     u.ID,
		Email:      u.Email,
		FirstName:  u.FirstName,
		LastName:   u.LastName,
		Role:       u.Role,
		OccurredAt: time.Now().UTC(),
	})
}

func (u *User) MarkRestaurantInitiated(restaurantID uuid.UUID, businessName, vatNumber string) {
	u.events = append(u.events, RestaurantInitiated{
		RestaurantID: restaurantID,
		OwnerID:      u.ID,
		BusinessName: businessName,
		VATNumber:    vatNumber,
		OccurredAt:   time.Now().UTC(),
	})
}

func (u *User) PullEvents() []DomainEvent {
	events := u.events
	u.events = nil
	return events
}
