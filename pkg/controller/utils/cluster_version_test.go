package utils

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestExtractVersion(t *testing.T) {
	t.Run("extractVersion", func(t *testing.T) {
		version, err := extractVersion("1.203.0.20200908-220956")
		assert.NoError(t, err)
		assert.NotNil(t, version)

		version, err = extractVersion("2.003.0.20200908-220956")
		assert.NoError(t, err)
		assert.NotNil(t, version)

		version, err = extractVersion("1.003.5.20200908-220956")
		assert.NoError(t, err)
		assert.NotNil(t, version)
	})
	t.Run("extractVersion fails on malformed version", func(t *testing.T) {
		version, err := extractVersion("1.203")
		assert.Error(t, err)
		assert.Nil(t, version)

		version, err = extractVersion("2.003.x.20200908-220956")
		assert.Error(t, err)
		assert.Nil(t, version)

		version, err = extractVersion("")
		assert.Error(t, err)
		assert.Nil(t, version)

		version, err = extractVersion("abc")
		assert.Error(t, err)
		assert.Nil(t, version)

		version, err = extractVersion("a.bcd.e")
		assert.Error(t, err)
		assert.Nil(t, version)
	})
}

func TestCompareClusterVersion(t *testing.T) {
	t.Run("compareClusterVersion a == b", func(t *testing.T) {
		a := &clusterVersionInfo{
			major:   1,
			minor:   200,
			release: 0,
		}
		b := &clusterVersionInfo{
			major:   1,
			minor:   200,
			release: 0,
		}
		comparison, err := compareClusterVersion(a, b)
		assert.NoError(t, err)
		assert.Equal(t, 0, comparison)
	})
	t.Run("compareClusterVersion a < b", func(t *testing.T) {
		a := &clusterVersionInfo{
			major:   1,
			minor:   0,
			release: 0,
		}
		b := &clusterVersionInfo{
			major:   1,
			minor:   200,
			release: 0,
		}
		comparison, err := compareClusterVersion(a, b)
		assert.NoError(t, err)
		assert.Less(t, comparison, 0)

		a = &clusterVersionInfo{
			major:   0,
			minor:   0,
			release: 0,
		}
		comparison, err = compareClusterVersion(a, b)
		assert.NoError(t, err)
		assert.Less(t, comparison, 0)

		a = &clusterVersionInfo{
			major:   0,
			minor:   2000,
			release: 3000,
		}
		comparison, err = compareClusterVersion(a, b)
		assert.NoError(t, err)
		assert.Less(t, comparison, 0)

		a = &clusterVersionInfo{
			major:   1,
			minor:   200,
			release: 0,
		}
		b = &clusterVersionInfo{
			major:   1,
			minor:   200,
			release: 1,
		}

		comparison, err = compareClusterVersion(a, b)
		assert.NoError(t, err)
		assert.Less(t, comparison, 0)

	})
	t.Run("compareClusterVersion a > b", func(t *testing.T) {
		a := &clusterVersionInfo{
			major:   1,
			minor:   200,
			release: 0,
		}
		b := &clusterVersionInfo{
			major:   1,
			minor:   100,
			release: 0,
		}
		comparison, err := compareClusterVersion(a, b)
		assert.NoError(t, err)
		assert.Greater(t, comparison, 0)

		a = &clusterVersionInfo{
			major:   2,
			minor:   0,
			release: 0,
		}
		comparison, err = compareClusterVersion(a, b)
		assert.NoError(t, err)
		assert.Greater(t, comparison, 0)

		a = &clusterVersionInfo{
			major:   1,
			minor:   201,
			release: 0,
		}
		comparison, err = compareClusterVersion(a, b)
		assert.NoError(t, err)
		assert.Greater(t, comparison, 0)

		a = &clusterVersionInfo{
			major:   1,
			minor:   0,
			release: 0,
		}
		b = &clusterVersionInfo{
			major:   0,
			minor:   0,
			release: 20000,
		}

		comparison, err = compareClusterVersion(a, b)
		assert.NoError(t, err)
		assert.Greater(t, comparison, 0)
	})
}

func TestIsSupportedClusterVersion(t *testing.T) {
	t.Run("isSupportedClusterVersion", func(t *testing.T) {
		a := &clusterVersionInfo{
			major:   2,
			minor:   0,
			release: 0,
		}
		isSupported, err := isSupportedClusterVersion(a)
		assert.NoError(t, err)
		assert.True(t, isSupported)

		a = minSupportedClusterVersion
		isSupported, err = isSupportedClusterVersion(a)
		assert.NoError(t, err)
		assert.True(t, isSupported)

		a = &clusterVersionInfo{
			major:   1,
			minor:   196,
			release: 10000,
		}
		isSupported, err = isSupportedClusterVersion(a)
		assert.NoError(t, err)
		assert.False(t, isSupported)
	})
}
