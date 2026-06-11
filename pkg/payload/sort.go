package payload

import (
	"sort"

	"github.com/SaaSAILabs/git-attest/pkg/active"
)

// SortEvents sorts a slice of FlightEvents chronologically by timestamp.
func SortEvents(events []active.FlightEvent) {
	sort.Slice(events, func(i, j int) bool {
		return events[i].Timestamp < events[j].Timestamp
	})
}
