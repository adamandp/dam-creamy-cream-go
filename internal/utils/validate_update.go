package utils

import (
	"errors"
	"fmt"
	"strings"
)

func ValidateUpdate(changed int, unchanged []string) error {
	if changed == 0 {
		return errors.New("📭 No changes detected. Everything is already up to date! 👍")
	}

	if len(unchanged) > 0 {
		return fmt.Errorf("📭 No changes detected in %s fields. Everything’s already up to date!", strings.Join(unchanged, ", "))
	}

	return nil
}
