package restaurant

import (
	"fmt"

	"github.com/google/uuid"

	"restaurant-service/internal/domain/topping"
	apperr "restaurant-service/internal/shared/errors"
)

func ValidateToppingSelections(
	toppingIDs []uuid.UUID,
	toppingByID map[uuid.UUID]topping.Topping,
) error {
	for _, toppingID := range toppingIDs {
		if _, ok := toppingByID[toppingID]; !ok {
			return fmt.Errorf(
				"topping %s does not exist: %w",
				toppingID,
				apperr.ErrNotFound,
			)
		}
	}

	return nil
}
