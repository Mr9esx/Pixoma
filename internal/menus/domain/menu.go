package domain

import "errors"

// ErrValidation marks invalid menu/card configurations.
var ErrValidation = errors.New("menucard validation failed")

func hasHTTPPrefix(u string) bool {
	if len(u) >= 7 && u[:7] == "http://" {
		return true
	}
	return len(u) >= 8 && u[:8] == "https://"
}
