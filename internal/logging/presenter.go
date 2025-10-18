package logging

import (
	"fmt"
)

// PresentError formats an error for user display with masking.
func PresentError(context string, err error) string {
	if err == nil {
		return ""
	}
	return fmt.Sprintf("%s: %s", context, Mask(err.Error()))
}
