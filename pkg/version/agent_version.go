package version

import (
	"fmt"
	"github.com/go-logr/logr"
)

// Pre-production, adapt accordingly once images are released
var minSupportedAgentVersion = &versionInfo{
	major:   1,
	minor:   198,
	release: 0,
}

func IsAgentVersionSupported(logger logr.Logger, versionString string) bool {
	if versionString == "" {
		// If version string is empty, latest agent image is used which is assumed to be supported
		return true
	}

	agentVersion, err := extractVersion(versionString)
	if err != nil {
		logger.Error(err, err.Error())
		return false
	}

	isSupported, err := IsSupportedAgentVersion(agentVersion)
	if err != nil {
		logger.Error(err, err.Error())
		return false
	}

	return isSupported
}

func IsSupportedAgentVersion(agentVersion *versionInfo) (bool, error) {
	if agentVersion == nil {
		err := fmt.Errorf("parameter must not be nil")
		return false, err
	}

	comparision, err := compareVersionInfo(agentVersion, minSupportedAgentVersion)
	return comparision >= 0, err
}
