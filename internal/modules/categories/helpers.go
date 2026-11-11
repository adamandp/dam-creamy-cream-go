package categories

func valueOrNil(v *string) any {
	if v == nil {
		return nil
	}
	return *v
}

// categoryRevisionKey intentionally has no TTL. An expired version counter
// could resurrect a previously cached page. Page values remain TTL-bound.
