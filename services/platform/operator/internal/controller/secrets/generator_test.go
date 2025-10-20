package secrets

import (
	"strings"
	"testing"

	"github.com/sethvargo/go-password/password"
	"github.com/stretchr/testify/assert"
)

func TestLengths(t *testing.T) {
	// valid
	p, err := Generate(32, false)
	assert.NoError(t, err)
	assert.Len(t, p, 32)
	// valid - without symbols
	p, err = Generate(24, true)
	assert.NoError(t, err)
	assert.Len(t, p, 24)
	// valid - valid with symbols
	p, err = Generate(16, false)
	assert.NoError(t, err)
	assert.Len(t, p, 16)
	// invalid - without symbols
	p, err = Generate(16, true)
	assert.Error(t, err)
	assert.Len(t, p, 0)
	// invalid - with symbols
	p, err = Generate(12, true)
	assert.Error(t, err)
	assert.Len(t, p, 0)
}

func TestSpecialCharacters(t *testing.T) {
	p, err := Generate(24, false)
	assert.NoError(t, err)
	assert.Len(t, p, 24)
	assert.True(t, strings.ContainsAny(string(p), password.Symbols))
	p, err = Generate(24, true)
	assert.NoError(t, err)
	assert.Len(t, p, 24)
	assert.False(t, strings.ContainsAny(string(p), password.Symbols))
}
