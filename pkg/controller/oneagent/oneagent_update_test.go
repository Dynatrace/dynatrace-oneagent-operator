package oneagent

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewerVersion(t *testing.T) {
	assert.True(t, isDesiredNewer("1.200.1.12345", "1.201.1.12345", consoleLogger))
	assert.True(t, isDesiredNewer("1.200.1.12345", "2.200.1.12345", consoleLogger))
	assert.True(t, isDesiredNewer("1.200.1.12345", "1.200.2.12345", consoleLogger))
	assert.True(t, isDesiredNewer("1.200.1.12345", "1.200.1.123456", consoleLogger))
}

func TestBackportVersion(t *testing.T) {
	assert.False(t, isDesiredNewer("1.202.1.12345", "1.201.1.12345", consoleLogger))
	assert.False(t, isDesiredNewer("1.201.2.12345", "1.201.1.12345", consoleLogger))
	assert.False(t, isDesiredNewer("1.201.1.12345", "1.201.1.12344", consoleLogger))
	assert.False(t, isDesiredNewer("2.201.1.12345", "1.201.1.12345", consoleLogger))
}

func TestSameVersion(t *testing.T) {
	assert.False(t, isDesiredNewer("1.202.1.12345", "1.202.1.12345", consoleLogger))
	assert.False(t, isDesiredNewer("2.202.1.12345", "2.202.1.12345", consoleLogger))
	assert.False(t, isDesiredNewer("1.202.2.12345", "1.202.2.12345", consoleLogger))
	assert.False(t, isDesiredNewer("1.202.1.1", "1.202.1.1", consoleLogger))
}
