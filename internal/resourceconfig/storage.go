package config

import (
	"regexp"
	"strconv"
)

var sizeRE = regexp.MustCompile(`^[1-9][0-9]*(Ki|Mi|Gi|Ti)$`)

// ValidSize accepts quantities whose byte count fits both runtime backends.
func ValidSize(value string) bool {
	// Reject malformed quantities before slicing the unit suffix.
	if !sizeRE.MatchString(value) {
		return false
	}
	n, err := strconv.ParseUint(value[:len(value)-2], 10, 64)
	shift := map[string]uint{"Ki": 10, "Mi": 20, "Gi": 30, "Ti": 40}[value[len(value)-2:]]
	return err == nil && n <= (^uint64(0)>>1)>>shift
}
