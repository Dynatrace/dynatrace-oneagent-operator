package oneagent

import (
	"github.com/stretchr/testify/assert"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"testing"
)

func TestNewerVersion(t *testing.T) {
	logger := logf.ZapLoggerTo(os.Stdout, true)
	assert.True(t, isDesiredNewer("1.200.1.12345", "1.201.1.12345", logger))
	assert.True(t, isDesiredNewer("1.200.1.12345", "2.200.1.12345", logger))
	assert.True(t, isDesiredNewer("1.200.1.12345", "1.200.2.12345", logger))
	assert.True(t, isDesiredNewer("1.200.1.12345", "1.200.1.123456", logger))
}

func TestBackportVersion(t *testing.T) {
	logger := logf.ZapLoggerTo(os.Stdout, true)
	assert.False(t, isDesiredNewer("1.202.1.12345", "1.201.1.12345", logger))
	assert.False(t, isDesiredNewer("1.201.2.12345", "1.201.1.12345", logger))
	assert.False(t, isDesiredNewer("1.201.1.12345", "1.201.1.12344", logger))
	assert.False(t, isDesiredNewer("2.201.1.12345", "1.201.1.12345", logger))
}

func TestSameVersion(t *testing.T) {
	logger := logf.ZapLoggerTo(os.Stdout, true)
	assert.False(t, isDesiredNewer("1.202.1.12345", "1.202.1.12345", logger))
	assert.False(t, isDesiredNewer("2.202.1.12345", "2.202.1.12345", logger))
	assert.False(t, isDesiredNewer("1.202.2.12345", "1.202.2.12345", logger))
	assert.False(t, isDesiredNewer("1.202.1.1", "1.202.1.1", logger))
}
