package config

import (
	"fmt"

	"github.com/Mr9esx/Pixoma/internal/platform/bootstrap"
)

// LoadVerifiedAgentToken reads the plaintext file and checks it against bootstrap hash.
func LoadVerifiedAgentToken(boot *bootstrap.Store, path string) (string, error) {
	tok, err := ReadAgentTokenFile(path)
	if err != nil {
		return "", err
	}
	ok, err := boot.VerifyAgentToken(tok)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", fmt.Errorf("agent token file %s does not match bootstrap hash", path)
	}
	return tok, nil
}
