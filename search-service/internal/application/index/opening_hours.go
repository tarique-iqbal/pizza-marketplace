package index

import "search-service/internal/domain/index"

// wireOpeningHours mirrors restaurant-service's OpeningHours wire shape
// (day-keyed, JSON tags lowercase weekday names).
type wireDayRange struct {
	Open  string `json:"open"`
	Close string `json:"close"`
}

type wireOpeningHours struct {
	Monday    []wireDayRange `json:"monday"`
	Tuesday   []wireDayRange `json:"tuesday"`
	Wednesday []wireDayRange `json:"wednesday"`
	Thursday  []wireDayRange `json:"thursday"`
	Friday    []wireDayRange `json:"friday"`
	Saturday  []wireDayRange `json:"saturday"`
	Sunday    []wireDayRange `json:"sunday"`
}

// flattenOpeningHours converts the day-keyed wire shape into one entry per
// range across the whole week — lets the open-now Painless filter check any
// weekday without picking a field name dynamically.
func flattenOpeningHours(w wireOpeningHours) []index.IndexedOpeningHours {
	days := []struct {
		weekday string
		ranges  []wireDayRange
	}{
		{"monday", w.Monday},
		{"tuesday", w.Tuesday},
		{"wednesday", w.Wednesday},
		{"thursday", w.Thursday},
		{"friday", w.Friday},
		{"saturday", w.Saturday},
		{"sunday", w.Sunday},
	}

	out := make([]index.IndexedOpeningHours, 0)
	for _, d := range days {
		for _, r := range d.ranges {
			out = append(out, index.IndexedOpeningHours{Weekday: d.weekday, Open: r.Open, Close: r.Close})
		}
	}

	return out
}
