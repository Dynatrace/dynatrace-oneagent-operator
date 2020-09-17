package version

import (
	"github.com/stretchr/testify/assert"
	"os"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"testing"
)

func TestIsRemoteClusterVersionSupported(t *testing.T) {
	logger := logf.ZapLoggerTo(os.Stdout, true)

	t.Run("IsRemoteClusterVersionSupported", func(t *testing.T) {
		isSupported := IsRemoteClusterVersionSupported(logger, "1.203.0")
		assert.True(t, isSupported)
	})
	t.Run("IsRemoteClusterVersionSupported unsupported version", func(t *testing.T) {
		isSupported := IsRemoteClusterVersionSupported(logger, "0.000.0")
		assert.False(t, isSupported)
	})
	t.Run("IsRemoteClusterVersionSupported dtclient is nil", func(t *testing.T) {
		isSupported := IsRemoteClusterVersionSupported(logger, "")
		assert.False(t, isSupported)
	})
}

func TestIsSupportedClusterVersion(t *testing.T) {
	t.Run("IsSupportedClusterVersion", func(t *testing.T) {
		a := versionInfo{
			major:   2,
			minor:   0,
			release: 0,
		}
		isSupported := isSupportedClusterVersion(a)
		assert.True(t, isSupported)

		a = minSupportedClusterVersion
		isSupported = isSupportedClusterVersion(a)
		assert.True(t, isSupported)

		a = versionInfo{
			major:   1,
			minor:   196,
			release: 10000,
		}
		isSupported = isSupportedClusterVersion(a)
		assert.False(t, isSupported)
	})
}
