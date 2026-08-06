package restaurant

type ChecklistItem string

const (
	ChecklistBasic        ChecklistItem = "basic"
	ChecklistContact      ChecklistItem = "contact"
	ChecklistAddress      ChecklistItem = "address"
	ChecklistDelivery     ChecklistItem = "delivery"
	ChecklistPayment      ChecklistItem = "payment"
	ChecklistOpeningHours ChecklistItem = "openinghours"
)

type Checklist map[ChecklistItem]bool

func NewChecklist() Checklist {
	return Checklist{
		ChecklistBasic:        false,
		ChecklistContact:      false,
		ChecklistAddress:      false,
		ChecklistDelivery:     false,
		ChecklistPayment:      false,
		ChecklistOpeningHours: false,
	}
}

func (c Checklist) Complete(item ChecklistItem) {
	c[item] = true
}

func (c Checklist) Reopen(item ChecklistItem) {
	c[item] = false
}

func (c Checklist) IsCompleted() bool {
	required := []ChecklistItem{
		ChecklistBasic,
		ChecklistContact,
		ChecklistAddress,
		ChecklistDelivery,
		ChecklistPayment,
		ChecklistOpeningHours,
	}

	for _, item := range required {
		if !c[item] {
			return false
		}
	}

	return true
}
