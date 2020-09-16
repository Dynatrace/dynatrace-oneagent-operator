package version

import (
	"github.com/stretchr/testify/assert"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"testing"
)

func TestIsAgentVersionSupported(t *testing.T) {
	logger := logf.ZapLoggerTo(os.Stdout, true)

	isSupported := IsAgentVersionSupported(logger, "2.0.0")
	assert.True(t, isSupported)

	isSupported = IsAgentVersionSupported(logger, "1.203.0")
	assert.True(t, isSupported)

	isSupported = IsAgentVersionSupported(logger, "0.0.0")
	assert.False(t, isSupported)

	isSupported = IsAgentVersionSupported(logger, "1.197.200")
	assert.False(t, isSupported)

	isSupported = IsAgentVersionSupported(logger, "")
	assert.True(t, isSupported)
}

func TestIsSupportedAgentVersion(t *testing.T) {
	t.Run("IsSupportedAgentVersion", func(t *testing.T) {
		a := &versionInfo{
			major:   2,
			minor:   0,
			release: 0,
		}
		isSupported, err := IsSupportedAgentVersion(a)
		assert.NoError(t, err)
		assert.True(t, isSupported)

		a = &versionInfo{
			major:   0,
			minor:   0,
			release: 0,
		}
		isSupported, err = IsSupportedAgentVersion(a)
		assert.NoError(t, err)
		assert.False(t, isSupported)
	})

	t.Run("IsSupportedAgentVersion parameter is nil", func(t *testing.T) {
		isSupported, err := IsSupportedAgentVersion(nil)
		assert.Error(t, err)
		assert.False(t, isSupported)
	})
}
