package restaurant

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"restaurant-service/internal/domain/restaurant"
)

func TestNewChecklist(t *testing.T) {
	checklist := restaurant.NewChecklist()

	assert.False(t, checklist[restaurant.ChecklistBasic])
	assert.False(t, checklist[restaurant.ChecklistContract])
	assert.False(t, checklist[restaurant.ChecklistAddress])
	assert.False(t, checklist[restaurant.ChecklistDelivery])
	assert.False(t, checklist[restaurant.ChecklistPayment])
	assert.False(t, checklist[restaurant.ChecklistOpeningHours])
}

func TestChecklist_Complete(t *testing.T) {
	checklist := restaurant.NewChecklist()

	checklist.Complete(restaurant.ChecklistBasic)

	assert.True(t, checklist[restaurant.ChecklistBasic])
}

func TestChecklist_Reopen(t *testing.T) {
	checklist := restaurant.NewChecklist()

	checklist.Complete(restaurant.ChecklistBasic)

	assert.True(t, checklist[restaurant.ChecklistBasic])

	checklist.Reopen(restaurant.ChecklistBasic)

	assert.False(t, checklist[restaurant.ChecklistBasic])
}

func TestChecklist_IsCompleted_ReturnsFalse(t *testing.T) {
	checklist := restaurant.NewChecklist()

	checklist.Complete(restaurant.ChecklistBasic)

	assert.False(t, checklist.IsCompleted())
}

func TestChecklist_IsCompleted_ReturnsTrue(t *testing.T) {
	checklist := restaurant.NewChecklist()

	items := []restaurant.ChecklistItem{
		restaurant.ChecklistBasic,
		restaurant.ChecklistContract,
		restaurant.ChecklistAddress,
		restaurant.ChecklistDelivery,
		restaurant.ChecklistPayment,
		restaurant.ChecklistOpeningHours,
	}

	for _, item := range items {
		checklist.Complete(item)
	}

	assert.True(t, checklist.IsCompleted())
}
