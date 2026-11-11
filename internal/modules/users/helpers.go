package users

import (
	"time"
)

func bcryptInput(password string) []byte {
	raw := []byte(password)
	if len(raw) > 72 {
		return raw[:72]
	}
	return raw
}

func valueOrNil(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}

func dateOrNil(v *time.Time) any {
	if v == nil {
		return nil
	}
	return *v
}
