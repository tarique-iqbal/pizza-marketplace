package pizza

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
)

type PizzaStatus string

const (
	PizzaAvailable   PizzaStatus = "available"
	PizzaUnavailable PizzaStatus = "unavailable"
	PizzaArchived    PizzaStatus = "archived"
)

type Pizza struct {
	ID           uuid.UUID      `gorm:"type:uuid;primaryKey"`
	RestaurantID uuid.UUID      `gorm:"column:restaurant_id;type:uuid;not null;index"`
	Name         string         `gorm:"size:255;not null"`
	Image        *string        `gorm:"size:255"`
	IsVegetarian bool           `gorm:"not null;default:false"`
	Status       PizzaStatus    `gorm:"type:pizza_status_enum;not null;default:'available'"`
	SortOrder    int            `gorm:"not null;default:0;check:sort_order >= 0"`
	Toppings     datatypes.JSON `gorm:"type:jsonb;not null;default:'[]'"`
	CreatedAt    time.Time      `gorm:"type:timestamptz;autoCreateTime"`
	UpdatedAt    *time.Time     `gorm:"type:timestamptz;autoUpdateTime;default:null"`
}

func (Pizza) TableName() string {
	return "pizzas"
}

func NewPizza(id uuid.UUID, restaurantID uuid.UUID) *Pizza {
	return &Pizza{
		ID:           id,
		RestaurantID: restaurantID,
	}
}

func (p *Pizza) WithDetails(
	name string,
	image *string,
	isVegetarian bool,
	status PizzaStatus,
	sortOrder int,
) *Pizza {
	p.Name = name
	p.Image = image
	p.IsVegetarian = isVegetarian
	p.Status = status
	p.SortOrder = sortOrder
	return p
}

func (p *Pizza) ToppingIDs() ([]uuid.UUID, error) {
	if len(p.Toppings) == 0 {
		return nil, nil
	}

	var ids []uuid.UUID
	if err := json.Unmarshal(p.Toppings, &ids); err != nil {
		return nil, err
	}

	return ids, nil
}

func (p *Pizza) SetToppingIDs(ids []uuid.UUID) error {
	if ids == nil {
		ids = []uuid.UUID{}
	}

	seen := make(map[uuid.UUID]bool, len(ids))
	for _, id := range ids {
		if seen[id] {
			return fmt.Errorf("topping %s: %w", id, ErrDuplicateTopping)
		}
		seen[id] = true
	}

	data, err := json.Marshal(ids)
	if err != nil {
		return err
	}

	p.Toppings = data

	return nil
}
