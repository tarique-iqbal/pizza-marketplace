package restaurant

type DayRange struct {
	Open  string `json:"open"`
	Close string `json:"close"`
}

type OpeningHours struct {
	Monday    []DayRange `json:"monday"`
	Tuesday   []DayRange `json:"tuesday"`
	Wednesday []DayRange `json:"wednesday"`
	Thursday  []DayRange `json:"thursday"`
	Friday    []DayRange `json:"friday"`
	Saturday  []DayRange `json:"saturday"`
	Sunday    []DayRange `json:"sunday"`
}
