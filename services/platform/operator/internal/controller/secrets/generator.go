package secrets

import (
	"fmt"

	"github.com/sethvargo/go-password/password"
)

func Generate(passwordLength int, disableSpecialCharacters bool) ([]byte, error) {
	if passwordLength < 15 {
		return nil, fmt.Errorf("password length %d is too short: must be greater than 15", passwordLength)
	}

	if disableSpecialCharacters && passwordLength < 23 {
		return nil, fmt.Errorf("password length of %d is too short: must be greater than 23 when special characters are disabled", passwordLength)
	}

	numDigits := passwordLength / 2
	numSymbols := 0
	if !disableSpecialCharacters {
		numDigits = passwordLength / 3
		numSymbols = passwordLength / 3
	}

	p, err := password.Generate(passwordLength, numDigits, numSymbols, false, true)
	return []byte(p), err
}
