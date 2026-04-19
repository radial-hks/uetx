package build

import (
	"github.com/radial/uetx/internal/domain"
)

// NeedsBreakOut returns true when a BreakOut node is required.
func NeedsBreakOut(ot domain.OutputType, routing []string) bool {
	if ot != domain.CMOTFloat4 {
		return false
	}
	for _, slot := range routing {
		if _, ok := domain.ScalarSlots[slot]; ok {
			return true
		}
	}
	return false
}
