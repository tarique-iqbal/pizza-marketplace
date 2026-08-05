package restaurant

type ChecklistItem string

const (
	ChecklistBasic        ChecklistItem = "basic"
	ChecklistContract     ChecklistItem = "contract"
	ChecklistAddress      ChecklistItem = "address"
	ChecklistDelivery     ChecklistItem = "delivery"
	ChecklistPayment      ChecklistItem = "payment"
	ChecklistOpeningHours ChecklistItem = "openinghours"
)

type Checklist map[ChecklistItem]bool

func NewChecklist() Checklist {
	return Checklist{
		ChecklistBasic:        false,
		ChecklistContract:     false,
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
		ChecklistContract,
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
